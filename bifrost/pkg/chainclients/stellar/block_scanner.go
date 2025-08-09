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

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/config"

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

// StellarBlockScanner is to scan the blocks
type StellarBlockScanner struct {
	cfg              config.BifrostBlockScannerConfiguration
	logger           zerolog.Logger
	db               blockscanner.ScannerStorage
	bridge           thorclient.ThorchainBridge
	solvencyReporter SolvencyReporter
	horizonClient    *horizonclient.Client
	sorobanRPCClient *SorobanRPCClient

	globalNetworkFeeQueue chan common.NetworkFee

	// feeCache contains a rolling window of suggested fees.
	feeCache []sdkmath.Uint
	lastFee  sdkmath.Uint
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
		return nil, errors.New("horizon client is nil")
	}
	// sorobanRPCClient is optional for backward compatibility
	// if sorobanRPCClient == nil {
	// 	return nil, errors.New("soroban RPC client is nil")
	// }

	logger := log.Logger.With().Str("module", "blockscanner").Str("chain", cfg.ChainID.String()).Logger()

	return &StellarBlockScanner{
		cfg:                   cfg,
		logger:                logger,
		db:                    scanStorage,
		horizonClient:         horizonClient,
		feeCache:              make([]sdkmath.Uint, 0),
		lastFee:               sdkmath.NewUint(baseFeeStroops),
		bridge:                bridge,
		solvencyReporter:      solvencyReporter,
		sorobanRPCClient:      sorobanRPCClient,
		globalNetworkFeeQueue: globalNetworkFeeQueue,
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

	// Create TxInItem
	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          tx.Hash,
		Sender:      payment.From,
		To:          payment.To,
		Coins:       common.Coins{coin},
		Memo:        tx.Memo,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(uint64(tx.FeeCharged))},
		},
	}

	return txInItem, nil
}

// processInvokeHostFunctionOperation processes smart contract calls (including router events)
func (c *StellarBlockScanner) processInvokeHostFunctionOperation(tx horizon.Transaction, operation operations.InvokeHostFunction, height int64) (*types.TxInItem, error) {
	// Check if this is a router contract call
	routerAddresses, err := c.getRouterAddresses()
	if err != nil {
		c.logger.Error().Err(err).Msg("failed to get router addresses")
		return nil, nil
	}

	// Check if this operation involves any router contract
	isRouterCall := false
	for _, routerAddr := range routerAddresses {
		if c.isRouterOperation(operation, routerAddr) {
			isRouterCall = true
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
			return c.processRouterEvent(event, height)
		}
	}

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

	// Create TxInItem for account creation
	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          tx.Hash,
		Sender:      operation.Funder,
		To:          operation.Account,
		Coins: common.Coins{
			common.NewCoin(common.XLMAsset, cosmos.NewUint(uint64(startingBalance*10000000))), // Convert XLM to stroops
		},
		Memo: tx.Memo,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(uint64(tx.FeeCharged))},
		},
	}

	return txInItem, nil
}

// processRouterEvent processes a router event and converts it to TxInItem
func (c *StellarBlockScanner) processRouterEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	eventType := strings.ToLower(event.Type)

	switch eventType {
	case "deposit", "router_deposit":
		return c.processRouterDepositEvent(event, height)
	case "transfer_out", "router_transfer_out", "transferout":
		return c.processRouterTransferOutEvent(event, height)
	case "deposit_with_expiry", "depositwithexpiry":
		return c.processRouterDepositEvent(event, height) // Handle as regular deposit
	case "transfer_allowance", "transferallowance":
		return c.processRouterTransferAllowanceEvent(event, height)
	case "return_vault_assets", "returnvaultassets":
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
func (c *StellarBlockScanner) processRouterDepositEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
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

	// Create TxInItem
	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          event.TransactionHash,
		Sender:      event.FromAddress,
		To:          event.ToAddress,
		Coins:       common.Coins{coin},
		Memo:        event.Memo,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(baseFeeStroops)},
		},
	}

	c.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("from", event.FromAddress).
		Str("to", event.ToAddress).
		Str("asset", mapping.SwitchlyProtocolAsset.String()).
		Str("amount", coin.Amount.String()).
		Str("memo", event.Memo).
		Msg("processed router deposit event")

	return txInItem, nil
}

// processRouterTransferOutEvent processes router transfer out events
func (c *StellarBlockScanner) processRouterTransferOutEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Transfer out events are outbound transactions, not inbound
	// We don't generate TxInItems for these, but we log them for monitoring
	c.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("to", event.ToAddress).
		Str("asset", event.Asset).
		Str("amount", event.Amount).
		Msg("router transfer out event detected")

	return nil, nil
}

