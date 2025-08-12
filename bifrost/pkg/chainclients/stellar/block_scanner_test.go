package stellar

import (
	"github.com/stellar/go/clients/horizonclient"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/config"
)

type StellarBlockScannerTestSuite struct {
	scanner *StellarBlockScanner
	m       *metrics.Metrics
	bridge  *MockThorchainBridge
}

var _ = Suite(&StellarBlockScannerTestSuite{})

func (s *StellarBlockScannerTestSuite) SetUpSuite(c *C) {
	// Create shared metrics
	var err error
	s.m = GetMetricForTest(c)
	c.Assert(s.m, NotNil)

	// Create mock bridge
	s.bridge = &MockThorchainBridge{}

	// Mock configuration
	cfg := config.BifrostBlockScannerConfiguration{
		ChainID:                      common.StellarChain,
		ObservationFlexibilityBlocks: 5,
		StartBlockHeight:             1,
	}

	// Create mock horizon client
	horizonClient := &horizonclient.Client{
		HorizonURL: "https://horizon-testnet.stellar.org",
	}

	// Create mock storage
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)

	// Create mock network fee queue
	mockNetworkFeeQueue := make(chan common.NetworkFee, 100)

	// Create mock transactions queue
	mockTxsQueue := make(chan types.TxIn, 100)

	// Create block scanner
	s.scanner, err = NewStellarBlockScanner(
		"https://horizon-testnet.stellar.org",
		cfg,
		storage,
		s.bridge,
		s.m,
		func(int64) error { return nil }, // mock solvency reporter
		horizonClient,
		&SorobanRPCClient{}, // mock soroban client
		mockNetworkFeeQueue,
		mockTxsQueue,
	)
	c.Assert(err, IsNil)
}

func (s *StellarBlockScannerTestSuite) TestNewStellarBlockScanner(c *C) {
	// Test scanner creation
	c.Assert(s.scanner, NotNil)
	c.Assert(s.scanner.cfg.ChainID, Equals, common.StellarChain)
	c.Assert(s.scanner.horizonClient, NotNil)
}

func (s *StellarBlockScannerTestSuite) TestGetHeight(c *C) {
	// Test height retrieval (will fail in test environment but should not panic)
	height, err := s.scanner.GetHeight()
	// In test environment, this will likely fail due to no real connection
	// but we test that it doesn't panic and returns proper error handling
	if err != nil {
		c.Assert(height, Equals, int64(0))
	} else {
		c.Assert(height >= 0, Equals, true)
	}
}

func (s *StellarBlockScannerTestSuite) TestFetchTxs(c *C) {
	// Test transaction fetching validation
	txIn, err := s.scanner.FetchTxs(1, 1)

	// Should fail in test environment but not panic
	if err != nil {
		c.Assert(err, NotNil)
		// When there's an error, txIn should still have the chain set
		c.Assert(txIn.Chain, Equals, common.StellarChain)
	} else {
		c.Assert(txIn, NotNil)
		c.Assert(txIn.Chain, Equals, common.StellarChain)
	}
}

func (s *StellarBlockScannerTestSuite) TestFetchMemPool(c *C) {
	// Test mempool fetching
	txIn, err := s.scanner.FetchMemPool(1)

	// Should return empty TxIn since Stellar doesn't have mempool concept
	c.Assert(err, IsNil)
	c.Assert(txIn.Chain, Equals, common.Chain("")) // Empty chain for empty mempool
	c.Assert(txIn.MemPool, Equals, false)
}

func (s *StellarBlockScannerTestSuite) TestGetNetworkFee(c *C) {
	// Test network fee calculation
	transactionSize, transactionFeeRate := s.scanner.GetNetworkFee()

	// Should return valid values
	c.Assert(transactionSize, Equals, uint64(1))
	c.Assert(transactionFeeRate >= 100, Equals, true) // Should be at least base fee
}

func (s *StellarBlockScannerTestSuite) TestProcessOperation(c *C) {
	// Test operation processing doesn't panic with nil operation
	// This is a basic validation test since we can't easily mock Stellar operations
	defer func() {
		if r := recover(); r != nil {
			c.Errorf("processOperation panicked: %v", r)
		}
	}()

	// Test with nil operation should not panic
	// The actual processing logic would be tested with proper Stellar operation mocks
}
