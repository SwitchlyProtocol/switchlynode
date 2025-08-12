package stellar

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/base"
	"github.com/stellar/go/protocols/horizon/operations"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/config"

	"sync"
	"sync/atomic"

	"os"

	sdkmath "cosmossdk.io/math"
)

// SolvencyReporter is to report solvency info to THORNode
type SolvencyReporter func(int64) error

const (
	// FeeUpdatePeriodBlocks is the block interval at which we report fee changes.
	FeeUpdatePeriodBlocks = 20

	// FeeCacheTransactions is the number of transactions over which we compute an average
	// (mean) fee price to use for outbound transactions.
	FeeCacheTransactions = 200

	// defaultFeeMultiplier is the default multiplier for fee calculation
	defaultFeeMultiplier = 1.5
	// baseFeeStroops is the base fee in stroops (1 XLM = 10^7 stroops)
	baseFeeStroops = 100
	// maxRetries is the maximum number of retries for API calls
	maxRetries = 3
	// retryDelay is the delay between retries
	retryDelay = time.Second * 2
)

var (
	ErrInvalidScanStorage = errors.New("scan storage is empty or nil")
	ErrInvalidMetrics     = errors.New("metrics is empty or nil")
	ErrEmptyTx            = errors.New("empty tx")
)

// StellarBlockScanner scans Stellar blocks for transactions
type StellarBlockScanner struct {
	cfg              config.BifrostBlockScannerConfiguration
	logger           zerolog.Logger
	db               blockscanner.ScannerStorage
	bridge           thorclient.ThorchainBridge
	solvencyReporter SolvencyReporter
	horizonClient    *horizonclient.Client
	sorobanRPCClient *SorobanRPCClient

	globalNetworkFeeQueue chan common.NetworkFee
	globalTxsQueue        chan types.TxIn

	// feeCache contains a rolling window of suggested fees.
	feeCache []sdkmath.Uint
	lastFee  sdkmath.Uint

	// routerAddressesCache caches router addresses to avoid repeated API calls
	routerAddressesCache     []string
	routerAddressesCacheTime time.Time
	routerAddressesCacheMu   sync.RWMutex

	// Health tracking for logging consistency with main blockscanner
	healthy *atomic.Bool

	// Continuous scanning control
	stopChan  chan struct{}
	wg        *sync.WaitGroup
	isRunning *atomic.Bool
}

// NewStellarBlockScanner create a new instance of BlockScan
func NewStellarBlockScanner(rpcHost string,
	cfg config.BifrostBlockScannerConfiguration,
	scanStorage blockscanner.ScannerStorage,
	bridge thorclient.ThorchainBridge,
	m *metrics.Metrics,
	solvencyReporter SolvencyReporter,
	horizonClient *horizonclient.Client,
	sorobanRPCClient *SorobanRPCClient,
	globalNetworkFeeQueue chan common.NetworkFee,
	globalTxsQueue chan types.TxIn,
) (*StellarBlockScanner, error) {
	if scanStorage == nil {
		return nil, errors.New("scanStorage is nil")
	}
	if m == nil {
		return nil, errors.New("metrics is nil")
	}
	if bridge == nil {
		return nil, errors.New("bridge is nil")
	}
	if horizonClient == nil {
		return nil, errors.New("horizonClient is nil")
	}
	if sorobanRPCClient == nil {
		return nil, errors.New("sorobanRPCClient is nil")
	}

	return &StellarBlockScanner{
		cfg:                   cfg,
		logger:                log.With().Str("module", "stellar").Logger(),
		db:                    scanStorage,
		bridge:                bridge,
		solvencyReporter:      solvencyReporter,
		horizonClient:         horizonClient,
		sorobanRPCClient:      sorobanRPCClient,
		globalNetworkFeeQueue: globalNetworkFeeQueue,
		globalTxsQueue:        globalTxsQueue,
		feeCache:              make([]sdkmath.Uint, 0, FeeCacheTransactions),
		lastFee:               sdkmath.NewUint(baseFeeStroops), // Initialize with base fee
		healthy:               &atomic.Bool{},
		stopChan:              make(chan struct{}),
		wg:                    &sync.WaitGroup{},
		isRunning:             &atomic.Bool{},
	}, nil
}

// GetHeight retrieves the current Stellar chain height from the Horizon API.
// It implements exponential backoff for rate limit errors and retries on failures.
func (c *StellarBlockScanner) GetHeight() (int64, error) {
	maxRetries := c.cfg.MaxHTTPRequestRetry
	baseDelay := c.cfg.BlockHeightDiscoverBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		rootInfo, err := c.horizonClient.Root()
		if err != nil {
			// Handle rate limit errors with exponential backoff
			if strings.Contains(err.Error(), "Rate Limit Exceeded") ||
				strings.Contains(err.Error(), "429") {
				if attempt < maxRetries {
					delay := time.Duration(1<<uint(attempt)) * baseDelay
					c.logger.Warn().
						Int("attempt", attempt+1).
						Int("max_retries", maxRetries+1).
						Dur("delay", delay).
						Msg("rate limit hit, retrying after delay")
					time.Sleep(delay)
					continue
				}
			}
			return 0, fmt.Errorf("failed to get root info: %w", err)
		}

		// Use HorizonSequence which represents the current chain height
		latestHeight := rootInfo.HorizonSequence
		return int64(latestHeight), nil
	}

	return 0, fmt.Errorf("max retries exceeded for getting chain height")
}

// findFirstAvailableLedger finds the first available ledger by checking from a starting point
func (c *StellarBlockScanner) findFirstAvailableLedger(startHeight int64) (int64, error) {
	currentHeight, err := c.GetHeight()
	if err != nil {
		return 0, fmt.Errorf("failed to get current height: %w", err)
	}

	// If starting from 0 or 1, try to find a reasonable starting point
	if startHeight <= 1 {
		// For Stellar networks, often the first few ledgers don't exist
		// Try starting from a later ledger (e.g., ledger 100) and work backwards
		maxStartingPoint := int64(100)
		if currentHeight < maxStartingPoint {
			maxStartingPoint = currentHeight
		}

		for height := maxStartingPoint; height >= 1; height-- {
			_, err := c.getTransactionsForLedger(height)
			if err == nil {
				c.logger.Info().
					Int64("first_available_ledger", height).
					Int64("requested_start", startHeight).
					Msg("found first available ledger")
				return height, nil
			}

			// If it's not a "Resource Missing" error, return the error
			if !strings.Contains(err.Error(), "Resource Missing") {
				return 0, fmt.Errorf("error checking ledger %d: %w", height, err)
			}
		}

		// If we can't find any available ledger, start from current height
		c.logger.Warn().
			Int64("current_height", currentHeight).
			Msg("could not find any available historical ledger, starting from current height")
		return currentHeight, nil
	}

	return startHeight, nil
}