// processRouterTransferAllowanceEvent processes router transfer allowance events (vault rotation)
func (c *StellarBlockScanner) processRouterTransferAllowanceEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
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

	// Create TxInItem for vault rotation
	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          event.TransactionHash,
		Sender:      event.FromAddress, // Old vault
		To:          event.ToAddress,   // New vault
		Coins:       common.Coins{coin},
		Memo:        event.Memo,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(baseFeeStroops)},
		},
	}

	c.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("old_vault", event.FromAddress).
		Str("new_vault", event.ToAddress).
		Str("asset", mapping.SwitchlyProtocolAsset.String()).
		Str("amount", coin.Amount.String()).
		Msg("processed router transfer allowance event")

	return txInItem, nil
}

// processRouterReturnVaultAssetsEvent processes router return vault assets events
func (c *StellarBlockScanner) processRouterReturnVaultAssetsEvent(event *RouterEvent, height int64) (*types.TxInItem, error) {
	// Find the asset mapping
	mapping, found := GetAssetByAddress(event.Asset)
	if !found {
		c.logger.Warn().
			Str("asset_address", event.Asset).
			Msg("unsupported asset in router return vault assets event")
		return nil, nil
	}

	// Convert amount
	coin, err := mapping.ConvertToSwitchlyProtocolAmount(event.Amount)
	if err != nil {
		return nil, fmt.Errorf("failed to convert amount: %w", err)
	}

	// Create TxInItem for vault asset return
	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          event.TransactionHash,
		Sender:      event.FromAddress, // Old vault
		To:          event.ToAddress,   // New vault
		Coins:       common.Coins{coin},
		Memo:        event.Memo,
		Gas: common.Gas{
			{Asset: common.XLMAsset, Amount: cosmos.NewUint(baseFeeStroops)},
		},
	}

	c.logger.Debug().
		Str("tx_hash", event.TransactionHash).
		Str("from_vault", event.FromAddress).
		Str("to_vault", event.ToAddress).
		Str("asset", mapping.SwitchlyProtocolAsset.String()).
		Str("amount", coin.Amount.String()).
		Msg("processed router return vault assets event")

	return txInItem, nil
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

// getRouterAddresses gets all router contract addresses
func (c *StellarBlockScanner) getRouterAddresses() ([]string, error) {
	contracts, err := c.bridge.GetContractAddress()
	if err != nil {
		return nil, fmt.Errorf("failed to get contract addresses: %w", err)
	}

	var routerAddresses []string
	for _, contract := range contracts {
		if addr, ok := contract.Contracts[common.StellarChain]; ok && !addr.IsEmpty() {
			routerAddresses = append(routerAddresses, addr.String())
		}
	}

	return routerAddresses, nil
}

// isRouterOperation checks if an operation involves a router contract
func (c *StellarBlockScanner) isRouterOperation(op operations.InvokeHostFunction, routerAddr string) bool {
	// Parse the operation to check if it involves the router contract
	// This is a simplified check - in practice, you'd parse the XDR to get the contract address
	return strings.Contains(op.Function, routerAddr)
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
// It includes gap detection to ensure no blocks are missed during scanning.
func (c *StellarBlockScanner) FetchTxs(height, chainHeight int64) (types.TxIn, error) {
	var txIn types.TxIn
	txIn.Chain = c.cfg.ChainID
	txIn.Filtered = true
	txIn.MemPool = false
	txIn.ConfirmationRequired = 0
	txIn.AllowFutureObservation = false
	txIn.TxArray = nil

	// Check for gaps in scanning and log warnings for significant gaps
	if chainHeight > 0 && height > 0 {
		expectedHeight := height - 1 // Previous expected height
		if expectedHeight > 0 && chainHeight > expectedHeight {
			gap := chainHeight - expectedHeight
			if gap > 10 { // Log warning for gaps larger than 10 blocks
				c.logger.Warn().
					Int64("expected_height", expectedHeight).
					Int64("actual_height", chainHeight).
					Int64("gap_size", gap).
					Msg("STELLAR: Large gap detected in block scanning - some blocks may have been missed")
			}
		}
	}

	// Get transactions for this ledger
	txs, err := c.getTransactionsForLedger(height)
	if err != nil {
		// Handle special case where ledger doesn't exist
		if strings.Contains(err.Error(), "Resource Missing") {
			c.logger.Debug().
				Int64("height", height).
				Msg("ledger does not exist, returning empty transaction list")
			return txIn, nil // Return empty transactions, let the scanner continue
		}
		return txIn, fmt.Errorf("failed to get transactions for ledger %d: %w", height, err)
	}

	// Process each transaction
	txInItems, err := c.processTxs(height, txs)
	if err != nil {
		return txIn, fmt.Errorf("failed to process transactions: %w", err)
	}

	txIn.TxArray = txInItems

	// Update fees if we're close to the chain tip
	if chainHeight-height <= c.cfg.ObservationFlexibilityBlocks {
		if err := c.updateFees(height); err != nil {
			c.logger.Error().Err(err).Msg("failed to update fees")
		}
	}

	c.logger.Debug().
		Int64("height", height).
		Int("tx_count", len(txInItems)).
		Msg("successfully fetched transactions")

	return txIn, nil
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
