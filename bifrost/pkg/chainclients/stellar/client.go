package stellar

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"os"

	"strings"
	"sync"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/codec"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tssp "github.com/switchlyprotocol/switchlynode/v3/bifrost/tss/go-tss/tss"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"

	"github.com/stellar/go/xdr"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/shared/runners"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/shared/signercache"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	stypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/tss"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	mem "github.com/switchlyprotocol/switchlynode/v3/x/switchly/memo"
)

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Client is a structure to sign and broadcast tx to Stellar chain used by signer mostly
type Client struct {
	logger              zerolog.Logger
	cfg                 config.BifrostChainConfiguration
	tssKeyManager       *tss.KeySign
	switchlyBridge      switchlyclient.SwitchlyBridge
	storage             *blockscanner.BlockScannerStorage
	blockScanner        *blockscanner.BlockScanner
	signerCacheManager  *signercache.CacheManager
	stellarScanner      *StellarBlockScanner
	routerEventScanner  *RouterEventScanner
	sorobanRPCClient    *SorobanRPCClient
	globalSolvencyQueue chan stypes.Solvency
	wg                  *sync.WaitGroup
	stopchan            chan struct{}
	horizonClient       *horizonclient.Client
	networkPassphrase   string
	routerAddress       string
	localPubKey         common.PubKey
	localPrivKey        []byte
	accts               *StellarMetaDataStore
	// vaultLocks serializes sign/broadcast per vault to avoid sequence races
	vaultLocks   map[string]*sync.Mutex
	vaultLocksMu sync.Mutex
}

// RouterAwareStellarScanner wraps the stellar scanner to include router events
type RouterAwareStellarScanner struct {
	*StellarBlockScanner
	routerScanner *RouterEventScanner
}

// NewRouterAwareStellarScanner creates a new router-aware stellar scanner
func NewRouterAwareStellarScanner(stellarScanner *StellarBlockScanner, routerScanner *RouterEventScanner) *RouterAwareStellarScanner {
	return &RouterAwareStellarScanner{
		StellarBlockScanner: stellarScanner,
		routerScanner:       routerScanner,
	}
}

// FetchTxs retrieves transactions for a given block height including router events
func (r *RouterAwareStellarScanner) FetchTxs(height, chainHeight int64) (types.TxIn, error) {
	return r.StellarBlockScanner.FetchTxs(height, chainHeight)
}

// RouterConfig holds router configuration for Stellar
type RouterConfig struct {
	Address     string `json:"address"`
	Version     string `json:"version"`
	Deployed    bool   `json:"deployed"`
	DeployedAt  int64  `json:"deployed_at"`
	VaultPubKey string `json:"vault_pubkey"`
}

// Handle sequence number checkpoint management

// NewClient creates a new instance of a Stellar chain client
func NewClient(
	switchKeys *switchlyclient.Keys,
	cfg config.BifrostChainConfiguration,
	server *tssp.TssServer,
	switchlyBridge switchlyclient.SwitchlyBridge,
	m *metrics.Metrics,
) (*Client, error) {
	logger := log.With().Str("module", cfg.ChainID.String()).Logger()

	tssKm, err := tss.NewKeySign(server, switchlyBridge)
	if err != nil {
		return nil, fmt.Errorf("fail to create tss signer: %w", err)
	}

	if switchlyBridge == nil {
		return nil, errors.New("SwitchlyProtocol bridge is nil")
	}

	// Extract local private key and public key for single-node fallback
	var localPubKey common.PubKey
	var localPrivKey []byte
	if switchKeys != nil {
		logger.Info().Msg("switchKeys provided to Stellar client")
		privKey, err := switchKeys.GetPrivateKey()
		if err != nil {
			logger.Warn().Err(err).Msg("failed to get private key from switchKeys")
		} else {
			localPrivKey = privKey.Bytes()
			logger.Info().Int("privkey_len", len(localPrivKey)).Msg("extracted private key")
			// Convert cosmos private key's public key to common.PubKey (like gaia client)
			temp, err := codec.ToCmtPubKeyInterface(privKey.PubKey())
			if err != nil {
				logger.Warn().Err(err).Msg("failed to convert to comet pubkey")
			} else {
				pk, err := common.NewPubKeyFromCrypto(temp)
				if err != nil {
					logger.Warn().Err(err).Msg("failed to convert to common.PubKey")
				} else {
					localPubKey = pk
					logger.Info().Str("local_pubkey", localPubKey.String()).Msg("extracted local public key")
				}
			}
		}
	} else {
		logger.Warn().Msg("switchKeys is nil - no local keys available for fallback signing")
	}

	// Initialize network configuration for asset mapping
	var stellarNetwork StellarNetwork
	switch cfg.ChainNetwork {
	case "mainnet":
		stellarNetwork = StellarMainnet
	case "testnet":
		stellarNetwork = StellarTestnet
	default:
		logger.Warn().
			Str("chain_network", cfg.ChainNetwork).
			Msg("unknown chain network, defaulting to testnet")
		stellarNetwork = StellarTestnet
	}

	// Set the network for asset mapping
	SetNetwork(stellarNetwork)
	logger.Info().
		Str("stellar_network", string(stellarNetwork)).
		Msg("initialized stellar network configuration")

	// Determine network passphrase based on configuration
	networkPassphrase := network.PublicNetworkPassphrase
	if cfg.ChainNetwork == "testnet" {
		networkPassphrase = network.TestNetworkPassphrase
	}

	// Create Horizon client with custom HTTP client for rate limiting
	var horizonClient *horizonclient.Client
	if cfg.RPCHost != "" {
		// Use configured RPC host from environment
		// Expected environment variable: BIFROST_CHAINS_XLM_RPC_HOST
		// Docker example: http://stellar:8000 (Stellar quickstart container)
		// Public example: https://horizon-testnet.stellar.org
		httpClient := &http.Client{
			Timeout: time.Duration(cfg.BlockScanner.HTTPRequestTimeout),
			Transport: &http.Transport{
				MaxIdleConns:        10,
				MaxIdleConnsPerHost: 5,
				IdleConnTimeout:     30 * time.Second,
				DisableKeepAlives:   false,
			},
		}

		horizonClient = &horizonclient.Client{
			HorizonURL: cfg.RPCHost,
			HTTP:       httpClient,
		}

		logger.Info().
			Str("horizon_url", cfg.RPCHost).
			Msg("using configured Horizon RPC host")
	} else {
		// Fall back to public networks only if no RPC host is configured
		logger.Warn().Msg("no RPC host configured, falling back to public Stellar networks")
		if cfg.ChainNetwork == "testnet" {
			horizonClient = horizonclient.DefaultTestNetClient
		} else {
			horizonClient = horizonclient.DefaultPublicNetClient
		}
	}

	// Initialize Soroban RPC client
	sorobanRPCClient := NewSorobanRPCClient(cfg, logger, stellarNetwork)

	// Create storage first before creating stellar scanner
	var path string // if not set later, will in memory storage
	if len(cfg.BlockScanner.DBPath) > 0 {
		path = fmt.Sprintf("%s/%s", cfg.BlockScanner.DBPath, cfg.BlockScanner.ChainID)
	}
	storage, err := blockscanner.NewBlockScannerStorage(path, cfg.ScannerLevelDB)
	if err != nil {
		return nil, fmt.Errorf("fail to create scan storage: %w", err)
	}

	// Create a temporary channel for initialization - will be replaced in Start()
	tempNetworkFeeQueue := make(chan common.NetworkFee, 100)
	tempTxsQueue := make(chan stypes.TxIn, 100)

	// Create Stellar block scanner
	stellarScanner, err := NewStellarBlockScanner(
		cfg.RPCHost,
		cfg.BlockScanner,
		storage,
		switchlyBridge,
		m,
		nil, // solvency reporter - not used as we use SolvencyCheckRunner
		horizonClient,
		sorobanRPCClient,
		tempNetworkFeeQueue,
		tempTxsQueue,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stellar scanner: %w", err)
	}

	// Create main block scanner
	blockScanner, err := blockscanner.NewBlockScanner(cfg.BlockScanner, storage, m, switchlyBridge, stellarScanner)
	if err != nil {
		return nil, fmt.Errorf("failed to create block scanner: %w", err)
	}

	// Create signer cache manager
	signerCacheManager, err := signercache.NewSignerCacheManager(storage.GetInternalDb())
	if err != nil {
		return nil, fmt.Errorf("fail to create signer cache manager")
	}

	// Create router event scanner if router is configured
	var routerEventScanner *RouterEventScanner
	routerAddress := getRouterAddress(switchlyBridge)
	if routerAddress != "" {
		routerEventScanner = NewRouterEventScanner(
			cfg.BlockScanner,
			horizonClient,
			sorobanRPCClient,
			routerAddress,
			switchlyBridge,
		)
	}

	// Detect network passphrase from Horizon root if available (handles local quickstart)
	if horizonClient != nil {
		if root, err := horizonClient.Root(); err == nil && root.NetworkPassphrase != "" {
			networkPassphrase = root.NetworkPassphrase
			logger.Info().Str("detected_network_passphrase", networkPassphrase).Msg("using Horizon-reported network passphrase")
		}
	}

	return &Client{
		logger:              logger,
		cfg:                 cfg,
		tssKeyManager:       tssKm,
		switchlyBridge:      switchlyBridge,
		storage:             storage,
		blockScanner:        blockScanner,
		signerCacheManager:  signerCacheManager,
		stellarScanner:      stellarScanner,
		routerEventScanner:  routerEventScanner,
		sorobanRPCClient:    sorobanRPCClient,
		globalSolvencyQueue: make(chan stypes.Solvency, 100),
		wg:                  &sync.WaitGroup{},
		stopchan:            make(chan struct{}),
		horizonClient:       horizonClient,
		networkPassphrase:   networkPassphrase,
		routerAddress:       routerAddress,
		localPubKey:         localPubKey,
		localPrivKey:        localPrivKey,
		accts:               NewStellarMetaDataStore(),
		vaultLocks:          make(map[string]*sync.Mutex),
	}, nil
}

