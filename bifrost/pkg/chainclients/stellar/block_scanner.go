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
		cfg:              cfg,
		logger:           logger,
		db:               scanStorage,
		horizonClient:    horizonClient,
		feeCache:         make([]sdkmath.Uint, 0),
		lastFee:          sdkmath.NewUint(baseFeeStroops),
		bridge:           bridge,
		solvencyReporter: solvencyReporter,
		sorobanRPCClient: sorobanRPCClient,
	}, nil
}

// GetHeight returns the latest ledger sequence number with retry logic for rate limits
func (c *StellarBlockScanner) GetHeight() (int64, error) {
	maxRetries := c.cfg.MaxHTTPRequestRetry
	baseDelay := c.cfg.BlockHeightDiscoverBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		ledger, err := c.horizonClient.Root()
		if err != nil {
			// Check if it's a rate limit error
			if strings.Contains(err.Error(), "Rate Limit Exceeded") ||
				strings.Contains(err.Error(), "429") {
				if attempt < maxRetries {
					// Exponential backoff for rate limits: 2^attempt * baseDelay
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
			return 0, fmt.Errorf("fail to get root info: %w", err)
		}

		return int64(ledger.HorizonSequence), nil
	}

	return 0, fmt.Errorf("max retries exceeded for getting chain height")
}

// FetchMemPool returns nothing since we are only concerned about finalized transactions in Stellar
func (c *StellarBlockScanner) FetchMemPool(height int64) (types.TxIn, error) {
	return types.TxIn{}, nil
}

// GetNetworkFee returns current chain network fee according to Bifrost.
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

		c.globalNetworkFeeQueue <- common.NetworkFee{
			Chain:           c.cfg.ChainID,
			Height:          height,
			TransactionSize: 1,
			TransactionRate: avgFee.Uint64(),
		}

		c.lastFee = avgFee
		c.logger.Info().
			Uint64("fee", avgFee.Uint64()).
			Int64("height", height).
			Msg("sent network fee to THORChain")
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
	coin, err := mapping.ConvertToTHORChainAmount(event.Amount)
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
		Str("asset", mapping.THORChainAsset.String()).
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
	coin, err := mapping.ConvertToTHORChainAmount(event.Amount)
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
		Str("asset", mapping.THORChainAsset.String()).
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
	coin, err := mapping.ConvertToTHORChainAmount(event.Amount)
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
		Str("asset", mapping.THORChainAsset.String()).
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

// parseAssetAndAmount parses a Stellar asset and amount into a THORChain coin
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

// FetchTxs fetches transactions for a given block height
func (c *StellarBlockScanner) FetchTxs(height, chainHeight int64) (types.TxIn, error) {
	c.logger.Debug().
		Int64("height", height).
		Int64("chain_height", chainHeight).
		Msg("fetching transactions")

	txIn := types.TxIn{
		Chain:                  common.StellarChain,
		TxArray:                nil,
		Filtered:               false,
		MemPool:                false,
		ConfirmationRequired:   0,
		AllowFutureObservation: false,
	}

	// Get transactions for this ledger
	txs, err := c.getTransactionsForLedger(height)
	if err != nil {
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
