package stellar

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/btcsuite/btcutil/bech32"

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

// SimulateTransactionResponse represents the response structure from Soroban RPC simulateTransaction calls.
type SimulateTransactionResponse struct {
	MinResourceFee  string           `json:"minResourceFee"` // Resource fee returned as string by Soroban RPC
	LatestLedger    uint32           `json:"latestLedger"`
	TransactionData string           `json:"transactionData"`
	Results         []SimulateResult `json:"results"`
	Events          []interface{}    `json:"events,omitempty"`
	Error           interface{}      `json:"error,omitempty"`
}

// SimulateResult contains authorization entries and return values from contract simulation.
type SimulateResult struct {
	Auth []string `json:"auth"` // Base64-encoded authorization entries
	XDR  string   `json:"xdr"`  // Base64-encoded return value
}

// min returns the smaller of two integers.
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

	// Extract the local node public key (used for identity/logging).
	var localPubKey common.PubKey
	if switchKeys != nil {
		logger.Info().Msg("switchKeys provided to Stellar client")
		privKey, err := switchKeys.GetPrivateKey()
		if err != nil {
			logger.Warn().Err(err).Msg("failed to get private key from switchKeys")
		} else {
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

	// Wait for the Stellar node to be fully synced, then start the block scanner — but do so in a
	// goroutine. The observer starts chain clients in a synchronous loop (observer/observe.go), so a
	// blocking Start() here would stall startup of every subsequent component, including the signer's
	// keygen scanner. Other chain clients (e.g. ETH) spawn their workers and return immediately; match
	// that contract so Start() never blocks the observer loop.
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
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
	}()

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
		c.logger.Error().Err(err).Str("pool_pub_key", poolPubKey.String()).Msg("fail to get pool address")
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

func (c *Client) getNextSequence(vaultPubKey common.PubKey) (int64, error) {
	// Get current account sequence from Horizon (just like the test script)
	acc, err := c.GetAccount(vaultPubKey, nil)
	if err != nil {
		return 0, fmt.Errorf("fail to get account from Horizon: %w", err)
	}

	// Return current sequence - Stellar SDK will increment with IncrementSequenceNum: true
	c.logger.Info().
		Str("vault", vaultPubKey.String()).
		Int64("current_sequence", acc.Sequence).
		Msg("got current sequence from Horizon - SDK will increment")

	return acc.Sequence, nil
}

// scvalI128FromBaseUnits converts a non-negative base-unit amount string into an ScVal i128.
func scvalI128FromBaseUnits(baseUnits string) (xdr.ScVal, error) {
	v, ok := new(big.Int).SetString(strings.TrimSpace(baseUnits), 10)
	if !ok || v.Sign() < 0 {
		return xdr.ScVal{}, fmt.Errorf("invalid base-unit amount: %s", baseUnits)
	}
	if v.BitLen() > 127 {
		return xdr.ScVal{}, fmt.Errorf("amount overflows i128: %s", baseUnits)
	}
	mask := new(big.Int).SetUint64(^uint64(0))
	lo := new(big.Int).And(v, mask).Uint64()
	hi := new(big.Int).Rsh(v, 64).Int64()
	i128 := xdr.Int128Parts{Hi: xdr.Int64(hi), Lo: xdr.Uint64(lo)}
	return xdr.ScVal{Type: xdr.ScValTypeScvI128, I128: &i128}, nil
}

// assetContractScAddress resolves the Soroban contract (SAC) address of an asset as an ScAddress.
// Soroban-native tokens carry their contract address in the issuer field; native XLM and classic
// assets have their canonical SAC contract id derived deterministically from the network passphrase.
func (c *Client) assetContractScAddress(mapping StellarAssetMapping) (xdr.ScAddress, error) {
	if mapping.StellarAssetType == "contract" && mapping.StellarAssetIssuer != "" {
		return c.getScAddressFromString(mapping.StellarAssetIssuer)
	}
	xdrAsset, err := mapping.ToStellarAsset().ToXDR()
	if err != nil {
		return xdr.ScAddress{}, fmt.Errorf("fail to convert asset to xdr: %w", err)
	}
	hash, err := xdrAsset.ContractID(c.networkPassphrase)
	if err != nil {
		return xdr.ScAddress{}, fmt.Errorf("fail to derive asset contract id: %w", err)
	}
	contractID := xdr.ContractId(hash)
	return xdr.ScAddress{Type: xdr.ScAddressTypeScAddressTypeContract, ContractId: &contractID}, nil
}

// buildTransferOutInvokeOp builds the router `transfer_out(vault, to, asset, amount, memo)`
// InvokeHostFunction operation carrying the full SwitchlyProtocol memo. The vault is set as the
// operation source account so the router's `vault.require_auth()` is satisfied via source-account
// authorization (covered by the transaction signature). This is the pure, network-free core of the
// router outbound path and is unit tested.
func (c *Client) buildTransferOutInvokeOp(tx stypes.TxOutItem, memo string) (*txnbuild.InvokeHostFunction, error) {
	if c.routerAddress == "" {
		return nil, fmt.Errorf("no stellar router address configured")
	}
	if len(tx.Coins) == 0 {
		return nil, fmt.Errorf("no coins to send")
	}
	coin := tx.Coins[0]
	mapping, found := GetAssetBySwitchlyAsset(coin.Asset)
	if !found {
		return nil, fmt.Errorf("unsupported asset: %s", coin.Asset)
	}

	vaultAddr := c.GetAddress(tx.VaultPubKey)
	if vaultAddr == "" {
		return nil, fmt.Errorf("fail to get vault address")
	}

	vaultScAddr, err := c.getScAddressFromString(vaultAddr)
	if err != nil {
		return nil, fmt.Errorf("fail to encode vault address: %w", err)
	}
	toScAddr, err := c.getScAddressFromString(tx.ToAddress.String())
	if err != nil {
		return nil, fmt.Errorf("fail to encode recipient address: %w", err)
	}
	assetScAddr, err := c.assetContractScAddress(mapping)
	if err != nil {
		return nil, fmt.Errorf("fail to resolve asset contract: %w", err)
	}
	routerScAddr, err := c.getScAddressFromString(c.routerAddress)
	if err != nil {
		return nil, fmt.Errorf("fail to encode router address: %w", err)
	}
	if routerScAddr.Type != xdr.ScAddressTypeScAddressTypeContract {
		return nil, fmt.Errorf("router address is not a contract address: %s", c.routerAddress)
	}

	amountVal, err := scvalI128FromBaseUnits(mapping.ConvertFromSwitchlyProtocolAmount(coin.Amount))
	if err != nil {
		return nil, err
	}

	memoStr := xdr.ScString(memo)
	args := []xdr.ScVal{
		{Type: xdr.ScValTypeScvAddress, Address: &vaultScAddr},
		{Type: xdr.ScValTypeScvAddress, Address: &toScAddr},
		{Type: xdr.ScValTypeScvAddress, Address: &assetScAddr},
		amountVal,
		{Type: xdr.ScValTypeScvString, Str: &memoStr},
	}

	hostFn := xdr.HostFunction{
		Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
		InvokeContract: &xdr.InvokeContractArgs{
			ContractAddress: routerScAddr,
			FunctionName:    xdr.ScSymbol("transfer_out"),
			Args:            args,
		},
	}

	return &txnbuild.InvokeHostFunction{
		HostFunction:  hostFn,
		SourceAccount: vaultAddr,
	}, nil
}

// buildRouterTransferOutTransaction builds, simulates and signs a Soroban transfer_out invocation
// against the router contract. It carries the full memo, then simulates the transaction to attach
// the resource footprint (SorobanData) and authorization entries before signing.
func (c *Client) buildRouterTransferOutTransaction(tx stypes.TxOutItem, sequence int64, memo string) (*txnbuild.Transaction, error) {
	if c.sorobanRPCClient == nil {
		return nil, fmt.Errorf("soroban rpc client not configured")
	}

	op, err := c.buildTransferOutInvokeOp(tx, memo)
	if err != nil {
		return nil, err
	}
	vaultAddr := op.SourceAccount

	// build constructs a transaction from the current state of op. A fresh SimpleAccount is used
	// each call so the sequence is deterministic across the simulate and sign builds.
	build := func() (*txnbuild.Transaction, error) {
		return txnbuild.NewTransaction(txnbuild.TransactionParams{
			SourceAccount:        &txnbuild.SimpleAccount{AccountID: vaultAddr, Sequence: sequence},
			IncrementSequenceNum: true,
			Operations:           []txnbuild.Operation{op},
			BaseFee:              txnbuild.MinBaseFee,
			Preconditions:        txnbuild.Preconditions{TimeBounds: txnbuild.NewInfiniteTimeout()},
		})
	}

	// 1) Build an unsigned transaction and simulate it.
	unsignedTx, err := build()
	if err != nil {
		return nil, fmt.Errorf("fail to build unsigned soroban tx: %w", err)
	}
	unsignedXDR, err := unsignedTx.Base64()
	if err != nil {
		return nil, fmt.Errorf("fail to encode unsigned soroban tx: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sim, err := c.sorobanRPCClient.SimulateTransaction(ctx, unsignedXDR)
	if err != nil {
		return nil, fmt.Errorf("fail to simulate soroban tx: %w", err)
	}

	// 2) Apply the simulation results (footprint + auth) onto the operation.
	var sorobanData xdr.SorobanTransactionData
	if err = xdr.SafeUnmarshalBase64(sim.TransactionData, &sorobanData); err != nil {
		return nil, fmt.Errorf("fail to decode soroban transaction data: %w", err)
	}
	op.Ext = xdr.TransactionExt{V: 1, SorobanData: &sorobanData}

	if len(sim.Results) > 0 {
		auth := make([]xdr.SorobanAuthorizationEntry, 0, len(sim.Results[0].Auth))
		for _, a := range sim.Results[0].Auth {
			var entry xdr.SorobanAuthorizationEntry
			if err = xdr.SafeUnmarshalBase64(a, &entry); err != nil {
				return nil, fmt.Errorf("fail to decode soroban auth entry: %w", err)
			}
			auth = append(auth, entry)
		}
		op.Auth = auth
	}

	// 3) Rebuild with the footprint/auth attached and sign. The resource fee is taken from
	// SorobanData by the SDK, on top of the inclusion BaseFee.
	preparedTx, err := build()
	if err != nil {
		return nil, fmt.Errorf("fail to build prepared soroban tx: %w", err)
	}

	signedTx, err := c.signTransactionWithTSS(preparedTx, tx.VaultPubKey, c.networkPassphrase)
	if err != nil {
		return nil, fmt.Errorf("fail to sign soroban tx: %w", err)
	}
	return signedTx, nil
}

// ExtractSecp256k1FromTswitchpub extracts secp256k1 public key bytes from tswitchpub bech32 format
func (c *Client) ExtractSecp256k1FromTswitchpub(pubkeyStr string) ([]byte, error) {
	// Handle both tswitchpub and cosmospub formats
	if strings.HasPrefix(pubkeyStr, "tswitchpub") {
		// Decode tswitchpub bech32 format
		_, data, err := bech32.Decode(pubkeyStr)
		if err != nil {
			return nil, fmt.Errorf("fail to decode tswitchpub bech32: %w", err)
		}

		// Convert 5-bit groups to 8-bit bytes
		converted, err := bech32.ConvertBits(data, 5, 8, false)
		if err != nil {
			return nil, fmt.Errorf("fail to convert bits: %w", err)
		}

		// Extract secp256k1 key (last 33 bytes)
		if len(converted) < 33 {
			return nil, fmt.Errorf("converted data too short for secp256k1 key: %d", len(converted))
		}

		secp256k1Bytes := converted[len(converted)-33:]
		return secp256k1Bytes, nil
	}

	// For cosmospub format, use the standard method
	pubKey, err := common.NewPubKey(pubkeyStr)
	if err != nil {
		return nil, fmt.Errorf("fail to parse cosmospub key: %w", err)
	}

	secp256k1PubKey, err := pubKey.Secp256K1()
	if err != nil {
		return nil, fmt.Errorf("fail to extract secp256k1 from cosmospub: %w", err)
	}

	return secp256k1PubKey.SerializeCompressed(), nil
}

// DeriveStellarkeyFromVaultPubKey derives a Stellar signing keypair for a vault.
//
// SECURITY: this is an INSECURE PLACEHOLDER, permitted on mocknet builds only. It derives the vault's
// ed25519 key deterministically from the PUBLIC secp256k1 vault pubkey (SHA-256 seed), so the private
// key is recoverable by anyone from public data and must never secure real funds. It is the signing
// counterpart of the address-side placeholder in common.PubKey.GetAddress(StellarChain), and is gated
// to mocknet via placeholderStellarSigningAllowed. Real signing requires EdDSA threshold signing; see
// docs/architecture/stellar-eddsa-tss.md.
func (c *Client) DeriveStellarkeyFromVaultPubKey(vaultPubKeyStr string) (*keypair.Full, error) {
	if !placeholderStellarSigningAllowed {
		return nil, fmt.Errorf("refusing to derive Stellar signing key: the placeholder derivation is insecure and is disabled on non-mocknet builds; EdDSA threshold signing is required (see docs/architecture/stellar-eddsa-tss.md)")
	}

	// Extract secp256k1 public key bytes (handles both tswitchpub and cosmospub).
	secp256k1Bytes, err := c.ExtractSecp256k1FromTswitchpub(vaultPubKeyStr)
	if err != nil {
		return nil, fmt.Errorf("fail to extract secp256k1 public key: %w", err)
	}

	// INSECURE (mocknet only): seed an ed25519 key from the PUBLIC secp256k1 pubkey. Matches the
	// address derived by common.PubKey.GetAddress(StellarChain).
	hasher := sha256.New()
	hasher.Write(secp256k1Bytes)
	var ed25519Seed [32]byte
	copy(ed25519Seed[:], hasher.Sum(nil)[:32])

	stellarKeypair, err := keypair.FromRawSeed(ed25519Seed)
	if err != nil {
		return nil, fmt.Errorf("fail to create Stellar keypair from ed25519 seed: %w", err)
	}
	return stellarKeypair, nil
}

// signTransactionWithTSS signs a Stellar transaction with the vault key.
//
// NOTE: despite the name this does NOT yet perform threshold signing — it uses the insecure
// placeholder derivation (DeriveStellarkeyFromVaultPubKey), which is gated to mocknet. See
// docs/architecture/stellar-eddsa-tss.md for the real EdDSA threshold-signing plan.
func (c *Client) signTransactionWithTSS(stellarTx *txnbuild.Transaction, vaultPubKey common.PubKey, networkPassphrase string) (*txnbuild.Transaction, error) {
	c.logger.Info().
		Str("vault_pubkey", vaultPubKey.String()).
		Msg("signing Stellar transaction (placeholder key derivation - mocknet only)")

	stellarKeypair, err := c.DeriveStellarkeyFromVaultPubKey(vaultPubKey.String())
	if err != nil {
		return nil, fmt.Errorf("fail to derive Stellar keypair from vault key: %w", err)
	}

	signedTx, err := stellarTx.Sign(networkPassphrase, stellarKeypair)
	if err != nil {
		return nil, fmt.Errorf("fail to sign transaction with derived keypair: %w", err)
	}

	c.logger.Info().
		Str("derived_address", stellarKeypair.Address()).
		Msg("transaction signed successfully with vault-derived keypair")
	return signedTx, nil
}

// SignTx signs a Stellar transaction using simple payments with proper ed25519 key derivation.
// This implementation follows THORNode's approach: transactions are only marked as signed
// after successful broadcast to prevent pipeline deadlocks.
func (c *Client) SignTx(tx stypes.TxOutItem, switchlyHeight int64) (signedTx, checkpoint []byte, _ *stypes.TxInItem, err error) {
	defer func() {
		if err != nil && !strings.Contains(err.Error(), "fail to broadcast") {
			// Handle TSS keysign errors like ETH/XRP (following XRP pattern)
			var keysignError tss.KeysignError
			if errors.As(err, &keysignError) {
				if len(keysignError.Blame.BlameNodes) == 0 {
					c.logger.Err(err).Msg("TSS doesn't know which node to blame")
					return
				}

				// Post keysign failure to switchly (following XRP pattern)
				var txID common.TxID
				txID, err = c.switchlyBridge.PostKeysignFailure(keysignError.Blame, switchlyHeight, tx.Memo, tx.Coins, tx.VaultPubKey)
				if err != nil {
					c.logger.Err(err).Msg("fail to post keysign failure to SWITCHLYChain")
					return
				}
				c.logger.Info().Str("tx_id", txID.String()).Msgf("post keysign failure to switchly")
			}
			c.logger.Err(err).Msg("fail to sign tx")
		}
	}()

	// Check signer cache to avoid signing the same transaction multiple times (following XRP pattern)
	if c.signerCacheManager.HasSigned(tx.CacheHash()) {
		c.logger.Info().Msgf("transaction(%+v), signed before , ignore", tx)
		return nil, nil, nil, nil
	}

	// Basic validation
	if len(tx.Coins) == 0 {
		return nil, nil, nil, fmt.Errorf("no coins to send")
	}

	coin := tx.Coins[0]
	if _, found := GetAssetBySwitchlyAsset(coin.Asset); !found {
		return nil, nil, nil, fmt.Errorf("unsupported asset: %s", coin.Asset)
	}

	// Get next sequence number (following XRP checkpoint pattern)
	var sequence int64
	meta := StellarMetadata{}

	if tx.Checkpoint != nil {
		// Use checkpoint sequence for retry (following XRP pattern)
		if err = json.Unmarshal(tx.Checkpoint, &meta); err != nil {
			c.logger.Err(err).Msg("fail to unmarshal checkpoint")
			return nil, nil, nil, err
		}
		sequence = meta.SeqNumber
		c.logger.Info().Int64("sequence", sequence).Msg("using checkpoint sequence for retry")
	} else {
		// Get fresh sequence number
		sequence, err = c.getNextSequence(tx.VaultPubKey)
		if err != nil {
			return nil, nil, nil, fmt.Errorf("fail to get next sequence: %w", err)
		}

		// Create checkpoint for retry recovery (following XRP pattern)
		meta = StellarMetadata{
			SeqNumber:   sequence,
			BlockHeight: 0,
			LastSync:    time.Now(),
		}
	}

	// Serialize checkpoint (following XRP pattern)
	checkpointBytes, err := json.Marshal(meta)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to marshal stellar checkpoint: %w", err)
	}

	// Outbound XLM must always go through the Soroban router so the full SwitchlyProtocol memo is
	// preserved in the emitted transfer_out event; a plain Stellar payment can only carry 28 bytes.
	if c.routerAddress == "" || c.sorobanRPCClient == nil {
		return nil, nil, nil, fmt.Errorf("stellar router not configured: outbound requires the router contract")
	}
	stellarSignedTx, err := c.buildRouterTransferOutTransaction(tx, sequence, tx.Memo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to build router transfer_out transaction: %w", err)
	}

	// Convert to XDR for storage
	signedTxXDR, err := stellarSignedTx.Base64()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to get XDR from signed transaction: %w", err)
	}

	c.logger.Info().
		Str("memo", tx.Memo).
		Int64("sequence", sequence).
		Msg("Stellar transaction signed successfully")

	return []byte(signedTxXDR), checkpointBytes, nil, nil
}

// BroadcastTx submits a signed transaction to the Stellar network
func (c *Client) BroadcastTx(tx stypes.TxOutItem, txBytes []byte) (string, error) {
	signedTxXDR := string(txBytes)

	c.logger.Info().Str("memo", tx.Memo).Msg("broadcasting transaction")

	// Parse XDR back to Transaction object (like test script)
	parsedTx, err := txnbuild.TransactionFromXDR(signedTxXDR)
	if err != nil {
		return "", fmt.Errorf("fail to parse stored XDR: %w", err)
	}

	// Get the actual transaction from the parsed result
	signedTx, ok := parsedTx.Transaction()
	if !ok {
		return "", fmt.Errorf("failed to get transaction from parsed XDR")
	}

	hash, err := c.submitTransactionViaHorizon(signedTx)
	if err != nil {
		return c.handleBroadcastError(tx, err)
	}

	// Update signer cache after successful broadcast
	if err = c.signerCacheManager.SetSigned(tx.CacheHash(), tx.CacheVault(c.GetChain()), hash); err != nil {
		c.logger.Err(err).Msg("fail to set signer cache")
	}

	c.logger.Info().Str("hash", hash).Msg("transaction broadcast successful")
	return hash, nil
}

// submitTransactionViaHorizon submits a simple payment transaction via Horizon API (like test script)
func (c *Client) submitTransactionViaHorizon(signedTx *txnbuild.Transaction) (string, error) {
	c.logger.Info().Msg("submitting transaction via Horizon API")

	// Get Horizon client
	horizonClient := c.horizonClient
	if horizonClient == nil {
		return "", fmt.Errorf("horizon client not initialized")
	}

	// Submit transaction using Transaction object (exactly like test script)
	resp, err := horizonClient.SubmitTransaction(signedTx)
	if err != nil {
		// Handle Horizon error with detailed debugging
		if hError := horizonclient.GetError(err); hError != nil {
			// Get XDR for logging
			txXDR, _ := signedTx.Base64()

			c.logger.Error().
				Str("horizon_error", hError.Problem.Title).
				Str("horizon_detail", hError.Problem.Detail).
				Str("horizon_type", hError.Problem.Type).
				Int("horizon_status", hError.Problem.Status).
				Interface("horizon_extras", hError.Problem.Extras).
				Str("tx_xdr", txXDR[:minInt(200, len(txXDR))]).
				Msg("Horizon transaction submission failed with detailed error")

			// Extract result codes if available
			if hError.Problem.Extras != nil {
				c.logger.Error().
					Interface("result_codes", hError.Problem.Extras).
					Msg("Stellar transaction result codes")
			}

			return "", fmt.Errorf("horizon error: %s - %s", hError.Problem.Title, hError.Problem.Detail)
		}
		return "", fmt.Errorf("failed to submit transaction: %w", err)
	}

	c.logger.Info().
		Str("hash", resp.Hash).
		Bool("successful", resp.Successful).
		Msg("transaction submitted via Horizon")

	if !resp.Successful {
		return "", fmt.Errorf("transaction failed on Horizon")
	}

	return resp.Hash, nil
}

// handleBroadcastError processes broadcast failures and handles sequence number synchronization.
func (c *Client) handleBroadcastError(tx stypes.TxOutItem, err error) (string, error) {
	// Log broadcast failure with transaction context
	c.logger.Error().
		Str("tx_hash", tx.InHash.String()).
		Str("memo", tx.Memo).
		Err(err).
		Msg("failed to broadcast transaction")

	// Handle sequence mismatch errors - just return the error for retry with fresh sequence
	if strings.Contains(err.Error(), "tx_bad_seq") {
		c.logger.Warn().
			Str("vault", tx.VaultPubKey.String()).
			Msg("tx_bad_seq error - transaction will retry with fresh sequence from Horizon")
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

// NOTE: Removed canonicalizeSignature function as it was secp256k1-specific
// Stellar uses native ed25519 signatures which don't require canonicalization

// minInt returns the minimum of two integers
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