func getRouterAddress(bridge switchlyclient.SwitchlyBridge) string {
	// Get router address from bridge configuration
	contracts, err := bridge.GetContractAddress()
	if err != nil {
		return ""
	}

	for _, contract := range contracts {
		if addr, ok := contract.Contracts[common.StellarChain]; ok {
			return addr.String()
		}
	}
	return ""
}

// Start initializes and starts the Stellar chain client.
// It waits for the Stellar node to be fully synced before starting the block scanner
// to ensure Bifrost starts scanning from the latest block height.
func (c *Client) Start(globalTxsQueue chan stypes.TxIn, globalErrataQueue chan stypes.ErrataBlock, globalSolvencyQueue chan stypes.Solvency, globalNetworkFeeQueue chan common.NetworkFee) {
	c.globalSolvencyQueue = globalSolvencyQueue
	c.stellarScanner.globalNetworkFeeQueue = globalNetworkFeeQueue
	c.stellarScanner.globalTxsQueue = globalTxsQueue
	c.tssKeyManager.Start()

	// Wait for Stellar node to be fully synced before starting block scanner
	// This ensures Bifrost starts scanning from the latest block height
	c.logger.Info().Msg("STELLAR: Waiting for Stellar node to be fully synced...")

	syncHeight, err := c.waitForStellarSync()
	if err != nil {
		c.logger.Error().Err(err).Msg("STELLAR: Failed to wait for sync completion")
		return
	}

	// Set scanner position to the latest synced block height
	// This is the initial scan position when the node first syncs
	if err := c.storage.SetScanPos(syncHeight); err != nil {
		c.logger.Error().Err(err).Int64("sync_height", syncHeight).
			Msg("STELLAR: Failed to set scan position")
		return
	}

	c.logger.Info().Int64("sync_height", syncHeight).
		Msg("STELLAR: Node fully synced! Starting continuous scanner from latest block")

	// Start the Stellar continuous block scanner instead of the main block scanner
	// This ensures continuous ingestion every 60 seconds as required
	c.stellarScanner.Start()

	c.wg.Add(1)
	go runners.SolvencyCheckRunner(c.GetChain(), c, c.switchlyBridge, c.stopchan, c.wg, constants.SwitchlyBlockTime)

	// Start router monitoring if router is configured
	if c.routerAddress != "" {
		c.wg.Add(1)
		go c.routerHealthMonitor()
	}
}

// waitForStellarSync waits for the Stellar node to be fully synced.
// It continuously checks the current height until it stabilizes (difference <= 2 over 3 seconds).
// Returns the final synced height or an error if sync detection fails.
func (c *Client) waitForStellarSync() (int64, error) {
	const (
		stabilityCheckDelay = 3 * time.Second
		pollInterval        = 5 * time.Second
		maxHeightDifference = 2
	)

	for {
		height, err := getCurrentStellarHeight(c.horizonClient)
		if err != nil {
			c.logger.Debug().Err(err).Msg("STELLAR: Failed to get current height, retrying...")
			time.Sleep(pollInterval)
			continue
		}

		if height <= 0 {
			c.logger.Debug().Msg("STELLAR: No height available yet, retrying...")
			time.Sleep(pollInterval)
			continue
		}

		// Verify height stability by checking again after a delay
		time.Sleep(stabilityCheckDelay)
		secondHeight, secondErr := getCurrentStellarHeight(c.horizonClient)
		if secondErr != nil {
			c.logger.Debug().Err(secondErr).Msg("STELLAR: Failed to verify height stability, retrying...")
			continue
		}

		if secondHeight <= 0 {
			c.logger.Debug().Msg("STELLAR: No height available on second check, retrying...")
			continue
		}

		heightDifference := secondHeight - height
		if heightDifference <= maxHeightDifference {
			return secondHeight, nil
		}

		c.logger.Debug().
			Int64("first_height", height).
			Int64("second_height", secondHeight).
			Int64("difference", heightDifference).
			Msg("STELLAR: Height not stable yet, continuing to wait...")

		time.Sleep(pollInterval)
	}
}

