package stellar

import (
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	tssp "gitlab.com/thorchain/thornode/v3/bifrost/tss/go-tss/tss"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/strkey"
	"github.com/stellar/go/txnbuild"

	"github.com/stellar/go/xdr"
	"gitlab.com/thorchain/thornode/v3/bifrost/blockscanner"
	"gitlab.com/thorchain/thornode/v3/bifrost/metrics"
	"gitlab.com/thorchain/thornode/v3/bifrost/pkg/chainclients/shared/runners"
	"gitlab.com/thorchain/thornode/v3/bifrost/pkg/chainclients/shared/signercache"
	"gitlab.com/thorchain/thornode/v3/bifrost/thorclient"
	stypes "gitlab.com/thorchain/thornode/v3/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/v3/bifrost/tss"
	"gitlab.com/thorchain/thornode/v3/common"
	"gitlab.com/thorchain/thornode/v3/config"
	"gitlab.com/thorchain/thornode/v3/constants"
)

// Client is a structure to sign and broadcast tx to Stellar chain used by signer mostly
type Client struct {
	logger              zerolog.Logger
	cfg                 config.BifrostChainConfiguration
	tssKeyManager       *tss.KeySign
	thorchainBridge     thorclient.ThorchainBridge
	storage             *blockscanner.BlockScannerStorage
	blockScanner        *blockscanner.BlockScanner
	signerCacheManager  *signercache.CacheManager
	stellarScanner      *StellarBlockScanner
	globalSolvencyQueue chan stypes.Solvency
	wg                  *sync.WaitGroup
	stopchan            chan struct{}
	horizonClient       *horizonclient.Client
	networkPassphrase   string
}

// NewClient creates a new instance of a Stellar chain client
func NewClient(
	thorKeys *thorclient.Keys,
	cfg config.BifrostChainConfiguration,
	server *tssp.TssServer,
	thorchainBridge thorclient.ThorchainBridge,
	m *metrics.Metrics,
) (*Client, error) {
	logger := log.With().Str("module", cfg.ChainID.String()).Logger()

	tssKm, err := tss.NewKeySign(server, thorchainBridge)
	if err != nil {
		return nil, fmt.Errorf("fail to create tss signer: %w", err)
	}

	if thorchainBridge == nil {
		return nil, errors.New("thorchain bridge is nil")
	}

	// Determine network passphrase based on configuration
	networkPassphrase := network.PublicNetworkPassphrase
	if cfg.ChainNetwork == "testnet" {
		networkPassphrase = network.TestNetworkPassphrase
	}

	// Create Horizon client
	horizonClient := horizonclient.DefaultPublicNetClient
	if cfg.RPCHost != "" {
		horizonClient = &horizonclient.Client{
			HorizonURL: cfg.RPCHost,
		}
	}

	c := &Client{
		logger:            logger,
		cfg:               cfg,
		tssKeyManager:     tssKm,
		thorchainBridge:   thorchainBridge,
		wg:                &sync.WaitGroup{},
		stopchan:          make(chan struct{}),
		horizonClient:     horizonClient,
		networkPassphrase: networkPassphrase,
	}

	var path string // if not set later, will in memory storage
	if len(c.cfg.BlockScanner.DBPath) > 0 {
		path = fmt.Sprintf("%s/%s", c.cfg.BlockScanner.DBPath, c.cfg.BlockScanner.ChainID)
	}
	c.storage, err = blockscanner.NewBlockScannerStorage(path, c.cfg.ScannerLevelDB)
	if err != nil {
		return nil, fmt.Errorf("fail to create scan storage: %w", err)
	}

	c.stellarScanner, err = NewStellarBlockScanner(
		c.cfg.RPCHost,
		c.cfg.BlockScanner,
		c.storage,
		c.thorchainBridge,
		m,
		c.ReportSolvency,
		c.horizonClient,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create stellar scanner: %w", err)
	}

	c.blockScanner, err = blockscanner.NewBlockScanner(c.cfg.BlockScanner, c.storage, m, c.thorchainBridge, c.stellarScanner)
	if err != nil {
		return nil, fmt.Errorf("failed to create block scanner: %w", err)
	}

	signerCacheManager, err := signercache.NewSignerCacheManager(c.storage.GetInternalDb())
	if err != nil {
		return nil, fmt.Errorf("fail to create signer cache manager")
	}
	c.signerCacheManager = signerCacheManager

	return c, nil
}

