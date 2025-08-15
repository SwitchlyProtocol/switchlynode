package signer

import (
	"errors"
	"fmt"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/blockscanner"
	btypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/blockscanner/types"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pubkeymanager"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	ttypes "github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

type SwitchlyBlockScan struct {
	logger         zerolog.Logger
	wg             *sync.WaitGroup
	stopChan       chan struct{}
	txOutChan      chan types.TxOut
	keygenChan     chan ttypes.KeygenBlock
	cfg            config.BifrostBlockScannerConfiguration
	scannerStorage blockscanner.ScannerStorage
	switchly      switchlyclient.SwitchlyBridge
	errCounter     *prometheus.CounterVec
	pubkeyMgr      pubkeymanager.PubKeyValidator
}

// NewSwitchlyBlockScan create a new instance of switchly block scanner
func NewSwitchlyBlockScan(cfg config.BifrostBlockScannerConfiguration, scanStorage blockscanner.ScannerStorage, switchly switchlyclient.SwitchlyBridge, m *metrics.Metrics, pubkeyMgr pubkeymanager.PubKeyValidator) (*SwitchlyBlockScan, error) {
	if scanStorage == nil {
		return nil, errors.New("scanStorage is nil")
	}
	if m == nil {
		return nil, errors.New("metric is nil")
	}
	return &SwitchlyBlockScan{
		logger:         log.With().Str("module", "blockscanner").Str("chain", "SWITCHLY").Logger(),
		wg:             &sync.WaitGroup{},
		stopChan:       make(chan struct{}),
		txOutChan:      make(chan types.TxOut),
		keygenChan:     make(chan ttypes.KeygenBlock),
		cfg:            cfg,
		scannerStorage: scanStorage,
		switchly:      switchly,
		errCounter:     m.GetCounterVec(metrics.SwitchlyBlockScannerError),
		pubkeyMgr:      pubkeyMgr,
	}, nil
}

// GetMessages return the channel
func (b *SwitchlyBlockScan) GetTxOutMessages() <-chan types.TxOut {
	return b.txOutChan
}

func (b *SwitchlyBlockScan) GetKeygenMessages() <-chan ttypes.KeygenBlock {
	return b.keygenChan
}

func (b *SwitchlyBlockScan) GetHeight() (int64, error) {
	return b.switchly.GetBlockHeight()
}

// SwitchlyBlockScan's GetNetworkFee only exists to satisfy the BlockScannerFetcher interface
// and should never be called, since broadcast network fees are for external chains' observed fees.
func (b *SwitchlyBlockScan) GetNetworkFee() (transactionSize, transactionFeeRate uint64) {
	b.logger.Error().Msg("SwitchlyBlockScan GetNetworkFee was called (which should never happen)")
	return 0, 0
}

func (c *SwitchlyBlockScan) FetchMemPool(height int64) (types.TxIn, error) {
	return types.TxIn{}, nil
}

func (b *SwitchlyBlockScan) FetchTxs(height, _ int64) (types.TxIn, error) {
	if err := b.processTxOutBlock(height); err != nil {
		return types.TxIn{}, err
	}
	if err := b.processKeygenBlock(height); err != nil {
		return types.TxIn{}, err
	}
	return types.TxIn{}, nil
}

func (b *SwitchlyBlockScan) processKeygenBlock(blockHeight int64) error {
	pk := b.pubkeyMgr.GetNodePubKey()
	keygen, err := b.switchly.GetKeygenBlock(blockHeight, pk.String())
	if err != nil {
		return fmt.Errorf("fail to get keygen from switchly: %w", err)
	}

	// custom error (to be dropped and not logged) because the block is
	// available yet
	if keygen.Height == 0 {
		return btypes.ErrUnavailableBlock
	}

	if len(keygen.Keygens) > 0 {
		b.keygenChan <- keygen
	}
	return nil
}

func (b *SwitchlyBlockScan) processTxOutBlock(blockHeight int64) error {
	for _, pk := range b.pubkeyMgr.GetSignPubKeys() {
		if len(pk.String()) == 0 {
			continue
		}
		tx, err := b.switchly.GetKeysign(blockHeight, pk.String())
		if err != nil {
			if errors.Is(err, btypes.ErrUnavailableBlock) {
				// custom error (to be dropped and not logged) because the block is
				// available yet
				return btypes.ErrUnavailableBlock
			}
			return fmt.Errorf("fail to get keysign from block scanner: %w", err)
		}

		if len(tx.TxArray) == 0 {
			b.logger.Debug().Int64("block", blockHeight).Msg("nothing to process")
			continue
		}
		b.txOutChan <- tx
	}
	return nil
}