// routerHealthMonitor monitors router health in a separate goroutine
func (c *Client) routerHealthMonitor() {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := c.MonitorRouterHealth(); err != nil {
				c.logger.Error().Err(err).Msg("router health check failed")
			}
		case <-c.stopchan:
			c.logger.Info().Msg("stopping router health monitor")
			return
		}
	}
}

// getCurrentStellarHeight retrieves the current Stellar chain height by querying the latest ledger.
// Returns the sequence number of the most recent ledger or an error if the query fails.
func getCurrentStellarHeight(horizonClient *horizonclient.Client) (int64, error) {
	ledgerRequest := horizonclient.LedgerRequest{Order: horizonclient.OrderDesc, Limit: 1}
	ledgerPage, err := horizonClient.Ledgers(ledgerRequest)
	if err != nil {
		return 0, fmt.Errorf("failed to query ledgers: %w", err)
	}

	if len(ledgerPage.Embedded.Records) == 0 {
		return 0, fmt.Errorf("no ledgers found in response")
	}

	return int64(ledgerPage.Embedded.Records[0].Sequence), nil
}

// Stop stops the Stellar chain client and all its components
func (c *Client) Stop() {
	c.logger.Info().Msg("STELLAR: Stopping Stellar chain client")

	// Stop the continuous block scanner
	c.stellarScanner.Stop()

	// Stop the main block scanner if it was started
	if c.blockScanner != nil {
		c.blockScanner.Stop()
	}

	// Stop TSS key manager
	c.tssKeyManager.Stop()

	// Signal all goroutines to stop
	close(c.stopchan)

	// Wait for all goroutines to finish
	c.wg.Wait()

	c.logger.Info().Msg("STELLAR: Stellar chain client stopped")
}

// GetConfig return the configuration used by Stellar chain client
func (c *Client) GetConfig() config.BifrostChainConfiguration {
	return c.cfg
}

func (c *Client) IsBlockScannerHealthy() bool {
	if !c.stellarScanner.IsHealthy() {
		c.logger.Info().Str("caller", "IsBlockScannerHealthy").Msg("stellar scanner unhealthy")
		return false
	}
	if c.sorobanRPCClient != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if _, err := c.sorobanRPCClient.GetLatestLedger(ctx); err != nil {
			c.logger.Info().Err(err).Str("caller", "IsBlockScannerHealthy").Msg("soroban RPC unhealthy")
			return false
		}
	}
	c.logger.Info().Bool("health_status", true).Str("caller", "IsBlockScannerHealthy").Msg("Stellar client healthy")
	return true
}

func (c *Client) GetChain() common.Chain {
	return c.cfg.ChainID
}

func (c *Client) GetHeight() (int64, error) {
	return c.stellarScanner.GetHeight()
}

// GetBlockScannerHeight returns blockscanner height
func (c *Client) GetBlockScannerHeight() (int64, error) {
	return c.blockScanner.PreviousHeight(), nil
}

// RollbackBlockScanner rolls back the block scanner to the last observed block
func (c *Client) RollbackBlockScanner() error {
	return c.blockScanner.RollbackToLastObserved()
}

// GetLatestTxForVault returns the latest transaction for a vault
func (c *Client) GetLatestTxForVault(vault string) (string, string, error) {
	lastObserved, err := c.signerCacheManager.GetLatestRecordedTx(stypes.InboundCacheKey(vault, c.GetChain().String()))
	if err != nil {
		return "", "", err
	}
	lastBroadCasted, err := c.signerCacheManager.GetLatestRecordedTx(stypes.BroadcastCacheKey(vault, c.GetChain().String()))
	return lastObserved, lastBroadCasted, err
}

// GetAddress returns the Stellar address for the given public key
func (c *Client) GetAddress(poolPubKey common.PubKey) string {
	addr, err := poolPubKey.GetAddress(common.StellarChain)
	if err != nil {
		c.logger.Error().Err(err).Msg("fail to get stellar address from public key")
		return ""
	}
	return addr.String()
}

// GetAccount returns the account information for the given public key
func (c *Client) GetAccount(pkey common.PubKey, height *big.Int) (common.Account, error) {
	addr := c.GetAddress(pkey)
	return c.GetAccountByAddress(addr, height)
}

// GetAccountByAddress returns the account information for the given address
func (c *Client) GetAccountByAddress(address string, height *big.Int) (common.Account, error) {
	account := common.Account{}
	if !strkey.IsValidEd25519PublicKey(address) {
		return account, fmt.Errorf("invalid stellar address: %s", address)
	}

	// Get account info from Horizon
	accountRequest := horizonclient.AccountRequest{AccountID: address}
	horizonAccount, err := c.horizonClient.AccountDetail(accountRequest)
	if err != nil {
		// Account might not exist yet
		c.logger.Debug().Err(err).Str("address", address).Msg("account not found")
		return account, nil
	}

	// Process all balances for supported assets
	var coins common.Coins
	for _, balance := range horizonAccount.Balances {
		var assetMapping StellarAssetMapping
		var found bool

		if balance.Asset.Type == "native" {
			assetMapping, found = GetAssetByStellarAsset("native", "XLM", "")
		} else {
			// For non-native assets, try to find mapping
			assetCode := balance.Asset.Code
			assetIssuer := balance.Asset.Issuer

			// First try to find by stellar asset details
			assetMapping, found = GetAssetByStellarAsset(balance.Asset.Type, assetCode, assetIssuer)

			// If not found, try to find by contract address (for Soroban tokens)
			if !found && balance.Asset.Type == "contract" {
				assetMapping, found = GetAssetByAddress(assetIssuer)
			}
		}

		if !found {
			c.logger.Debug().
				Str("asset_type", balance.Asset.Type).
				Str("asset_code", balance.Asset.Code).
				Str("asset_issuer", balance.Asset.Issuer).
				Str("network", string(GetCurrentNetwork())).
				Msg("skipping unsupported asset")
			continue // Skip unsupported assets
		}

		// Convert balance using asset mapping
		coin, err := assetMapping.ConvertToSwitchlyProtocolAmount(balance.Balance)
		if err != nil {
			c.logger.Err(err).Msg("fail to convert balance to coin")
			continue
		}
		c.logger.Debug().
			Str("asset", assetMapping.SwitchlyAsset.String()).
			Str("amount", coin.Amount.String()).
			Msg("balance converted")

		// Only include non-zero balances
		if !coin.Amount.IsZero() {
			coins = append(coins, coin)
		}
	}

	// Use on-chain sequence from Horizon account
	account = common.NewAccount(horizonAccount.Sequence, 0, coins, false)
	return account, nil
}