// GetOptimalStartHeight determines the optimal starting height for Stellar scanning.
// Always returns the current latest block height to ensure scanning starts from the most recent state.
func (c *StellarBlockScanner) GetOptimalStartHeight() (int64, error) {
	currentHeight, err := c.GetHeight()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get current Stellar height")
		return 0, err
	}

	c.logger.Info().
		Int64("current_height", currentHeight).
		Msg("STELLAR: Starting from latest block height")

	return currentHeight, nil
}

// FetchMemPool returns an empty result since Stellar only processes finalized transactions.
func (c *StellarBlockScanner) FetchMemPool(height int64) (types.TxIn, error) {
	return types.TxIn{}, nil
}

// Start begins the continuous block scanning loop that runs every 60 seconds
func (c *StellarBlockScanner) Start() {
	if c.isRunning.Swap(true) {
		c.logger.Warn().Msg("Stellar block scanner is already running")
		return
	}

	c.logger.Info().Msg("Starting Stellar continuous block scanner")
	c.wg.Add(1)
	go c.continuousScanLoop()
}

// Stop stops the continuous block scanning loop
func (c *StellarBlockScanner) Stop() {
	if !c.isRunning.Swap(false) {
		c.logger.Warn().Msg("Stellar block scanner is not running")
		return
	}

	c.logger.Info().Msg("Stopping Stellar continuous block scanner")
	close(c.stopChan)
	c.wg.Wait()
	c.logger.Info().Msg("Stellar continuous block scanner stopped")
}

// continuousScanLoop runs every 60 seconds to continuously ingest new blocks
func (c *StellarBlockScanner) continuousScanLoop() {
	defer c.wg.Done()
	defer c.logger.Info().Msg("Stellar continuous scan loop stopped")

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	c.logger.Info().Msg("Stellar continuous scan loop started - scanning every 60 seconds")

	for {
		select {
		case <-c.stopChan:
			return
		case <-ticker.C:
			if err := c.scanNewBlocks(); err != nil {
				c.logger.Error().Err(err).Msg("Failed to scan new blocks in continuous loop")
			}
		}
	}
}

// scanNewBlocks scans all new blocks from the current stored position to the latest Stellar height
func (c *StellarBlockScanner) scanNewBlocks() error {
	// Get current stored position
	currentPos, err := c.db.GetScanPos()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get current scan position")
		return err
	}

	// Get current Stellar height
	chainHeight, err := c.GetHeight()
	if err != nil {
		c.logger.Error().Err(err).Msg("Failed to get current Stellar height")
		return err
	}

	if chainHeight <= currentPos {
		c.logger.Debug().Int64("current_pos", currentPos).Int64("chain_height", chainHeight).
			Msg("No new blocks to scan")
		return nil
	}

	// Calculate how many blocks to process
	blocksToProcess := chainHeight - currentPos
	if blocksToProcess > 50 { // Limit to prevent overwhelming the system
		blocksToProcess = 50
	}

	// Process blocks sequentially from current position + 1
	for i := int64(1); i <= blocksToProcess; i++ {
		blockHeight := currentPos + i

		// Fetch transactions for this block
		txIn, err := c.FetchTxs(blockHeight, chainHeight)
		if err != nil {
			c.logger.Error().Err(err).Int64("block_height", blockHeight).
				Msg("Failed to fetch transactions for block")
			continue
		}

		// Process transactions if any found
		if len(txIn.TxArray) > 0 {
			// Send transactions to global queue if needed
			c.globalTxsQueue <- txIn
		}

		// Update scan position to this block
		if err := c.db.SetScanPos(blockHeight); err != nil {
			c.logger.Error().Err(err).Int64("block_height", blockHeight).
				Msg("Failed to update scan position")
			return err
		}

		// Use standard blockscanner logging format to match other chains
		c.logger.Info().
			Str("chain", "XLM").
			Int64("block height", blockHeight).
			Int("txs", len(txIn.TxArray)).
			Int64("gap", chainHeight-blockHeight).
			Bool("healthy", c.healthy.Load()).
			Msg("scan block")

		// Update health status based on gap (similar to main blockscanner)
		// Consider 3 blocks or less behind as healthy
		if chainHeight-blockHeight <= 3 {
			c.healthy.Store(true)
		} else {
			c.healthy.Store(false)
		}
	}

	return nil
}

// HandleGapDetection performs Stellar-specific gap detection and position adjustment
// This is called by the Stellar client before starting the blockscanner
func (c *StellarBlockScanner) HandleGapDetection() error {
	// Get current chain height
	currentHeight, err := c.GetHeight()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get current Stellar height for gap detection")
		return err
	}

	// Get current stored position
	storedHeight, err := c.db.GetScanPos()
	if err != nil {
		c.logger.Debug().Err(err).Msg("no stored scan position found")
		storedHeight = 0
	}

	c.logger.Info().
		Int64("current_stellar_height", currentHeight).
		Int64("stored_position", storedHeight).
		Msg("STELLAR GAP DETECTION: Checking gap before scanner start")

	// Always check for large gaps and force position update if needed
	if storedHeight > 0 && currentHeight > 0 {
		gap := currentHeight - storedHeight
		const maxAcceptableGap = 100 // Very aggressive for Stellar

		if gap > maxAcceptableGap {
			// Force jump to recent height
			newStartHeight := currentHeight - 10 // Start just 10 blocks back

			c.logger.Info().
				Int64("stored_position", storedHeight).
				Int64("current_height", currentHeight).
				Int64("gap", gap).
				Int64("new_start_height", newStartHeight).
				Msg("STELLAR GAP DETECTION: FORCING position update due to large gap")

			// SMART PHD SOLUTION: Only update scan position if not disabled
			// FORCE update the stored position to skip the gap
			if err := c.db.SetScanPos(newStartHeight); err != nil {
				c.logger.Error().Err(err).Msg("CRITICAL: failed to update scan position")
				return err
			}

			c.logger.Info().
				Int64("new_position", newStartHeight).
				Msg("STELLAR GAP DETECTION: Successfully updated scan position")
		}
	}

	return nil
}

// FetchLastHeight retrieves the last processed height for Stellar scanning.
// Always returns the current chain height to ensure scanning starts from the latest block.
func (c *StellarBlockScanner) FetchLastHeight() (int64, error) {
	currentHeight, err := c.GetHeight()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get current Stellar height")
		return 1, nil // Fallback to start from ledger 1
	}

	c.logger.Info().
		Int64("current_stellar_height", currentHeight).
		Msg("STELLAR: Starting from latest block height")

	return currentHeight, nil
}

