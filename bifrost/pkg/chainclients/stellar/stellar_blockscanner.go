// Package stellar implements the Stellar chain client for Thorchain
package stellar

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/protocols/horizon"
	"github.com/stellar/go/protocols/horizon/operations"

	stypes "gitlab.com/thorchain/thornode/bifrost/thorclient/types"
	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
)

// StellarBlockScanner is responsible for scanning blocks and collecting transactions
type StellarBlockScanner struct {
	client         *Client
	logger         zerolog.Logger
	globalTxsQueue chan stypes.TxIn
	stopChan       chan struct{}
	wg             *sync.WaitGroup
	lastBlock      int64
	healthy        bool
}

// NewStellarBlockScanner creates a new block scanner
func NewStellarBlockScanner(client *Client) (*StellarBlockScanner, error) {
	if client == nil {
		return nil, fmt.Errorf("client is nil")
	}

	return &StellarBlockScanner{
		client:   client,
		logger:   log.With().Str("module", "stellar_scanner").Logger(),
		stopChan: make(chan struct{}),
		wg:       &sync.WaitGroup{},
		healthy:  true,
	}, nil
}

// Start starts the scanner
func (s *StellarBlockScanner) Start(globalTxsQueue chan stypes.TxIn) error {
	s.globalTxsQueue = globalTxsQueue
	s.wg.Add(1)
	go s.scan()
	return nil
}

// Stop stops the scanner
func (s *StellarBlockScanner) Stop() {
	close(s.stopChan)
	s.wg.Wait()
}

// IsHealthy returns the health status of the scanner
func (s *StellarBlockScanner) IsHealthy() bool {
	return s.healthy
}

// GetHeight returns the current block height
func (s *StellarBlockScanner) GetHeight() (int64, error) {
	height, err := s.client.GetHeight()
	if err != nil {
		return 0, fmt.Errorf("failed to get height: %w", err)
	}
	return height, nil
}

// validateUSDCTransaction performs additional validation for USDC transactions
// It checks:
// 1. Asset type and code
// 2. USDC issuer address
// 3. Amount format
// 4. From/To addresses
func (s *StellarBlockScanner) validateUSDCTransaction(payment *operations.Payment) error {
	if payment.Asset.Type != "credit_alphanum4" || payment.Asset.Code != "USDC" {
		return fmt.Errorf("invalid asset type for USDC transaction")
	}

	// Validate USDC issuer
	expectedIssuer := "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN"
	if payment.Asset.Issuer != expectedIssuer {
		return fmt.Errorf("invalid USDC issuer: expected %s, got %s", expectedIssuer, payment.Asset.Issuer)
	}

	// Validate amount format
	_, err := strconv.ParseFloat(payment.Amount, 64)
	if err != nil {
		return fmt.Errorf("invalid USDC amount format: %w", err)
	}

	// Validate addresses
	if payment.From == "" || payment.To == "" {
		return fmt.Errorf("invalid USDC transaction: missing from/to addresses")
	}

	return nil
}

// processOperations processes a slice of operations for a transaction
func (s *StellarBlockScanner) processOperations(ops []operations.Operation, txHash, memo string, ledger int32) ([]stypes.TxInItem, error) {
	var txs []stypes.TxInItem
	startTime := time.Now()

	s.logger.Debug().
		Str("tx_id", txHash).
		Str("memo", memo).
		Int("operations", len(ops)).
		Msg("processing transaction operations")

	for _, op := range ops {
		payment, ok := op.(*operations.Payment)
		if !ok {
			s.logger.Debug().
				Str("tx_id", txHash).
				Str("operation_type", fmt.Sprintf("%T", op)).
				Msg("skipping non-payment operation")
			continue
		}

		var asset common.Asset
		var err error

		// Handle USDC transactions
		if payment.Asset.Type == "credit_alphanum4" && payment.Asset.Code == "USDC" {
			if err := s.validateUSDCTransaction(payment); err != nil {
				s.logger.Error().
					Err(err).
					Str("tx_id", txHash).
					Str("asset_type", payment.Asset.Type).
					Str("asset_code", payment.Asset.Code).
					Msg("invalid USDC transaction")
				RecordUSDCTransactionError("validation")
				continue
			}
			asset = stellarUSDC
			s.logger.Debug().
				Str("tx_id", txHash).
				Str("amount", payment.Amount).
				Str("from", payment.From).
				Str("to", payment.To).
				Msg("processing USDC transaction")
		} else if payment.Asset.Type == "native" {
			asset = common.XLMAsset
		} else {
			s.logger.Warn().
				Str("tx_id", txHash).
				Str("asset_type", payment.Asset.Type).
				Str("asset_code", payment.Asset.Code).
				Msg("unsupported asset type")
			continue
		}

		// Parse amount
		amount, err := strconv.ParseFloat(payment.Amount, 64)
		if err != nil {
			s.logger.Error().
				Err(err).
				Str("tx_id", txHash).
				Str("amount", payment.Amount).
				Msg("failed to parse amount")
			RecordUSDCTransactionError("amount_parse")
			continue
		}

		// Convert to base units (multiply by 1e7 for Stellar)
		amountInt := int64(amount * 1e7)
		if amountInt <= 0 {
			s.logger.Warn().
				Str("tx_id", txHash).
				Float64("amount", amount).
				Msg("invalid amount")
			RecordUSDCTransactionError("invalid_amount")
			continue
		}

		// Create transaction
		txIn := stypes.TxInItem{
			BlockHeight: int64(ledger),
			Tx:          txHash,
			Sender:      payment.From,
			To:          payment.To,
			Memo:        memo,
			Coins: common.Coins{
				common.NewCoin(asset, cosmos.NewUint(uint64(amountInt))),
			},
		}

		txs = append(txs, txIn)

		// Record metrics for USDC transactions
		if asset.Equals(stellarUSDC) {
			duration := time.Since(startTime).Seconds()
			RecordUSDCTransaction("success", amount, "process", duration)
			s.logger.Info().
				Str("tx_id", txHash).
				Float64("amount", amount).
				Float64("duration_seconds", duration).
				Msg("successfully processed USDC transaction")
		}
	}

	return txs, nil
}