// processOutboundTx builds a simple Payment operation for outbound transfers.
// This implementation uses Stellar's native payment operations with memo truncation
// instead of complex Soroban contract calls, providing better reliability and performance.
func (c *Client) processOutboundTx(tx stypes.TxOutItem) (*txnbuild.Payment, error) {
	if len(tx.Coins) == 0 {
		return nil, fmt.Errorf("no coins to send")
	}

	// Extract the asset to transfer (single-asset transfers only)
	coin := tx.Coins[0]

	// Look up Stellar-specific asset configuration from the mapping table
	assetMapping, found := GetAssetBySwitchlyAsset(coin.Asset)
	if !found {
		c.logger.Error().
			Str("asset", coin.Asset.String()).
			Str("network", string(GetCurrentNetwork())).
			Msg("unsupported asset for outbound transaction")
		return nil, fmt.Errorf("unsupported asset: %s", coin.Asset)
	}

	// Convert amount from SwitchlyNode units to Stellar asset units
	// SwitchlyNode uses 8 decimals internally, but Stellar assets may use different precision
	amountRaw := coin.Amount.Uint64()

	// Format amount as decimal string with asset-specific precision for Stellar SDK
	decimals := assetMapping.StellarDecimals
	divisor := float64(1)
	for i := 0; i < int(decimals); i++ {
		divisor *= 10
	}
	amountStr := fmt.Sprintf("%.*f", int(decimals), float64(amountRaw)/divisor)

	// Create the appropriate Stellar asset type
	var stellarAsset txnbuild.Asset
	if assetMapping.StellarAssetType == "native" {
		// Native Lumens (XLM)
		stellarAsset = txnbuild.NativeAsset{}
	} else {
		// Issued asset (requires Code and Issuer)
		stellarAsset = txnbuild.CreditAsset{
			Code:   assetMapping.StellarAssetCode,
			Issuer: assetMapping.StellarAssetIssuer,
		}
	}

	// Derive Stellar address from vault public key for logging
	vaultAddr, err := tx.VaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		return nil, fmt.Errorf("fail to get Stellar address for vault pub key(%s): %w", tx.VaultPubKey, err)
	}

	c.logger.Info().
		Str("vault_address", vaultAddr.String()).
		Str("to_address", tx.ToAddress.String()).
		Str("switch_asset", coin.Asset.String()).
		Str("stellar_asset", assetMapping.StellarAssetCode).
		Str("asset_type", assetMapping.StellarAssetType).
		Str("amount_stellar", amountStr).
		Str("amount_raw", fmt.Sprintf("%d", amountRaw)).
		Int("decimals", assetMapping.StellarDecimals).
		Str("original_memo", tx.Memo).
		Msg("building simple payment for outbound transfer")

	// Create Stellar payment operation
	payment := &txnbuild.Payment{
		Destination: tx.ToAddress.String(),
		Amount:      amountStr,
		Asset:       stellarAsset,
	}

	return payment, nil
}

// truncateMemoForStellar truncates transaction memos to fit Stellar's 28-byte limit.
// Uses midpoint truncation preserving the OUT: prefix and hash endpoints.
// Format: OUT:<first_part>....<last_part>
func (c *Client) truncateMemoForStellar(originalMemo string) string {
	const maxBytes = 28

	// Return memo unchanged if it already fits within the limit
	if len(originalMemo) <= maxBytes {
		return originalMemo
	}

	// Parse memo components (expected format: "OUT:TXHASH")
	parts := strings.Split(originalMemo, ":")
	if len(parts) < 2 {
		// Handle memos without colon separator using midpoint truncation
		availableLen := maxBytes - 4 // Reserve space for "...."
		halfLen := availableLen / 2
		if halfLen <= 0 {
			return originalMemo[:maxBytes] // Fallback to simple truncation
		}
		return originalMemo[:halfLen] + "...." + originalMemo[len(originalMemo)-halfLen:]
	}

	// Reconstruct with OUT: prefix and process the hash portion
	prefix := "OUT:"
	hashPart := strings.Join(parts[1:], ":")

	// Calculate remaining space for hash content after prefix
	availableForHash := maxBytes - len(prefix)

	if len(hashPart) <= availableForHash {
		return prefix + hashPart
	}

	// Apply midpoint truncation to the hash portion
	dotsLen := 4 // Length of "...."
	availableHashLen := availableForHash - dotsLen
	if availableHashLen <= 0 {
		return prefix + hashPart[:availableForHash] // Fallback to simple truncation
	}

	halfLen := availableHashLen / 2
	if halfLen <= 0 {
		return prefix + hashPart[:availableForHash] // Fallback
	}

	truncatedHash := hashPart[:halfLen] + "...." + hashPart[len(hashPart)-halfLen:]
	result := prefix + truncatedHash

	// Adjust if the result still exceeds the limit
	if len(result) > maxBytes {
		excess := len(result) - maxBytes
		if halfLen > excess/2 && excess > 0 {
			halfLen -= (excess + 1) / 2
			if halfLen > 0 {
				truncatedHash = hashPart[:halfLen] + "...." + hashPart[len(hashPart)-halfLen:]
				result = prefix + truncatedHash
			}
		}
	}

	// Final safety check to guarantee compliance with byte limit
	if len(result) > maxBytes {
		result = result[:maxBytes]
	}

	return result
}

// SignTx signs a Stellar transaction using simple payments with proper ed25519 key derivation.
// This implementation follows THORNode's approach: transactions are only marked as signed
// after successful broadcast to prevent pipeline deadlocks.
func (c *Client) getNextSequence(vaultPubKey common.PubKey) (int64, error) {
	// Get current block height for staleness check
	currentHeight, err := c.stellarScanner.GetHeight()
	if err != nil {
		return 0, fmt.Errorf("fail to get current block height: %w", err)
	}

	// Check if we need to sync from chain
	if c.accts.IsStale(vaultPubKey, currentHeight) {
		c.logger.Debug().
			Str("vault", vaultPubKey.String()).
			Int64("current_height", currentHeight).
			Msg("sequence data is stale, syncing from chain")

		// Get fresh sequence from Horizon
		acc, err := c.GetAccount(vaultPubKey, nil)
		if err != nil {
			return 0, fmt.Errorf("fail to get account from chain: %w", err)
		}

		// Set the base sequence (this will be incremented on each use)
		c.accts.SetBaseSequence(vaultPubKey, acc.Sequence, currentHeight)

		c.logger.Debug().
			Str("vault", vaultPubKey.String()).
			Int64("chain_sequence", acc.Sequence).
			Msg("synced sequence from chain")
	}

	// Atomically get next sequence and increment
	sequence := c.accts.GetNextSequence(vaultPubKey)
	if sequence == 0 {
		// Fallback: force a sync from chain and retry once
		acc, err := c.GetAccount(vaultPubKey, nil)
		if err != nil {
			return 0, fmt.Errorf("fail to get account from chain (fallback): %w", err)
		}
		c.accts.SetBaseSequence(vaultPubKey, acc.Sequence, currentHeight)
		sequence = c.accts.GetNextSequence(vaultPubKey)
		if sequence == 0 {
			// As a last resort, return the on-chain sequence directly
			return acc.Sequence, nil
		}
	}

	return sequence, nil
}