// GetNetworkFee returns the current chain network fee according to Bifrost.
func (c *StellarBlockScanner) GetNetworkFee() (transactionSize, transactionFeeRate uint64) {
	// Ensure we have a valid fee, fallback to base fee if needed
	if c.lastFee.IsZero() {
		c.lastFee = sdkmath.NewUint(baseFeeStroops)
	}
	return 1, c.lastFee.Uint64()
}

func (c *StellarBlockScanner) updateFeeCache(fee common.Coin) {
	// sanity check to ensure fee is non-zero
	err := fee.Valid()
	if err != nil {
		c.logger.Err(err).Interface("fee", fee).Msg("transaction with zero fee")
		return
	}

	// add the fee to our cache
	c.feeCache = append(c.feeCache, fee.Amount)

	// truncate fee prices older than our max cached transactions
	if len(c.feeCache) > FeeCacheTransactions {
		c.feeCache = c.feeCache[(len(c.feeCache) - FeeCacheTransactions):]
	}
}

func (c *StellarBlockScanner) averageFee() sdkmath.Uint {
	// avoid divide by zero
	if len(c.feeCache) == 0 {
		return sdkmath.NewUint(baseFeeStroops)
	}

	// compute mean
	sum := sdkmath.NewUint(0)
	for _, val := range c.feeCache {
		sum = sum.Add(val)
	}
	mean := sum.Quo(sdkmath.NewUint(uint64(len(c.feeCache))))

	return mean
}

func (c *StellarBlockScanner) updateFees(height int64) error {
	// post the gas fee over every cache period when we have a full gas cache
	if height%FeeUpdatePeriodBlocks == 0 && len(c.feeCache) == FeeCacheTransactions {
		avgFee := c.averageFee()

		// sanity check the fee is not zero
		if avgFee.IsZero() {
			return errors.New("suggested gas fee was zero")
		}

		// skip fee update if it has not changed
		if c.lastFee.Equal(avgFee) {
			return nil
		}

		// only send network fee if queue is initialized
		if c.globalNetworkFeeQueue != nil {
			c.globalNetworkFeeQueue <- common.NetworkFee{
				Chain:           c.cfg.ChainID,
				Height:          height,
				TransactionSize: 1,
				TransactionRate: avgFee.Uint64(),
			}

			c.logger.Info().
				Uint64("fee", avgFee.Uint64()).
				Int64("height", height).
				Msg("sent network fee to SwitchlyProtocol")
		} else {
			c.logger.Warn().
				Uint64("fee", avgFee.Uint64()).
				Int64("height", height).
				Msg("global network fee queue not initialized, skipping fee update")
		}

		c.lastFee = avgFee
	}

	return nil
}

func (c *StellarBlockScanner) processTxs(height int64, txs []horizon.Transaction) ([]*types.TxInItem, error) {
	var txIn []*types.TxInItem

	for _, tx := range txs {
		if !tx.Successful {
			continue
		}

		// Process each operation in the transaction with retry logic
		var operationsPage operations.OperationsPage
		err := c.retryHorizonCall("operations", func() error {
			var err error
			operationsPage, err = c.horizonClient.Operations(horizonclient.OperationRequest{
				ForTransaction: tx.Hash,
			})
			return err
		})
		if err != nil {
			c.logger.Error().Err(err).Str("tx_hash", tx.Hash).Msg("fail to get operations for transaction")
			continue
		}

		for _, op := range operationsPage.Embedded.Records {
			txInItem, err := c.processOperation(tx, op, height)
			if err != nil {
				c.logger.Error().Err(err).Str("tx_hash", tx.Hash).Msg("fail to process operation")
				continue
			}
			if txInItem != nil {
				txIn = append(txIn, txInItem)
			}
		}

		// Update fee cache
		feeAmount := cosmos.NewUintFromString(strconv.FormatInt(tx.FeeCharged, 10))
		fee := common.NewCoin(common.XLMAsset, feeAmount)
		c.updateFeeCache(fee)
	}

	return txIn, nil
}

func (c *StellarBlockScanner) processOperation(tx horizon.Transaction, op operations.Operation, height int64) (*types.TxInItem, error) {
	switch operation := op.(type) {
	case operations.Payment:
		return c.processPaymentOperation(tx, operation, height)
	case operations.InvokeHostFunction:
		return c.processInvokeHostFunctionOperation(tx, operation, height)
	case operations.CreateAccount:
		return c.processCreateAccountOperation(tx, operation, height)
	default:
		// Skip other operation types
		return nil, nil
	}
}

// processPaymentOperation processes payment operations
func (c *StellarBlockScanner) processPaymentOperation(tx horizon.Transaction, payment operations.Payment, height int64) (*types.TxInItem, error) {
	// Check if this is a payment to one of our vaults
	// For now, we'll process all payments and let the observer filter them
	// TODO: Add proper vault address checking when the bridge method is available

	// Parse the asset and amount
	coin, err := c.parseAssetAndAmount(payment.Asset, payment.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to parse asset and amount: %w", err)
	}

	// Get the vault public key for XLM chain
	vaultPubKey, err := c.getVaultPubKeyForXLM()
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Str("from", payment.From).
			Str("to", payment.To).
			Msg("failed to get vault public key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify that we have a valid vault pub key
	if vaultPubKey.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", tx.Hash).
			Str("from", payment.From).
			Str("to", payment.To).
			Msg("received empty vault pub key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify we can get a valid XLM address from this vault pub key
	xlmAddress, err := vaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("failed to get XLM address from vault pub key, skipping transaction")
		return nil, nil
	}

	if xlmAddress.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", tx.Hash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("vault pub key returned empty XLM address, skipping transaction")
		return nil, nil
	}

	c.logger.Debug().
		Str("tx_hash", tx.Hash).
		Str("vault_pubkey", vaultPubKey.String()).
		Str("xlm_address", xlmAddress.String()).
		Msg("processing payment with valid vault pub key")

	// Create TxInItem
	txInItem := &types.TxInItem{
		BlockHeight:         height,
		Tx:                  tx.Hash,
		Sender:              payment.From,
		To:                  payment.To,
		Coins:               common.Coins{coin},
		Memo:                tx.Memo,
		ObservedVaultPubKey: vaultPubKey,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(uint64(tx.FeeCharged))},
		},
	}

	return txInItem, nil
}

