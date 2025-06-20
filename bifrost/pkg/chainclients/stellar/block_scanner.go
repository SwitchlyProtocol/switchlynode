package stellar

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	sdkmath "cosmossdk.io/math"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/config"
)

// SolvencyReporter is to report solvency info to THORNode
type SolvencyReporter func(int64) error

const (
	// FeeUpdatePeriodBlocks is the block interval at which we report fee changes.
	FeeUpdatePeriodBlocks = 20

	// FeeCacheTransactions is the number of transactions over which we compute an average
	// (mean) fee price to use for outbound transactions.
	FeeCacheTransactions = 200
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
) (*StellarBlockScanner, error) {
	if scanStorage == nil {
		return nil, errors.New("scanStorage is nil")
	}
	if m == nil {
		return nil, errors.New("metrics is nil")
	}

	logger := log.Logger.With().Str("module", "blockscanner").Str("chain", cfg.ChainID.String()).Logger()

	return &StellarBlockScanner{
		cfg:              cfg,
		logger:           logger,
		db:               scanStorage,
		horizonClient:    horizonClient,
		feeCache:         make([]sdkmath.Uint, 0),
		lastFee:          sdkmath.NewUint(100), // Default base fee in stroops
		bridge:           bridge,
		solvencyReporter: solvencyReporter,
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
		return sdkmath.NewUint(100) // Default base fee
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
	// Only process payment operations for now
	payment, ok := op.(operations.Payment)
	if !ok {
		return nil, nil // Not a payment operation, skip
	}

	// Determine asset type and find mapping
	var assetMapping StellarAssetMapping
	var found bool

	if payment.Asset.Type == "native" {
		assetMapping, found = GetAssetByStellarAsset("native", "", "")
	} else {
		// For non-native assets, extract code and issuer
		assetCode := payment.Asset.Code
		assetIssuer := payment.Asset.Issuer
		assetMapping, found = GetAssetByStellarAsset(payment.Asset.Type, assetCode, assetIssuer)
	}

	if !found {
		return nil, nil // Asset not supported/whitelisted, skip
	}

	// Convert amount using the asset mapping
	coin, err := assetMapping.ConvertToTHORChainAmount(payment.Amount)
	if err != nil {
		return nil, fmt.Errorf("fail to convert amount: %w", err)
	}

	if coin.Amount.IsZero() {
		return nil, nil // Zero amount, skip
	}

	// Create addresses
	fromAddr, err := common.NewAddress(payment.From)
	if err != nil {
		return nil, fmt.Errorf("fail to parse from address: %w", err)
	}

	toAddr, err := common.NewAddress(payment.To)
	if err != nil {
		return nil, fmt.Errorf("fail to parse to address: %w", err)
	}

	// Create coins
	coins := common.Coins{coin}

	// Extract memo
	memo := tx.Memo

	txInItem := &types.TxInItem{
		BlockHeight: height,
		Tx:          tx.Hash,
		Sender:      fromAddr.String(),
		To:          toAddr.String(),
		Coins:       coins,
		Memo:        memo,
		Gas:         common.Gas{coin},
	}

	return txInItem, nil
}

// retryHorizonCall executes a function with exponential backoff retry logic for rate limits
func (c *StellarBlockScanner) retryHorizonCall(operation string, fn func() error) error {
	maxRetries := c.cfg.MaxHTTPRequestRetry
	baseDelay := c.cfg.BlockHeightDiscoverBackoff

	for attempt := 0; attempt <= maxRetries; attempt++ {
		err := fn()
		if err != nil {
			// Check if it's a rate limit error
			if strings.Contains(err.Error(), "Rate Limit Exceeded") ||
				strings.Contains(err.Error(), "429") {
				if attempt < maxRetries {
					// Exponential backoff for rate limits: 2^attempt * baseDelay
					delay := time.Duration(1<<uint(attempt)) * baseDelay
					c.logger.Warn().
						Str("operation", operation).
						Int("attempt", attempt+1).
						Int("max_retries", maxRetries+1).
						Dur("delay", delay).
						Msg("rate limit hit, retrying after delay")
					time.Sleep(delay)
					continue
				}
			}
			return err
		}
		return nil
	}

	return fmt.Errorf("max retries exceeded for operation: %s", operation)
}

// FetchTxs retrieves transactions for a given block height
func (c *StellarBlockScanner) FetchTxs(height, chainHeight int64) (types.TxIn, error) {
	txIn := types.TxIn{
		Chain:   c.cfg.ChainID,
		TxArray: nil,
	}

	// Get ledger information (just to verify it exists) with retry logic
	err := c.retryHorizonCall("ledger_detail", func() error {
		_, err := c.horizonClient.LedgerDetail(uint32(height))
		return err
	})
	if err != nil {
		return txIn, fmt.Errorf("fail to get ledger %d: %w", height, err)
	}

	// Get transactions for this ledger with retry logic
	var txPage horizon.TransactionsPage
	err = c.retryHorizonCall("transactions", func() error {
		txRequest := horizonclient.TransactionRequest{
			ForLedger: uint(height),
			Limit:     200, // Maximum allowed by Horizon
		}

		var err error
		txPage, err = c.horizonClient.Transactions(txRequest)
		return err
	})
	if err != nil {
		return txIn, fmt.Errorf("fail to get transactions for ledger %d: %w", height, err)
	}

	// Process all transactions
	var allTxs []horizon.Transaction
	allTxs = append(allTxs, txPage.Embedded.Records...)

	// Handle pagination if there are more transactions with retry logic
	for len(txPage.Embedded.Records) == 200 {
		err = c.retryHorizonCall("next_transactions_page", func() error {
			var err error
			txPage, err = c.horizonClient.NextTransactionsPage(txPage)
			return err
		})
		if err != nil {
			c.logger.Warn().Err(err).Int64("height", height).Msg("failed to get next transactions page, continuing")
			break // No more pages or error
		}
		allTxs = append(allTxs, txPage.Embedded.Records...)
	}

	// Process transactions
	txInItems, err := c.processTxs(height, allTxs)
	if err != nil {
		return txIn, fmt.Errorf("fail to process transactions: %w", err)
	}

	// Update fees
	if err := c.updateFees(height); err != nil {
		c.logger.Error().Err(err).Int64("height", height).Msg("fail to update fees")
	}

	txIn.TxArray = txInItems

	c.logger.Info().
		Int64("height", height).
		Int("tx_count", len(txInItems)).
		Msg("processed block")

	return txIn, nil
}
