package stellar

import (
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/stellar/go/clients/horizonclient"
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

// FetchTxs fetches transactions for a block height
func (s *StellarBlockScanner) FetchTxs(height, _ int64) (stypes.TxIn, error) {
	txIn := stypes.TxIn{
		Chain:   common.STELLARChain,
		TxArray: make([]stypes.TxInItem, 0),
	}

	req := horizonclient.TransactionRequest{
		ForLedger: uint(height),
		Limit:     100, // Adjust as needed
	}

	txs, err := s.client.client.Transactions(req)
	if err != nil {
		return txIn, fmt.Errorf("failed to get transactions: %w", err)
	}

	for _, tx := range txs.Embedded.Records {
		// Process operations to extract transfers
		ops, err := s.client.client.Operations(horizonclient.OperationRequest{
			ForTransaction: tx.Hash,
		})
		if err != nil {
			s.logger.Error().Err(err).Str("hash", tx.Hash).Msg("failed to get operations")
			continue
		}

		for _, op := range ops.Embedded.Records {
			payment, ok := op.(operations.Payment)
			if !ok {
				continue
			}

			// Only process XLM transfers for now
			if payment.Asset.Type != "native" {
				continue
			}

			txInItem := stypes.TxInItem{
				BlockHeight: height,
				Tx:          tx.Hash,
				Sender:      payment.From,
				To:          payment.To,
				Memo:        tx.Memo,
				Coins: common.Coins{
					common.NewCoin(common.XLMAsset, cosmos.NewUintFromString(payment.Amount)),
				},
			}
			txIn.TxArray = append(txIn.TxArray, txInItem)
		}
	}

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