// Start Stellar chain client
func (c *Client) Start(globalTxsQueue chan stypes.TxIn, globalErrataQueue chan stypes.ErrataBlock, globalSolvencyQueue chan stypes.Solvency, globalNetworkFeeQueue chan common.NetworkFee) {
	c.globalSolvencyQueue = globalSolvencyQueue
	c.stellarScanner.globalNetworkFeeQueue = globalNetworkFeeQueue
	c.tssKeyManager.Start()
	c.blockScanner.Start(globalTxsQueue, globalNetworkFeeQueue)
	c.wg.Add(1)
	go runners.SolvencyCheckRunner(c.GetChain(), c, c.thorchainBridge, c.stopchan, c.wg, constants.ThorchainBlockTime)
}

// Stop Stellar chain client
func (c *Client) Stop() {
	c.tssKeyManager.Stop()
	c.blockScanner.Stop()
	close(c.stopchan)
	c.wg.Wait()
}

// GetConfig return the configuration used by Stellar chain client
func (c *Client) GetConfig() config.BifrostChainConfiguration {
	return c.cfg
}

func (c *Client) IsBlockScannerHealthy() bool {
	return c.blockScanner.IsHealthy()
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
	// Implementation would query Stellar for the latest transaction
	// For now, return empty values
	return "", "", nil
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
			assetMapping, found = GetAssetByStellarAsset("native", "", "")
		} else {
			// For non-native assets
			assetCode := balance.Asset.Code
			assetIssuer := balance.Asset.Issuer
			assetMapping, found = GetAssetByStellarAsset(balance.Asset.Type, assetCode, assetIssuer)
		}

		if !found {
			continue // Skip unsupported assets
		}

		// Convert balance using asset mapping
		coin, err := assetMapping.ConvertToTHORChainAmount(balance.Balance)
		if err != nil {
			c.logger.Error().Err(err).Str("asset", assetMapping.THORChainAsset.String()).Msg("fail to convert balance")
			continue
		}

		if !coin.Amount.IsZero() {
			coins = append(coins, coin)
		}
	}

	account = common.NewAccount(0, 0, coins, false)
	return account, nil
}

// processOutboundTx processes an outbound transaction
func (c *Client) processOutboundTx(tx stypes.TxOutItem) (*txnbuild.Payment, error) {
	// Check if we have any coins to send
	if len(tx.Coins) == 0 {
		return nil, fmt.Errorf("no coins to send")
	}

	// Support single coin transactions
	coin := tx.Coins[0]

	// Find the asset mapping for this THORChain asset
	assetMapping, found := GetAssetByTHORChainAsset(coin.Asset)
	if !found {
		return nil, fmt.Errorf("unsupported asset: %s", coin.Asset)
	}

	// Convert amount from THORChain units to Stellar units
	stellarAmount := assetMapping.ConvertFromTHORChainAmount(coin.Amount)

	// Create payment operation with the appropriate asset
	payment := &txnbuild.Payment{
		Destination: tx.ToAddress.String(),
		Amount:      stellarAmount,
		Asset:       assetMapping.ToStellarAsset(),
	}

	return payment, nil
}