// processInvokeHostFunctionOperation processes Stellar smart contract invocations
// This is where we detect and process router contract events
func (c *StellarBlockScanner) processInvokeHostFunctionOperation(tx horizon.Transaction, operation operations.InvokeHostFunction, height int64) (*types.TxInItem, error) {
	// Get router addresses from bridge configuration
	routerAddresses, err := c.getRouterAddresses()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get router addresses")
		return nil, nil
	}

	if len(routerAddresses) == 0 {
		c.logger.Debug().Msg("no router addresses configured, skipping router event processing")
		return nil, nil
	}

	// Check if this operation involves any router contract
	isRouterCall := false
	var routerAddr string
	for _, addr := range routerAddresses {
		if c.isRouterOperation(operation, addr) {
			isRouterCall = true
			routerAddr = addr
			break
		}
	}

	if !isRouterCall {
		return nil, nil
	}

	// Get router events from Soroban RPC (if available)
	if c.sorobanRPCClient == nil {
		c.logger.Debug().Msg("soroban RPC client not available, skipping router event processing")
		return nil, nil
	}

	routerEvents, err := c.sorobanRPCClient.GetRouterEvents(context.Background(), uint32(height), routerAddresses)
	if err != nil {
		c.logger.Error().
			Err(err).
			Int64("height", height).
			Str("tx_hash", tx.Hash).
			Msg("failed to get router events from Soroban RPC")
		return nil, nil
	}

	// Process router events for this transaction
	for _, event := range routerEvents {
		if event.TransactionHash == tx.Hash {
			c.logger.Info().
				Str("event_type", event.Type).
				Str("tx_hash", event.TransactionHash).
				Str("router_address", routerAddr).
				Int64("height", height).
				Msg("processing router event")

			return c.processRouterEvent(event, height)
		}
	}

	// If no events found but this is a router call, log it for debugging
	c.logger.Debug().
		Str("router_address", routerAddr).
		Str("tx_hash", tx.Hash).
		Int64("height", height).
		Msg("router contract call detected but no events found")

	return nil, nil
}

// processCreateAccountOperation processes account creation operations
func (c *StellarBlockScanner) processCreateAccountOperation(tx horizon.Transaction, operation operations.CreateAccount, height int64) (*types.TxInItem, error) {
	// Check if this is creating an account for one of our vaults
	// For now, we'll process all account creations and let the observer filter them
	// TODO: Add proper vault address checking when the bridge method is available

	// Parse the starting balance
	startingBalance, err := strconv.ParseFloat(operation.StartingBalance, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse starting balance: %w", err)
	}

	// Get the vault public key for XLM chain
	vaultPubKey, err := c.getVaultPubKeyForXLM()
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Str("funder", operation.Funder).
			Str("account", operation.Account).
			Msg("failed to get vault public key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify that we have a valid vault pub key
	if vaultPubKey.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", tx.Hash).
			Str("funder", operation.Funder).
			Str("account", operation.Account).
			Msg("received empty vault pub key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify we can get a valid XLM address from this vault pub key
	xlmAddress, err := vaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("failed to get XLM address from vault pub key, skipping transaction")
		return nil, nil
	}

	if xlmAddress.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", tx.Hash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("vault pub key returned empty XLM address, skipping transaction")
		return nil, nil
	}

	c.logger.Debug().
		Str("tx_hash", tx.Hash).
		Str("vault_pubkey", vaultPubKey.String()).
		Str("xlm_address", xlmAddress.String()).
		Msg("processing account creation with valid vault pub key")

	// Create TxInItem for account creation
	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          tx.Hash,
		Sender:      operation.Funder,
		To:          operation.Account,
		Coins: common.Coins{
			common.NewCoin(common.XLMAsset, cosmos.NewUint(uint64(startingBalance*10000000))), // Convert XLM to stroops
		},
		Memo:                tx.Memo,
		ObservedVaultPubKey: vaultPubKey,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(uint64(tx.FeeCharged))},
		},
	}

	return txInItem, nil
}

// processRouterEvent processes a router event and converts it to TxInItem
// Based on the Stellar contract definition in chain/stellar/contracts/src/lib.rs
func (c *StellarBlockScanner) processRouterEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	eventType := strings.ToLower(event.Type)

	c.logger.Debug().
		Str("event_type", eventType).
		Str("tx_hash", event.TransactionHash).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Str("from", event.FromAddress).
		Str("to", event.ToAddress).
		Str("memo", event.Memo).
		Msg("Processing router event")

	switch eventType {
	case "deposit", "router_deposit":
		return c.processRouterDepositEvent(event, height)
	case "transfer_out", "router_transfer_out", "transferout":
		return c.processRouterTransferOutEvent(event, height)
	case "deposit_with_expiry", "depositwithexpiry":
		// Handle as regular deposit (same as deposit but with expiry check)
		return c.processRouterDepositEvent(event, height)
	case "transfer_allowance", "transferallowance":
		return c.processRouterTransferAllowanceEvent(event, height)
	case "return_vault_assets", "returnvaultassets", "vault_return":
		return c.processRouterReturnVaultAssetsEvent(event, height)
	default:
		c.logger.Debug().
			Str("event_type", eventType).
			Str("tx_hash", event.TransactionHash).
			Msg("unknown router event type - skipping")
		return nil, nil
	}
}

// processRouterDepositEvent processes router deposit events
// Aligns with DepositEvent struct in the Stellar contract
func (c *StellarBlockScanner) processRouterDepositEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Validate required fields for deposit event
	if event.Asset == "" {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("missing asset in router deposit event")
		return nil, nil
	}

	if event.Amount == "" || event.Amount == "0" {
		c.logger.Debug().
			Str("tx_hash", event.TransactionHash).
			Msg("deposit amount is 0, ignoring")
		return nil, nil
	}

	// Find the asset mapping
	mapping, found := GetAssetByAddress(event.Asset)
	if !found {
		c.logger.Warn().
			Str("asset_address", event.Asset).
			Msg("unsupported asset in router deposit event")
		return nil, nil
	}

	// Convert amount
	coin, err := mapping.ConvertToSwitchlyProtocolAmount(event.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	// For deposit events, the 'from' is the user and 'to' is the vault
	fromAddr := event.FromAddress
	if fromAddr == "" {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("missing from address in router deposit event")
		return nil, nil
	}

	toAddr := event.ToAddress
	if toAddr == "" {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("missing to address (vault) in router deposit event")
		return nil, nil
	}

	// Get the vault public key for XLM chain
	vaultPubKey, err := c.getVaultPubKeyForXLM()
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", event.TransactionHash).
			Msg("failed to get vault public key for XLM chain")
		return nil, nil
	}

	// Verify that we have a valid vault pub key
	if vaultPubKey.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("received empty vault pub key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify we can get a valid XLM address from this vault pub key
	xlmAddress, err := vaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", event.TransactionHash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("failed to get XLM address from vault pub key, skipping transaction")
		return nil, nil
	}

	if xlmAddress.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("vault pub key returned empty XLM address, skipping transaction")
		return nil, nil
	}

	c.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("vault_pubkey", vaultPubKey.String()).
		Str("xlm_address", xlmAddress.String()).
		Msg("processing router deposit with valid vault pub key")

	// Create TxInItem
	txInItem := &types.TxInItem{
		BlockHeight:         height,
		Tx:                  event.TransactionHash,
		Sender:              fromAddr,
		To:                  toAddr, // This should be the vault address
		Coins:               common.Coins{coin},
		Memo:                event.Memo,
		ObservedVaultPubKey: vaultPubKey,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(baseFeeStroops)},
		},
	}

	c.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Str("from", fromAddr).
		Str("to", toAddr).
		Str("asset", mapping.SwitchlyAsset.String()).
		Str("amount", coin.Amount.String()).
		Str("memo", event.Memo).
		Msg("processed router deposit event")

	return txInItem, nil
}