// processTransaction processes a single transaction and returns a slice of TxInItem
// It handles both XLM and USDC transactions, with specific validation for USDC
func (s *StellarBlockScanner) processTransaction(tx *horizon.Transaction) ([]stypes.TxInItem, error) {
	// In production, you would fetch the operations for this transaction from Horizon
	// For now, assume you have a function getOperationsForTransaction(tx) ([]operations.Operation, error)
	// This is a placeholder for actual implementation
	ops, err := getOperationsForTransaction(tx)
	if err != nil {
		return nil, err
	}
	return s.processOperations(ops, tx.Hash, tx.Memo, tx.Ledger)
}

// Placeholder for fetching operations for a transaction
func getOperationsForTransaction(tx *horizon.Transaction) ([]operations.Operation, error) {
	// In real code, use the Horizon client to fetch operations for the transaction
	return nil, fmt.Errorf("not implemented: fetch operations for transaction")
}

// FetchTxs fetches transactions for a block height
// It processes both XLM and USDC transactions, recording metrics for monitoring
func (s *StellarBlockScanner) FetchTxs(height, _ int64) (stypes.TxIn, error) {
	startTime := time.Now()
	txIn := stypes.TxIn{
		Chain:   common.STELLARChain,
		TxArray: make([]stypes.TxInItem, 0),
	}

	s.logger.Debug().
		Int64("height", height).
		Msg("fetching transactions for block")

	req := horizonclient.TransactionRequest{
		ForLedger: uint(height),
		Limit:     100, // Adjust as needed
	}

	txs, err := s.client.client.Transactions(req)
	if err != nil {
		s.logger.Error().
			Err(err).
			Int64("height", height).
			Msg("failed to get transactions")
		RecordUSDCTransactionError("fetch_transactions")
		return txIn, fmt.Errorf("failed to get transactions: %w", err)
	}

	s.logger.Debug().
		Int64("height", height).
		Int("tx_count", len(txs.Embedded.Records)).
		Msg("processing block transactions")

	for _, tx := range txs.Embedded.Records {
		txInItems, err := s.processTransaction(&tx)
		if err != nil {
			s.logger.Error().
				Err(err).
				Int64("height", height).
				Str("tx_id", tx.Hash).
				Msg("failed to process transaction")
			RecordUSDCTransactionError("process_transaction")
			continue
		}

		txIn.TxArray = append(txIn.TxArray, txInItems...)
	}

	// Record overall block processing metrics
	duration := time.Since(startTime).Seconds()
	RecordUSDCTransaction("block_complete", float64(len(txIn.TxArray)), "block_process", duration)

	s.logger.Info().
		Int64("height", height).
		Int("tx_count", len(txIn.TxArray)).
		Float64("duration_seconds", duration).
		Msg("completed block processing")

	return txIn, nil
}

func (s *StellarBlockScanner) scan() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stopChan:
			return
		default:
			currentHeight, err := s.GetHeight()
			if err != nil {
				s.healthy = false
				s.logger.Error().Err(err).Msg("failed to get current height")
				time.Sleep(time.Second * 5)
				continue
			}

			if currentHeight <= s.lastBlock {
				time.Sleep(time.Second * 5)
				continue
			}

			for height := s.lastBlock + 1; height <= currentHeight; height++ {
				txIn, err := s.FetchTxs(height, currentHeight)
				if err != nil {
					s.logger.Error().Err(err).Int64("height", height).Msg("failed to fetch transactions")
					continue
				}

				if len(txIn.TxArray) > 0 {
					s.logger.Info().Int64("height", height).Int("count", len(txIn.TxArray)).Msg("found transactions")
					s.globalTxsQueue <- txIn
				}
			}

			s.lastBlock = currentHeight
			s.healthy = true
		}
	}
}