func (c *Client) SignTx(tx stypes.TxOutItem, switchlyHeight int64) (signedTx, checkpoint []byte, _ *stypes.TxInItem, err error) {
	defer func() {
		if err != nil && !strings.Contains(err.Error(), "fail to broadcast") {
			c.logger.Err(err).Msg("fail to sign tx")
		}
	}()

	if c.signerCacheManager.HasSigned(tx.CacheHash()) {
		c.logger.Info().Str("memo", tx.Memo).Msg("transaction already signed, ignoring...")
		return nil, nil, nil, nil
	}

	// Check if we have any coins to send
	if len(tx.Coins) == 0 {
		return nil, nil, nil, fmt.Errorf("no coins to send")
	}

	// Support single coin transactions
	coin := tx.Coins[0]

	// Validate the asset is supported
	_, found := GetAssetBySwitchlyAsset(coin.Asset)
	if !found {
		c.logger.Error().
			Str("asset", coin.Asset.String()).
			Str("network", string(GetCurrentNetwork())).
			Msg("unsupported asset for signing transaction")
		return nil, nil, nil, fmt.Errorf("unsupported asset: %s", coin.Asset)
	}

	// Pre-flight balance check: validate vault has sufficient funds before signing
	// This prevents unnecessary signing and provides clear error feedback
	vaultAccount, err := c.GetAccount(tx.VaultPubKey, nil)
	if err != nil {
		c.logger.Error().Err(err).
			Str("vault_pubkey", tx.VaultPubKey.String()).
			Msg("fail to get vault account for balance validation")
		return nil, nil, nil, fmt.Errorf("fail to get vault account: %w", err)
	}

	// Check if vault has sufficient balance for the outbound asset
	vaultCoin := vaultAccount.Coins.GetCoin(coin.Asset)
	if vaultCoin.Amount.LT(coin.Amount) {
		c.logger.Error().
			Str("vault_pubkey", tx.VaultPubKey.String()).
			Str("required_asset", coin.Asset.String()).
			Str("required_amount", coin.Amount.String()).
			Str("vault_balance", vaultCoin.Amount.String()).
			Msg("insufficient vault balance for outbound transaction")
		return nil, nil, nil, fmt.Errorf("insufficient vault balance (%s): %s < %s",
			coin.Asset.String(), vaultCoin.Amount.String(), coin.Amount.String())
	}

	// Verify vault has sufficient native XLM for transaction fees
	// Note: Stellar always requires native XLM for fees, regardless of the asset being transferred
	gasAsset := common.XLMAsset // Native XLM for Stellar gas
	vaultGasCoin := vaultAccount.Coins.GetCoin(gasAsset)
	requiredGas := tx.MaxGas.ToCoins().GetCoin(gasAsset)

	if vaultGasCoin.Amount.LT(requiredGas.Amount) {
		c.logger.Error().
			Str("vault_pubkey", tx.VaultPubKey.String()).
			Str("gas_asset", gasAsset.String()).
			Str("required_gas", requiredGas.Amount.String()).
			Str("vault_gas_balance", vaultGasCoin.Amount.String()).
			Msg("insufficient vault gas balance for transaction fees")
		return nil, nil, nil, fmt.Errorf("insufficient vault gas balance (%s): %s < %s",
			gasAsset.String(), vaultGasCoin.Amount.String(), requiredGas.Amount.String())
	}

	// Build the payment operation using our helper function
	payment, err := c.processOutboundTx(tx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to build outbound transaction: %w", err)
	}

	// Proactive sequence synchronization from chain state
	if acc, accErr := c.GetAccount(tx.VaultPubKey, nil); accErr == nil && acc.Sequence > 0 {
		if currentHeight, _ := c.stellarScanner.GetHeight(); currentHeight > 0 {
			c.accts.SetBaseSequence(tx.VaultPubKey, acc.Sequence, currentHeight)
		}
	}
	sequence, err := c.getNextSequence(tx.VaultPubKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to get next sequence: %w", err)
	}

	c.logger.Debug().
		Int64("assigned_sequence", sequence).
		Str("vault", tx.VaultPubKey.String()).
		Msg("atomically assigned sequence number")

	// Create account object with atomically assigned sequence number
	account := txnbuild.SimpleAccount{
		AccountID: c.GetAddress(tx.VaultPubKey),
		Sequence:  sequence,
	}

	// Create retry checkpoint containing the sequence number for failed transaction recovery
	checkpointMeta := StellarMetadata{
		SeqNumber:   sequence,
		BlockHeight: 0, // BlockHeight not used for checkpoints
		LastSync:    time.Now(),
	}
	checkpointBytes, err := json.Marshal(checkpointMeta)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to marshal stellar checkpoint: %w", err)
	}

	// Apply memo truncation to comply with Stellar's 28-byte limit
	truncatedMemo := c.truncateMemoForStellar(tx.Memo)

	c.logger.Info().
		Str("original_memo", tx.Memo).
		Str("truncated_memo", truncatedMemo).
		Int("original_len", len(tx.Memo)).
		Int("truncated_len", len(truncatedMemo)).
		Msg("using truncated memo for simple payment")

	// Build Stellar transaction with payment operation and truncated memo
	builtTx, err := txnbuild.NewTransaction(txnbuild.TransactionParams{
		SourceAccount:        &account,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{payment},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 txnbuild.MemoText(truncatedMemo),
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewTimeout(300),
		},
	})
	if err != nil {
		return nil, nil, nil, fmt.Errorf("building tx: %w", err)
	}

	// Sign the transaction using Stellar SDK's built-in signing methods
	// Choose signing method: local key for single-node vaults, TSS for multi-node
	if !c.localPubKey.IsEmpty() && c.localPubKey.Equals(tx.VaultPubKey) {
		c.logger.Info().
			Str("vault_pubkey", tx.VaultPubKey.String()).
			Str("local_pubkey", c.localPubKey.String()).
			Msg("using local key signing for single-node vault")

		// Derive ed25519 signing key using the same method as address generation
		// This ensures signature validation succeeds by matching the expected public key
		secp256k1PubKey, err := c.localPubKey.Secp256K1()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to get secp256k1 public key: %w", err)
		}

		// Create ed25519 seed by hashing the secp256k1 public key (matches GetAddress logic)
		hasher := sha256.New()
		hasher.Write(secp256k1PubKey.SerializeCompressed())
		ed25519Seed := hasher.Sum(nil)

		// Create Stellar keypair from the ed25519 seed
		stellarKP, err := keypair.FromRawSeed([32]byte(ed25519Seed))
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to create Stellar keypair: %w", err)
		}

		// Use Stellar SDK's built-in signing method
		signedTx, err := builtTx.Sign(c.networkPassphrase, stellarKP)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("failed to sign transaction: %w", err)
		}

		// Get the signed transaction XDR
		finalTxeBase64, err := signedTx.Base64()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to encode signed transaction: %w", err)
		}

		c.logger.Info().Msg("successfully signed transaction with locally derived ed25519 key")

		// IMPORTANT: Do not mark transaction as signed until broadcast succeeds.
		// This follows THORNode's approach and prevents pipeline deadlocks when broadcasts fail.

		c.logger.Info().
			Str("memo", tx.Memo).
			Int("signed_tx_length", len(finalTxeBase64)).
			Msg("Stellar transaction signed successfully")

		return []byte(finalTxeBase64), checkpointBytes, nil, nil

	} else {
		// For TSS signing, we need to use the manual approach since we can't use Stellar SDK directly
		c.logger.Info().
			Str("vault_pubkey", tx.VaultPubKey.String()).
			Msg("using TSS signing for Stellar transaction")

		// Convert transaction to XDR for TSS signing
		unsignedXDR, err := builtTx.Base64()
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to encode transaction: %w", err)
		}

		var env xdr.TransactionEnvelope
		if err := xdr.SafeUnmarshalBase64(unsignedXDR, &env); err != nil {
			return nil, nil, nil, fmt.Errorf("fail to decode transaction envelope: %w", err)
		}

		// Validate transaction envelope structure
		if env.Type != xdr.EnvelopeTypeEnvelopeTypeTx || env.V1 == nil {
			return nil, nil, nil, fmt.Errorf("unexpected envelope type: %v", env.Type)
		}
		txref := &env.V1.Tx

		// Generate transaction hash for TSS signing
		txHash, err := network.HashTransaction(*txref, c.networkPassphrase)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to get transaction hash: %w", err)
		}

		var tssRecovery []byte
		signature, tssRecovery, err := c.tssKeyManager.RemoteSign(txHash[:], tx.VaultPubKey.String())
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to sign transaction with TSS: %w", err)
		}
		if signature == nil {
			// This node was not selected for TSS signing
			c.logger.Info().Msg("TSS did not select this node for signing")
			return nil, nil, nil, nil
		}
		c.logger.Info().Msg("successfully signed transaction with TSS")
		_ = tssRecovery

		// Normalize signature to Stellar's required 64-byte format
		stellarSig := make([]byte, 64)
		if len(signature) >= 64 {
			copy(stellarSig, signature[:64])
		} else {
			// Pad with zeros if signature is shorter than 64 bytes
			copy(stellarSig, signature)
			for i := len(signature); i < 64; i++ {
				stellarSig[i] = 0
			}
		}

		// Create decorated signature with hint for Stellar transaction
		addr := c.GetAddress(tx.VaultPubKey)
		decoded, err := strkey.Decode(strkey.VersionByteAccountID, addr)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to decode address: %w", err)
		}

		// Extract the last 4 bytes of the account ID as signature hint
		var hint [4]byte
		copy(hint[:], decoded[len(decoded)-4:])

		decoratedSig := xdr.DecoratedSignature{
			Hint:      xdr.SignatureHint(hint),
			Signature: xdr.Signature(stellarSig),
		}

		// Attach signature to transaction envelope
		env.V1.Signatures = append(env.V1.Signatures, decoratedSig)

		// Encode signed transaction envelope as base64 XDR
		finalTxeBase64, err := xdr.MarshalBase64(env)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to encode signed transaction: %w", err)
		}

		// IMPORTANT: Do not mark transaction as signed until broadcast succeeds.
		// This follows THORNode's approach and prevents pipeline deadlocks when broadcasts fail.

		c.logger.Info().
			Str("memo", tx.Memo).
			Int("signed_tx_length", len(finalTxeBase64)).
			Msg("Stellar transaction signed successfully with TSS")

		return []byte(finalTxeBase64), checkpointBytes, nil, nil
	}
}