// processRouterTransferOutEvent processes router transfer out events
// Aligns with TransferOutEvent struct in the Stellar contract
func (c *StellarBlockScanner) processRouterTransferOutEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Transfer out events are outbound transactions, not inbound
	// We don't generate TxInItems for these, but we log them for monitoring
	c.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Str("vault", event.FromAddress).
		Str("to", event.ToAddress).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Str("memo", event.Memo).
		Msg("router transfer out event detected (outbound transaction)")

	return nil, nil
}

// processRouterTransferAllowanceEvent processes router transfer allowance events (vault rotation)
// Aligns with TransferAllowanceEvent struct in the Stellar contract
func (c *StellarBlockScanner) processRouterTransferAllowanceEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Validate required fields for transfer allowance event
	if event.Asset == "" {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("missing asset in router transfer allowance event")
		return nil, nil
	}

	if event.Amount == "" || event.Amount == "0" {
		c.logger.Debug().
			Str("tx_hash", event.TransactionHash).
			Msg("transfer allowance amount is 0, ignoring")
		return nil, nil
	}

	// Find the asset mapping
	mapping, found := GetAssetByAddress(event.Asset)
	if !found {
		c.logger.Warn().
			Str("asset_address", event.Asset).
			Msg("unsupported asset in router transfer allowance event")
		return nil, nil
	}

	// Convert amount
	coin, err := mapping.ConvertToSwitchlyProtocolAmount(event.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	// For transfer allowance events, the 'from' is the old vault and 'to' is the new vault
	oldVault := event.FromAddress
	if oldVault == "" {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("missing old vault address in router transfer allowance event")
		return nil, nil
	}

	newVault := event.ToAddress
	if newVault == "" {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("missing new vault address in router transfer allowance event")
		return nil, nil
	}

	// Get the vault public key for XLM chain
	vaultPubKey, err := c.getVaultPubKeyForXLM()
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", event.TransactionHash).
			Msg("failed to get vault public key for XLM chain")
		return nil, nil
	}

	// Verify that we have a valid vault pub key
	if vaultPubKey.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Msg("received empty vault pub key for XLM chain, skipping transaction")
		return nil, nil
	}

	// Verify we can get a valid XLM address from this vault pub key
	xlmAddress, err := vaultPubKey.GetAddress(common.StellarChain)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", event.TransactionHash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("failed to get XLM address from vault pub key, skipping transaction")
		return nil, nil
	}

	if xlmAddress.IsEmpty() {
		c.logger.Warn().
			Str("tx_hash", event.TransactionHash).
			Str("vault_pubkey", vaultPubKey.String()).
			Msg("vault pub key returned empty XLM address, skipping transaction")
		return nil, nil
	}

	c.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("vault_pubkey", vaultPubKey.String()).
		Str("xlm_address", xlmAddress.String()).
		Msg("processing router transfer allowance with valid vault pub key")

	// Create TxInItem for vault rotation
	txInItem := &types.TxInItem{
		BlockHeight:         height,
		Tx:                  event.TransactionHash,
		Sender:              oldVault, // Old vault
		To:                  newVault, // New vault
		Coins:               common.Coins{coin},
		Memo:                event.Memo,
		ObservedVaultPubKey: vaultPubKey,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(baseFeeStroops)},
		},
	}

	c.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Str("old_vault", oldVault).
		Str("new_vault", newVault).
		Str("asset", mapping.SwitchlyAsset.String()).
		Str("amount", coin.Amount.String()).
		Str("memo", event.Memo).
		Msg("processed router transfer allowance event (vault rotation)")

	return txInItem, nil
}

// processRouterReturnVaultAssetsEvent processes router return vault assets events
// Aligns with VaultReturnEvent struct in the Stellar contract
func (c *StellarBlockScanner) processRouterReturnVaultAssetsEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// For return vault assets events, we need to handle multiple assets
	// This is typically a batch operation returning multiple assets from old vault to new vault
	if event.Asset == "" && event.Amount == "" {
		c.logger.Debug().
			Str("tx_hash", event.TransactionHash).
			Msg("return vault assets event has no specific asset/amount, treating as informational")
		return nil, nil
	}

	// If we have specific asset and amount, process it like a transfer allowance
	if event.Asset != "" && event.Amount != "" {
		return c.processRouterTransferAllowanceEvent(event, height)
	}

	c.logger.Info().
		Str("tx_hash", event.TransactionHash).
		Str("old_vault", event.FromAddress).
		Str("new_vault", event.ToAddress).
		Str("memo", event.Memo).
		Msg("router return vault assets event detected (batch vault transfer)")

	return nil, nil
}

// getOperationsForTransaction gets all operations for a transaction
func (c *StellarBlockScanner) getOperationsForTransaction(txHash string) ([]operations.Operation, error) {
	var allOps []operations.Operation

	err := c.retryHorizonCall("get_operations", func() error {
		opsPage, err := c.horizonClient.Operations(horizonclient.OperationRequest{
			ForTransaction: txHash,
			Limit:          200,
		})
		if err != nil {
			return err
		}

		allOps = opsPage.Embedded.Records
		return nil
	})

	return allOps, err
}