// SignTx signs a transaction
func (c *Client) SignTx(tx stypes.TxOutItem, thorchainHeight int64) (signedTx, checkpoint []byte, _ *stypes.TxInItem, err error) {
	defer func() {
		if err != nil {
			var keysignError tss.KeysignError
			if errors.As(err, &keysignError) {
				if len(keysignError.Blame.BlameNodes) == 0 {
					c.logger.Err(err).Msg("TSS doesn't know which node to blame")
					return
				}

				// key sign error forward the keysign blame to thorchain
				var txID common.TxID
				txID, err = c.thorchainBridge.PostKeysignFailure(keysignError.Blame, thorchainHeight, tx.Memo, tx.Coins, tx.VaultPubKey)
				if err != nil {
					c.logger.Err(err).Msg("fail to post keysign failure to THORChain")
					return
				}
				c.logger.Info().Str("tx_id", txID.String()).Msgf("post keysign failure to thorchain")
			}
			c.logger.Err(err).Msg("failed to sign tx")
			return
		}
	}()

	if c.signerCacheManager.HasSigned(tx.CacheHash()) {
		c.logger.Info().Interface("tx", tx).Msg("transaction already signed, ignoring...")
		return nil, nil, nil, nil
	}

	// Check if we have any coins to send
	if len(tx.Coins) == 0 {
		return nil, nil, nil, fmt.Errorf("no coins to send")
	}

	// Support single coin transactions
	coin := tx.Coins[0]

	// Find the asset mapping for this THORChain asset
	_, found := GetAssetByTHORChainAsset(coin.Asset)
	if !found {
		return nil, nil, nil, fmt.Errorf("unsupported asset: %s", coin.Asset)
	}

	payment, err := c.processOutboundTx(tx)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to process outbound tx: %w", err)
	}

	sourceAddress := c.GetAddress(tx.VaultPubKey)

	// Get account info to get sequence number
	accountRequest := horizonclient.AccountRequest{AccountID: sourceAddress}
	sourceAccount, err := c.horizonClient.AccountDetail(accountRequest)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to get source account: %w", err)
	}

	// Build transaction
	txParams := txnbuild.TransactionParams{
		SourceAccount:        &sourceAccount,
		IncrementSequenceNum: true,
		Operations:           []txnbuild.Operation{payment},
		BaseFee:              txnbuild.MinBaseFee,
		Memo:                 txnbuild.MemoText(tx.Memo),
		Preconditions: txnbuild.Preconditions{
			TimeBounds: txnbuild.NewInfiniteTimeout(),
		},
	}

	transaction, err := txnbuild.NewTransaction(txParams)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to build transaction: %w", err)
	}

	// Get transaction hash for signing
	txHash, err := transaction.Hash(c.networkPassphrase)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to get transaction hash: %w", err)
	}

	// Sign with TSS
	signature, _, err := c.tssKeyManager.RemoteSign(txHash[:], tx.VaultPubKey.String())
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to sign transaction with TSS: %w", err)
	}

	if signature == nil {
		// This node was not selected for signing
		return nil, nil, nil, nil
	}

	// Convert TSS signature to Stellar signature format
	stellarSig := make([]byte, 64)
	copy(stellarSig, signature)

	// Create decorated signature manually
	addr := c.GetAddress(tx.VaultPubKey)
	decoded, err := strkey.Decode(strkey.VersionByteAccountID, addr)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to decode address: %w", err)
	}

	var hint [4]byte
	copy(hint[:], decoded[len(decoded)-4:])

	decoratedSig := xdr.DecoratedSignature{
		Hint:      xdr.SignatureHint(hint),
		Signature: xdr.Signature(stellarSig),
	}

	// Add signature to transaction using the correct method
	signedTransaction, err := transaction.AddSignatureDecorated(decoratedSig)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to add signature to transaction: %w", err)
	}

	// Get signed transaction XDR
	txeBase64, err := signedTransaction.Base64()
	if err != nil {
		return nil, nil, nil, fmt.Errorf("fail to encode signed transaction: %w", err)
	}

	// Cache the signed transaction
	if err := c.signerCacheManager.SetSigned(tx.CacheHash(), txeBase64, tx.VaultPubKey.String()); err != nil {
		c.logger.Err(err).Msg("fail to mark transaction as signed")
	}

	return []byte(txeBase64), nil, nil, nil
}

// BroadcastTx broadcasts a transaction to the Stellar network
func (c *Client) BroadcastTx(tx stypes.TxOutItem, txBytes []byte) (string, error) {
	txeBase64 := string(txBytes)

	// Submit transaction
	resp, err := c.horizonClient.SubmitTransactionXDR(txeBase64)
	if err != nil {
		return "", fmt.Errorf("fail to broadcast transaction: %w", err)
	}

	return resp.Hash, nil
}

// ConfirmationCountReady returns true if the confirmation count is ready
func (c *Client) ConfirmationCountReady(txIn stypes.TxIn) bool {
	return c.GetConfirmationCount(txIn) >= c.cfg.BlockScanner.ObservationFlexibilityBlocks
}

// GetConfirmationCount returns the confirmation count for a transaction
func (c *Client) GetConfirmationCount(txIn stypes.TxIn) int64 {
	// For Stellar, we can consider transactions final after a few ledgers
	// This is a simplified implementation
	return 1
}

// ReportSolvency reports solvency to THORChain
func (c *Client) ReportSolvency(blockHeight int64) error {
	if !c.ShouldReportSolvency(blockHeight) {
		return nil
	}

	// Get all asgard vaults
	asgardVaults, err := c.thorchainBridge.GetAsgards()
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
	case <-time.After(constants.ThorchainBlockTime):
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
	// Implementation for handling observed transactions
	// This would typically update internal state or caches
	c.logger.Debug().
		Str("tx_hash", txIn.Tx).
		Int64("block_height", blockHeight).
		Msg("observed tx in")
}