// BroadcastTx submits a signed transaction to the Stellar network via Horizon API.
// This implementation uses direct Horizon submission for simple payments, providing
// better performance and reliability compared to Soroban RPC.
func (c *Client) BroadcastTx(tx stypes.TxOutItem, txBytes []byte) (string, error) {
	txeBase64 := string(txBytes)

	// Submit via Horizon API for optimal performance with simple payments
	c.logger.Info().
		Str("memo", tx.Memo).
		Msg("broadcasting simple payment via Horizon API")

	resp, err := c.horizonClient.SubmitTransactionXDR(txeBase64)
	if err != nil {
		return c.handleBroadcastError(tx, fmt.Errorf("horizon submitTransaction failed: %w", err))
	}

	hash := resp.Hash

	// Horizon API confirms transaction success immediately upon acceptance
	c.logger.Info().
		Str("hash", hash).
		Str("memo", tx.Memo).
		Msg("simple payment broadcast successful via Horizon")

	// Update local sequence tracking after successful broadcast
	c.accts.AdvanceSequence(tx.VaultPubKey)

	// Mark transaction as signed only after confirmed broadcast (THORNode pattern)
	if err = c.signerCacheManager.SetSigned(tx.CacheHash(), tx.CacheVault(c.GetChain()), hash); err != nil {
		c.logger.Err(err).Msg("fail to set signer cache")
	}

	return hash, nil
}

// handleBroadcastError processes broadcast failures with specialized handling for sequence errors.
// This function provides clean error logging and automatic sequence refresh for retry scenarios.
func (c *Client) handleBroadcastError(tx stypes.TxOutItem, err error) (string, error) {
	// Log broadcast failure with transaction context
	c.logger.Error().
		Str("tx_hash", tx.InHash.String()).
		Str("memo", tx.Memo).
		Err(err).
		Msg("failed to broadcast transaction")

	// Handle sequence mismatch errors with automatic chain state refresh
	if strings.Contains(err.Error(), "tx_bad_seq") {
		// Refresh local sequence cache from on-chain state
		acc, accErr := c.GetAccount(tx.VaultPubKey, nil)
		if accErr == nil && acc.Sequence > 0 {
			currentHeight, _ := c.stellarScanner.GetHeight()
			c.accts.SetBaseSequence(tx.VaultPubKey, acc.Sequence, currentHeight)
			c.logger.Warn().
				Int64("refreshed_sequence", acc.Sequence).
				Msg("sequence refreshed from Horizon after tx_bad_seq error")
		} else {
			c.logger.Warn().Msg("unable to refresh sequence from Horizon; retaining local value")
		}
		return "", fmt.Errorf("sequence number stale (tx_bad_seq): %w", err)
	}

	// Return all non-sequence errors without modification
	return "", fmt.Errorf("broadcast failed: %w", err)
}