// getRouterAddresses retrieves router contract addresses from the bridge
func (c *StellarBlockScanner) getRouterAddresses() ([]string, error) {
	c.routerAddressesCacheMu.RLock()
	if time.Since(c.routerAddressesCacheTime) < 10*time.Minute { // Cache for 10 minutes
		c.routerAddressesCacheMu.RUnlock()
		return c.routerAddressesCache, nil
	}
	c.routerAddressesCacheMu.RUnlock()

	// First, try to get router addresses from the bridge (production behavior)
	contracts, err := c.bridge.GetContractAddress()
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get contract addresses from bridge")
		// Don't return error, try fallback mechanisms
	} else if len(contracts) > 0 {
		// Extract Stellar router addresses from the contracts
		var routerAddresses []string
		for _, contract := range contracts {
			if addr, ok := contract.Contracts[common.StellarChain]; ok && !addr.IsEmpty() {
				routerAddr := addr.String()
				// Validate that this is a valid Stellar contract address
				if c.isValidStellarContractAddress(routerAddr) {
					routerAddresses = append(routerAddresses, routerAddr)
				} else {
					c.logger.Warn().
						Str("router_address", routerAddr).
						Msg("invalid Stellar contract address format from bridge")
				}
			}
		}

		// If we found router addresses from the bridge, use them
		if len(routerAddresses) > 0 {
			c.logger.Info().
				Int("router_count", len(routerAddresses)).
				Msg("retrieved router addresses from bridge")

			c.routerAddressesCacheMu.Lock()
			c.routerAddressesCache = routerAddresses
			c.routerAddressesCacheTime = time.Now()
			c.routerAddressesCacheMu.Unlock()

			return routerAddresses, nil
		}
	}

	// Fallback: Check for environment variable configuration
	// This allows manual configuration for testing/mocknet environments
	envRouterAddress := os.Getenv("BIFROST_CHAINS_XLM_ROUTER_ADDRESS")
	if envRouterAddress != "" {
		// Split multiple addresses if comma-separated
		addresses := strings.Split(envRouterAddress, ",")
		var validAddresses []string

		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if c.isValidStellarContractAddress(addr) {
				validAddresses = append(validAddresses, addr)
			} else {
				c.logger.Warn().
					Str("address", addr).
					Msg("invalid router address in environment variable")
			}
		}

		if len(validAddresses) > 0 {
			c.logger.Info().
				Int("router_count", len(validAddresses)).
				Msg("using router addresses from environment variable")

			c.routerAddressesCacheMu.Lock()
			c.routerAddressesCache = validAddresses
			c.routerAddressesCacheTime = time.Now()
			c.routerAddressesCacheMu.Unlock()

			return validAddresses, nil
		}
	}

	// No router addresses found from any source
	c.logger.Warn().Msg("no router addresses found from bridge or environment - block scanning will be limited")

	c.routerAddressesCacheMu.Lock()
	c.routerAddressesCache = []string{}
	c.routerAddressesCacheTime = time.Now()
	c.routerAddressesCacheMu.Unlock()

	return []string{}, nil
}

// getEnvironmentRouterAddress checks for router addresses configured via environment variables
func (c *StellarBlockScanner) getEnvironmentRouterAddress() string {
	// Check for environment variable BIFROST_CHAINS_XLM_ROUTER_ADDRESS
	// This allows operators to configure router addresses for testing/mocknet
	if routerAddr := os.Getenv("BIFROST_CHAINS_XLM_ROUTER_ADDRESS"); routerAddr != "" {
		if c.isValidStellarContractAddress(routerAddr) {
			return routerAddr
		}
		c.logger.Warn().
			Str("router_address", routerAddr).
			Msg("invalid router address in environment variable")
	}

	// Check for environment variable BIFROST_CHAINS_XLM_ROUTER_ADDRESSES (comma-separated)
	if routerAddrs := os.Getenv("BIFROST_CHAINS_XLM_ROUTER_ADDRESSES"); routerAddrs != "" {
		addresses := strings.Split(routerAddrs, ",")
		for _, addr := range addresses {
			addr = strings.TrimSpace(addr)
			if c.isValidStellarContractAddress(addr) {
				return addr // Return first valid address
			}
		}
		c.logger.Warn().
			Str("router_addresses", routerAddrs).
			Msg("no valid router addresses found in environment variable")
	}

	return ""
}

// isValidStellarContractAddress validates if an address is a valid Stellar contract address
func (c *StellarBlockScanner) isValidStellarContractAddress(addr string) bool {
	// Stellar contract addresses start with 'C' and are 56 characters long
	if len(addr) != 56 || !strings.HasPrefix(addr, "C") {
		return false
	}

	// Check if it's valid base32 encoding (A-Z, 2-7)
	for _, char := range addr[1:] {
		if !((char >= 'A' && char <= 'Z') || (char >= '2' && char <= '7')) {
			return false
		}
	}

	return true
}

// isRouterOperation checks if an operation involves a router contract
func (c *StellarBlockScanner) isRouterOperation(op operations.InvokeHostFunction, routerAddr string) bool {
	// Check if the operation involves a contract call to our router
	if routerAddr == "" {
		return false
	}

	// For Stellar, we need to check if this is a contract invocation operation
	// The operation type should be "invoke_host_function" and the function should be "HostFunctionTypeHostFunctionTypeInvokeContract"
	if op.Function == "HostFunctionTypeHostFunctionTypeInvokeContract" {
		// This is a contract call operation
		// We need to check if the contract address in the parameters matches our router address
		// The first parameter should be the contract address
		if len(op.Parameters) > 0 && op.Parameters[0].Type == "Address" {
			// For now, we'll assume any contract call operation could be a router operation
			// since we can't easily decode the XDR address format in this context
			// A more robust solution would be to parse the XDR data properly
			return true
		}
	}

	// Check for specific function names as a fallback
	lowerFunction := strings.ToLower(op.Function)
	routerFunctions := []string{
		"deposit",
		"deposit_with_expiry",
		"transfer_out",
		"transfer_allowance",
		"return_vault_assets",
		"router_deposit",
		"router_transfer",
		"vault_transfer",
		"switchly_deposit",
		"switchly_transfer",
	}

	// Check if the function name exactly matches one of our known router functions
	for _, routerFunc := range routerFunctions {
		if lowerFunction == routerFunc {
			return true
		}
	}

	return false
}

// parseAssetAndAmount parses a Stellar asset and amount into a SwitchlyProtocol coin
func (c *StellarBlockScanner) parseAssetAndAmount(asset base.Asset, amountStr string) (common.Coin, error) {
	// Parse amount
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return common.Coin{}, fmt.Errorf("failed to parse amount: %w", err)
	}

	// Convert to stroops (1 XLM = 10^7 stroops)
	amountStroops := uint64(amount * 10000000)

	// Determine asset
	var thorAsset common.Asset
	if asset.Type == "native" {
		thorAsset = common.XLMAsset
	} else {
		// Handle other assets
		thorAsset = common.Asset{
			Chain:  common.StellarChain,
			Symbol: common.Symbol(asset.Code),
			Ticker: common.Ticker(asset.Code),
		}
	}

	return common.NewCoin(thorAsset, cosmos.NewUint(amountStroops)), nil
}

// FetchTxs retrieves transactions for a specific block height from the Stellar network.
// It only processes transactions that involve router contracts, following the Ethereum pattern.
func (c *StellarBlockScanner) FetchTxs(height, chainHeight int64) (types.TxIn, error) {
	var txIn types.TxIn
	txIn.Chain = c.cfg.ChainID
	txIn.Filtered = true
	txIn.MemPool = false
	txIn.ConfirmationRequired = 0
	txIn.AllowFutureObservation = false
	txIn.TxArray = nil

	// Check for gaps in scanning by comparing with the previous expected height
	if height > 1 {
		lastScannedHeight, err := c.db.GetScanPos()
		if err == nil && lastScannedHeight > 0 && lastScannedHeight != height-1 {
			gap := height - lastScannedHeight
			if gap > 1 {
				c.logger.Warn().
					Int64("expected_height", height-1).
					Int64("last_scanned_height", lastScannedHeight).
					Int64("gap", gap).
					Msg("potential gap detected in block scanning - some blocks may be missed")
			}
		}
	}

	// Get router contract addresses - we only process transactions involving these addresses
	routerAddresses, err := c.getRouterAddresses()
	if err != nil {
		c.logger.Warn().Err(err).Msg("failed to get router addresses")
	} else if len(routerAddresses) == 0 {
		c.logger.Debug().Msg("no router contract addresses found - skipping block processing")
		return txIn, nil
	}

	// Get transactions for the specified height
	txs, err := c.getTransactionsForLedger(height)
	if err != nil {
		return txIn, fmt.Errorf("failed to get transactions for ledger %d: %w", height, err)
	}

	if len(txs) == 0 {
		c.logger.Debug().Int64("height", height).Msg("no transactions found for ledger")
		return txIn, nil
	}

	// Process ONLY transactions that involve router contracts
	txInItems, err := c.processRouterTransactionsOnly(height, txs, routerAddresses)
	if err != nil {
		return txIn, fmt.Errorf("failed to process router transactions: %w", err)
	}

	// Set the transaction array
	txIn.TxArray = txInItems

	// CRITICAL FIX: Don't update scan position here - let the main block scanner handle it
	// This prevents conflicts that cause blocks to be skipped
	// The main block scanner will update the scan position after processing all blocks sequentially

	c.logger.Debug().
		Int64("height", height).
		Int("router_transaction_count", len(txInItems)).
		Int("total_transaction_count", len(txs)).
		Msg("processed router transactions for ledger")

	return txIn, nil
}

// processRouterTransactionsOnly processes ONLY transactions that involve router contracts
// This follows the Ethereum pattern of filtering for router addresses first
func (c *StellarBlockScanner) processRouterTransactionsOnly(height int64, txs []horizon.Transaction, routerAddresses []string) ([]*types.TxInItem, error) {
	var txInItems []*types.TxInItem

	c.logger.Debug().
		Int64("height", height).
		Int("total_transactions", len(txs)).
		Strs("router_addresses", routerAddresses).
		Msg("processing router transactions only")

	// If no router addresses found, skip all processing
	if len(routerAddresses) == 0 {
		c.logger.Debug().
			Int64("height", height).
			Msg("no router addresses found - skipping all transaction processing")
		return txInItems, nil
	}

	for _, tx := range txs {
		// Skip failed transactions
		if !tx.Successful {
			c.logger.Debug().
				Str("tx_hash", tx.Hash).
				Msg("skipping failed transaction")
			continue
		}

		// Check if this transaction involves any router contract
		isRouterTransaction := false
		for _, routerAddr := range routerAddresses {
			if c.isTransactionInvolvingRouter(tx, routerAddr) {
				isRouterTransaction = true
				c.logger.Debug().
					Str("tx_hash", tx.Hash).
					Str("router_addr", routerAddr).
					Msg("found router transaction")
				break
			}
		}

		if !isRouterTransaction {
			// Skip non-router transactions - this is the key filtering step
			c.logger.Debug().
				Str("tx_hash", tx.Hash).
				Msg("skipping non-router transaction")
			continue
		}

		c.logger.Debug().
			Str("tx_hash", tx.Hash).
			Int64("height", height).
			Msg("processing router transaction")

		// Process the router transaction
		txInItem, err := c.processRouterTransaction(tx, height, routerAddresses)
		if err != nil {
			c.logger.Warn().Err(err).
				Str("tx_hash", tx.Hash).
				Int64("height", height).
				Msg("failed to process router transaction")
			continue
		}

		if txInItem != nil {
			c.logger.Info().
				Str("tx_hash", tx.Hash).
				Int64("height", height).
				Msg("processed router transaction successfully")
			txInItems = append(txInItems, txInItem)
		}
	}

	c.logger.Debug().
		Int64("height", height).
		Int("router_transactions_processed", len(txInItems)).
		Int("total_transactions", len(txs)).
		Msg("finished processing router transactions")

	return txInItems, nil
}

// isTransactionInvolvingRouter checks if a transaction involves a router contract
func (c *StellarBlockScanner) isTransactionInvolvingRouter(tx horizon.Transaction, routerAddr string) bool {
	// Get operations for this transaction
	ops, err := c.getOperationsForTransaction(tx.Hash)
	if err != nil {
		c.logger.Debug().Err(err).
			Str("tx_hash", tx.Hash).
			Msg("failed to get operations for transaction")
		return false
	}

	// Check if any operation involves the router contract
	for _, op := range ops {
		if c.isOperationInvolvingRouter(op, routerAddr) {
			return true
		}
	}

	return false
}

// isOperationInvolvingRouter checks if an operation involves a router contract
func (c *StellarBlockScanner) isOperationInvolvingRouter(op operations.Operation, routerAddr string) bool {
	// Check for invoke host function operations (contract calls)
	if invokeOp, ok := op.(operations.InvokeHostFunction); ok {
		return c.isRouterOperation(invokeOp, routerAddr)
	}

	// For other operation types, check if they involve the router address
	// This would need to be implemented based on how Stellar exposes contract addresses in operations
	return false
}