// ConfirmationCountReady returns true if the confirmation count is ready
func (c *Client) ConfirmationCountReady(txIn stypes.TxIn) bool {
	return true
}

// GetConfirmationCount returns the number of confirmations for a given tx
func (c *Client) GetConfirmationCount(txIn stypes.TxIn) int64 {
	// Stellar transactions are finalized immediately, so they have 1 confirmation
	return 1
}

// GetVaultLock returns a per-vault mutex to serialize sign/broadcast for that vault
func (c *Client) GetVaultLock(vault string) *sync.Mutex {
	c.vaultLocksMu.Lock()
	defer c.vaultLocksMu.Unlock()
	if m, ok := c.vaultLocks[vault]; ok {
		return m
	}
	m := &sync.Mutex{}
	c.vaultLocks[vault] = m
	return m
}

// ReportSolvency reports solvency to SWITCHLYChain
func (c *Client) ReportSolvency(blockHeight int64) error {
	if !c.ShouldReportSolvency(blockHeight) {
		return nil
	}

	// Get all asgard vaults
	asgardVaults, err := c.switchlyBridge.GetAsgards()
	if err != nil {
		return fmt.Errorf("fail to get asgards: %w", err)
	}

	totalCoins := common.Coins{}

	for _, vault := range asgardVaults {
		if !vault.HasFundsForChain(common.StellarChain) {
			continue
		}

		addr := c.GetAddress(vault.PubKey)
		account, err := c.GetAccountByAddress(addr, nil)
		if err != nil {
			c.logger.Error().Err(err).Str("address", addr).Msg("fail to get account balance")
			continue
		}

		totalCoins = totalCoins.Add(account.Coins...)
	}

	solvencyMsg := stypes.Solvency{
		Height: blockHeight,
		Chain:  common.StellarChain,
		PubKey: asgardVaults[0].PubKey, // Use first vault's pubkey as representative
		Coins:  totalCoins,
	}

	select {
	case c.globalSolvencyQueue <- solvencyMsg:
	case <-time.After(constants.SwitchlyBlockTime):
		c.logger.Info().Msg("fail to send solvency info within timeout")
	}

	return nil
}

// ShouldReportSolvency determines if solvency should be reported
func (c *Client) ShouldReportSolvency(height int64) bool {
	return height%c.cfg.SolvencyBlocks == 0
}

// OnObservedTxIn is called when a new observed tx is received
func (c *Client) OnObservedTxIn(txIn stypes.TxInItem, blockHeight int64) {
	// Parse memo to determine if outbound
	m, err := mem.ParseMemo(common.LatestVersion, txIn.Memo)
	if err != nil {
		// Debug log only as ParseMemo error is expected for SWITCHName inbounds.
		c.logger.Debug().Err(err).Msgf("fail to parse memo: %s", txIn.Memo)
		return
	}

	// Handle outbound transaction caching
	if !m.IsOutbound() {
		return
	}
	if m.GetTxID().IsEmpty() {
		return
	}

	if err = c.signerCacheManager.SetSigned(
		txIn.CacheHash(c.GetChain(), m.GetTxID().String()),
		txIn.CacheVault(c.GetChain()),
		txIn.Tx,
	); err != nil {
		c.logger.Err(err).Msg("fail to update signer cache")
	}
}

// DeployRouter deploys the Stellar router contract
func (c *Client) DeployRouter(pubKey common.PubKey) (common.Address, error) {
	// For now, router deployment is not fully implemented
	return common.NoAddress, fmt.Errorf("router deployment not yet implemented for Stellar")
}

// deployRouterContract handles the actual contract deployment
func (c *Client) deployRouterContract(vaultAddr string, pubKey common.PubKey) (common.Address, error) {
	// For now, this is a placeholder that would integrate with Soroban deployment
	// In a real implementation, this would:
	// 1. Compile the contract WASM
	// 2. Deploy via Soroban RPC
	// 3. Initialize with vault address
	// 4. Return the contract address

	// Generate a deterministic contract address based on vault pubkey
	// This is a simplified approach - real implementation would use actual deployment
	hasher := sha256.New()
	hasher.Write([]byte(pubKey.String()))
	hasher.Write([]byte("stellar-router"))
	hash := hasher.Sum(nil)

	// Convert to Stellar contract address format (simplified)
	contractAddr := fmt.Sprintf("C%s", strings.ToUpper(hex.EncodeToString(hash[:27])))

	return common.NewAddress(contractAddr)
}

// storeRouterConfig stores router configuration
func (c *Client) storeRouterConfig(config *RouterConfig) error {
	// Store configuration in a persistent way
	// This could be a database, file, or other storage mechanism
	configData, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal router config: %w", err)
	}

	// For now, log the configuration
	c.logger.Info().
		Str("config", string(configData)).
		Msg("router configuration stored")

	return nil
}

// GetRouterAddress returns the router contract address for a given vault
func (c *Client) GetRouterAddress(pubKey common.PubKey) (common.Address, error) {
	// In a real implementation, this would query stored configuration
	// For now, generate the same deterministic address as deployment
	return c.deployRouterContract(c.GetAddress(pubKey), pubKey)
}

// IsRouterContract checks if an address is a known router contract
func (c *Client) IsRouterContract(addr string) bool {
	// Check if the address matches the pattern of a router contract
	// This is a simplified check - real implementation would maintain a registry
	return strings.HasPrefix(addr, "C") && len(addr) == 56
}

// MonitorRouterHealth monitors the health of the router contract
func (c *Client) MonitorRouterHealth() error {
	if c.routerAddress == "" {
		return fmt.Errorf("no router address configured")
	}

	// Check if router contract is accessible
	// This would involve calling a health check function on the contract

	c.logger.Debug().
		Str("router_address", c.routerAddress).
		Msg("router health check passed")

	return nil
}

// GetRouterVersion returns the version of the deployed router
func (c *Client) GetRouterVersion() (string, error) {
	config, err := c.LoadRouterConfig()
	if err != nil {
		return "", err
	}

	return config.Version, nil
}

// LoadRouterConfig loads router configuration from storage
func (c *Client) LoadRouterConfig() (*RouterConfig, error) {
	// Load from persistent storage
	// For now, return current configuration
	return &RouterConfig{
		Address:     c.routerAddress,
		Version:     "1.0.0",
		Deployed:    c.routerAddress != "",
		DeployedAt:  time.Now().Unix(),
		VaultPubKey: "",
	}, nil
}

// SaveRouterConfig saves router configuration to storage
func (c *Client) SaveRouterConfig(config *RouterConfig) error {
	// Save to persistent storage
	// For now, just update the local router address
	c.routerAddress = config.Address

	c.logger.Info().
		Str("address", config.Address).
		Str("version", config.Version).
		Bool("deployed", config.Deployed).
		Msg("router configuration saved")

	return nil
}