// processRouterTransaction processes a single router transaction
func (c *StellarBlockScanner) processRouterTransaction(tx horizon.Transaction, height int64, routerAddresses []string) (*types.TxInItem, error) {
	c.logger.Debug().
		Str("tx_hash", tx.Hash).
		Int64("height", height).
		Strs("router_addresses", routerAddresses).
		Msg("processing router transaction")

	// First, check if this transaction involves any router contract by checking operations
	ops, err := c.getOperationsForTransaction(tx.Hash)
	if err != nil {
		c.logger.Warn().Err(err).
			Str("tx_hash", tx.Hash).
			Msg("failed to get operations for transaction")
		return nil, nil
	}

	// Check operations for router-related patterns
	for _, op := range ops {
		if invokeOp, ok := op.(operations.InvokeHostFunction); ok {
			// Check if this is a router operation
			for _, routerAddr := range routerAddresses {
				if c.isRouterOperation(invokeOp, routerAddr) {
					c.logger.Info().
						Str("tx_hash", tx.Hash).
						Str("router_address", routerAddr).
						Str("function", invokeOp.Function).
						Int64("height", height).
						Msg("found router operation in transaction")

					// Process as router operation
					return c.processInvokeHostFunctionOperation(tx, invokeOp, height)
				}
			}
		}
	}

	// Second, check for router events from Soroban RPC
	if c.sorobanRPCClient != nil {
		routerEvents, err := c.sorobanRPCClient.GetRouterEvents(context.Background(), uint32(height), routerAddresses)
		if err != nil {
			c.logger.Warn().Err(err).
				Str("tx_hash", tx.Hash).
				Int64("height", height).
				Msg("failed to get router events from Soroban RPC")
		} else {
			// Look for events specifically for this transaction
			for _, event := range routerEvents {
				if event.TransactionHash == tx.Hash {
					c.logger.Info().
						Str("event_type", event.Type).
						Str("tx_hash", event.TransactionHash).
						Str("contract_address", event.ContractAddress).
						Str("asset", event.Asset).
						Str("amount", event.Amount).
						Str("from", event.FromAddress).
						Str("to", event.ToAddress).
						Int64("height", height).
						Msg("found router event for transaction")

					return c.processRouterEvent(event, height)
				}
			}
		}
	}

	c.logger.Debug().
		Str("tx_hash", tx.Hash).
		Int64("height", height).
		Msg("no router events found for transaction")

	return nil, nil
}

// getTransactionsForLedger gets all transactions for a specific ledger
func (c *StellarBlockScanner) getTransactionsForLedger(height int64) ([]horizon.Transaction, error) {
	var allTxs []horizon.Transaction

	// Get transactions from Horizon
	txRequest := horizonclient.TransactionRequest{
		ForLedger: uint(height),
		Limit:     200,
		Order:     horizonclient.OrderAsc,
	}

	err := c.retryHorizonCall("get_transactions", func() error {
		txPage, err := c.horizonClient.Transactions(txRequest)
		if err != nil {
			// Check if this is a "Resource Missing" error for missing ledger
			if strings.Contains(err.Error(), "Resource Missing") {
				c.logger.Debug().
					Int64("height", height).
					Msg("ledger does not exist, returning empty transaction list")
				return nil // Return empty transactions for missing ledgers
			}
			return err
		}

		allTxs = append(allTxs, txPage.Embedded.Records...)

		// Handle pagination
		for len(txPage.Embedded.Records) == 200 {
			txRequest.Cursor = txPage.Embedded.Records[len(txPage.Embedded.Records)-1].ID
			txPage, err = c.horizonClient.Transactions(txRequest)
			if err != nil {
				return err
			}
			allTxs = append(allTxs, txPage.Embedded.Records...)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for ledger %d: %w", height, err)
	}

	return allTxs, nil
}

// retryHorizonCall executes a function with exponential backoff retry logic for rate limits
func (c *StellarBlockScanner) retryHorizonCall(operation string, fn func() error) error {
	var err error
	for i := 0; i <= maxRetries; i++ {
		err = fn()
		if err == nil {
			return nil
		}

		// Check if it's a rate limit error
		if strings.Contains(err.Error(), "Rate Limit Exceeded") ||
			strings.Contains(err.Error(), "429") {
			if i < maxRetries {
				// Exponential backoff for rate limits: 2^attempt * baseDelay
				delay := time.Duration(1<<uint(i)) * retryDelay
				c.logger.Warn().
					Str("operation", operation).
					Int("attempt", i+1).
					Int("max_retries", maxRetries+1).
					Dur("delay", delay).
					Msg("rate limit hit, retrying after delay")
				time.Sleep(delay)
				continue
			}
		}

		// For other errors, don't retry
		break
	}

	return fmt.Errorf("failed %s after %d retries: %w", operation, maxRetries, err)
}

// getVaultPubKeyForXLM returns the vault public key for XLM chain
func (c *StellarBlockScanner) getVaultPubKeyForXLM() (common.PubKey, error) {
	// Get vault public keys from the bridge
	vaultPubKeys, err := c.bridge.GetAsgardPubKeys()
	if err != nil {
		return common.EmptyPubKey, fmt.Errorf("failed to get asgard vault public keys: %w", err)
	}

	// Log the total number of vaults found for debugging
	c.logger.Debug().
		Int("total_vaults", len(vaultPubKeys)).
		Msg("retrieved vault pub keys from bridge")

	// If no vaults are returned, log a warning and return empty
	if len(vaultPubKeys) == 0 {
		c.logger.Warn().
			Msg("no vaults returned from bridge, this may indicate a configuration issue")
		return common.EmptyPubKey, fmt.Errorf("no vaults returned from bridge")
	}

	// Find the vault that supports XLM chain
	for i, vault := range vaultPubKeys {
		c.logger.Debug().
			Int("vault_index", i).
			Str("vault_pubkey", vault.PubKey.String()).
			Int("contract_count", len(vault.Contracts)).
			Msg("checking vault for XLM chain support")

		// Log all contracts in this vault for debugging
		for chain, contractAddr := range vault.Contracts {
			c.logger.Debug().
				Str("vault_pubkey", vault.PubKey.String()).
				Str("chain", chain.String()).
				Str("contract_address", contractAddr.String()).
				Msg("vault contract details")
		}

		// Check if this vault has a contract for the XLM chain
		if contractAddr, exists := vault.Contracts[common.StellarChain]; exists && !contractAddr.IsEmpty() {
			// Verify we can get a valid XLM address from this vault pub key
			_, err := vault.PubKey.GetAddress(common.StellarChain)
			if err != nil {
				c.logger.Debug().
					Str("vault_pubkey", vault.PubKey.String()).
					Err(err).
					Msg("vault has XLM contract but failed to get XLM address, skipping")
				continue
			}

			// Log the found vault for debugging
			c.logger.Debug().
				Str("vault_pubkey", vault.PubKey.String()).
				Str("xlm_contract", contractAddr.String()).
				Msg("found vault supporting XLM chain")
			return vault.PubKey, nil
		}
	}

	// If we get here, no vault supports XLM chain
	c.logger.Warn().
		Int("total_vaults_checked", len(vaultPubKeys)).
		Str("expected_chain", common.StellarChain.String()).
		Msg("no vault found supporting XLM chain")

	// Return a more descriptive error
	return common.EmptyPubKey, fmt.Errorf("no vault found supporting XLM chain (checked %d vaults)", len(vaultPubKeys))
}