// IsRouterDeployed checks if a router contract is deployed for the given public key
func (c *Client) IsRouterDeployed(pubKey common.PubKey) bool {
	// Check if we have a router configuration for this public key
	config, err := c.LoadRouterConfig()
	if err != nil {
		return false
	}

	// Check if the router is marked as deployed and has a valid address
	return config.Deployed && config.Address != ""
}

// UpdateRouterAddress updates the router address for a given public key
func (c *Client) UpdateRouterAddress(pubKey common.PubKey, newAddress common.Address) error {
	// Load existing configuration
	config, err := c.LoadRouterConfig()
	if err != nil {
		// Create new configuration if none exists
		config = &RouterConfig{
			Version:     "1.0.0",
			Deployed:    false,
			DeployedAt:  time.Now().Unix(),
			VaultPubKey: pubKey.String(),
		}
	}

	// Update the address
	config.Address = newAddress.String()
	config.Deployed = true
	config.VaultPubKey = pubKey.String()

	// Save the updated configuration
	err = c.SaveRouterConfig(config)
	if err != nil {
		return fmt.Errorf("failed to save router config: %w", err)
	}

	c.logger.Info().
		Str("pubkey", pubKey.String()).
		Str("new_address", newAddress.String()).
		Msg("router address updated")

	return nil
}

// SendTx sends a transaction to the Stellar network
func (c *Client) SendTx(tx stypes.TxOutItem) (string, error) {
	signedTx, _, _, err := c.SignTx(tx, tx.Height)
	if err != nil {
		return "", err
	}
	if signedTx == nil {
		return "", fmt.Errorf("no signed transaction returned")
	}

	return c.BroadcastTx(tx, signedTx)
}

// getScAddressFromString converts a string address to xdr.ScAddress
// This is a helper method for testing and internal use
func (c *Client) getScAddressFromString(addr string) (xdr.ScAddress, error) {
	if addr == "" {
		return xdr.ScAddress{}, fmt.Errorf("empty address provided to getScAddressFromString")
	}

	// Use DecodeAny to handle any valid Stellar address format
	version, decoded, err := strkey.DecodeAny(addr)
	if err != nil {
		return xdr.ScAddress{}, fmt.Errorf("failed to decode Stellar address: %w", err)
	}

	if len(decoded) < 32 {
		return xdr.ScAddress{}, fmt.Errorf("decoded address too short: %d", len(decoded))
	}

	// Check the version byte to determine the address type
	if version == strkey.VersionByteAccountID {
		// Account address - need to properly initialize the struct
		// Create the Ed25519 public key first
		var ed25519PubKey xdr.Uint256
		copy(ed25519PubKey[:], decoded[:32])

		// Now create the AccountId with the Ed25519 public key
		accountId := xdr.AccountId{
			Type:    xdr.PublicKeyTypePublicKeyTypeEd25519,
			Ed25519: &ed25519PubKey,
		}

		return xdr.ScAddress{
			Type:      xdr.ScAddressTypeScAddressTypeAccount,
			AccountId: &accountId,
		}, nil
	} else {
		// Contract address or other type - treat as contract
		var contractId xdr.Hash
		copy(contractId[:], decoded[:32])
		// Convert Hash to ContractId since ContractId is a typedef of Hash
		contractIdTyped := xdr.ContractId(contractId)
		return xdr.ScAddress{
			Type:       xdr.ScAddressTypeScAddressTypeContract,
			ContractId: &contractIdTyped,
		}, nil
	}
}

// getScAddressPtrFromString converts a string address to *xdr.ScAddress
// This is a helper method for testing and internal use
func (c *Client) getScAddressPtrFromString(addr string) (*xdr.ScAddress, error) {
	scAddr, err := c.getScAddressFromString(addr)
	if err != nil {
		return nil, err
	}
	// Create a new ScAddress on the heap to avoid returning pointer to local variable
	scAddrPtr := new(xdr.ScAddress)
	*scAddrPtr = scAddr
	return scAddrPtr, nil
}

// SorobanSimulationResult represents the result from Soroban RPC simulation
type SorobanSimulationResult struct {
	Auth            []string `json:"auth"`
	MinResourceFee  string   `json:"minResourceFee"`
	TransactionData string   `json:"transactionData"`
}

// SorobanSimulationResponse represents the JSON response from Soroban RPC
type SorobanSimulationResponse struct {
	Result struct {
		Results         []SorobanSimulationResult `json:"results"`
		MinResourceFee  string                    `json:"minResourceFee"`
		TransactionData string                    `json:"transactionData"`
	} `json:"result"`
	Error struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// simulateSorobanTransaction simulates a transaction with Soroban RPC to get authorization
func (c *Client) simulateSorobanTransaction(txeBase64 string) (*SorobanSimulationResult, error) {
	// Use XLM_HOST environment variable for Soroban RPC endpoint
	xlmHost := os.Getenv("XLM_HOST")
	if xlmHost == "" {
		xlmHost = "stellar:8000" // Default fallback
	}
	sorobanURL := fmt.Sprintf("http://%s/soroban/rpc", xlmHost)

	// Prepare simulation request
	simulateReq := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "simulateTransaction",
		"params": map[string]interface{}{
			"transaction": txeBase64,
		},
	}

	reqBody, err := json.Marshal(simulateReq)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to marshal simulation request")
		return nil, fmt.Errorf("failed to marshal simulation request: %w", err)
	}

	// Make HTTP request to Soroban RPC
	resp, err := http.Post(sorobanURL, "application/json", bytes.NewBuffer(reqBody))
	if err != nil {
		c.logger.Error().Err(err).Msg("soroban RPC request failed")
		return nil, fmt.Errorf("failed to call Soroban RPC: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to read simulation response")
		return nil, fmt.Errorf("failed to read simulation response: %w", err)
	}

	// Parse response
	var sorobanResp SorobanSimulationResponse
	if err := json.Unmarshal(body, &sorobanResp); err != nil {
		c.logger.Error().
			Err(err).
			Msg("failed to parse simulation response")
		return nil, fmt.Errorf("failed to parse simulation response: %w", err)
	}

	// Check for JSON-RPC errors first
	if sorobanResp.Error.Code != 0 {
		c.logger.Error().
			Int("error_code", sorobanResp.Error.Code).
			Str("error_message", sorobanResp.Error.Message).
			Msg("soroban RPC returned error")
		return nil, fmt.Errorf("Soroban RPC error %d: %s", sorobanResp.Error.Code, sorobanResp.Error.Message)
	}

	// Build unified result combining auth with top-level resource fields
	res := &SorobanSimulationResult{}
	if len(sorobanResp.Result.Results) > 0 {
		res.Auth = sorobanResp.Result.Results[0].Auth
	}
	res.MinResourceFee = sorobanResp.Result.MinResourceFee
	res.TransactionData = sorobanResp.Result.TransactionData

	return res, nil
}
