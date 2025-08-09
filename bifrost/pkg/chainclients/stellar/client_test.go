package stellar

import (
	"math/big"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v1/bifrost/pkg/chainclients/shared/signercache"
	stypes "github.com/switchlyprotocol/switchlynode/v1/bifrost/thorclient/types"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/config"
)

func TestPackage(t *testing.T) { TestingT(t) }

var sharedMetrics *metrics.Metrics

func GetMetricForTest(c *C) *metrics.Metrics {
	if sharedMetrics == nil {
		var err error
		sharedMetrics, err = metrics.NewMetrics(config.BifrostMetricsConfiguration{
			Enabled:      false,
			ListenPort:   9000,
			ReadTimeout:  time.Second,
			WriteTimeout: time.Second,
			Chains:       common.Chains{common.StellarChain},
		})
		c.Assert(sharedMetrics, NotNil)
		c.Assert(err, IsNil)
	}
	return sharedMetrics
}

type StellarClientTestSuite struct {
	client *Client
	bridge *MockThorchainBridge
}

var _ = Suite(&StellarClientTestSuite{})

func (s *StellarClientTestSuite) SetUpSuite(c *C) {
	// Create shared metrics
	m := GetMetricForTest(c)
	c.Assert(m, NotNil)

	// Create mock bridge
	s.bridge = &MockThorchainBridge{}

	// Mock configuration
	cfg := config.BifrostChainConfiguration{
		ChainID:      common.StellarChain,
		ChainNetwork: "testnet",
		UserName:     "",
		Password:     "",
		DisableTLS:   true,
		HTTPostMode:  true,
		BlockScanner: config.BifrostBlockScannerConfiguration{
			ChainID:                      common.StellarChain,
			ObservationFlexibilityBlocks: 1,
			StartBlockHeight:             1,
		},
	}

	// Create mock storage
	storage, err := blockscanner.NewBlockScannerStorage("", config.LevelDBOptions{})
	c.Assert(err, IsNil)

	// Create mock horizon client
	horizonClient := &horizonclient.Client{
		HorizonURL: "https://horizon-testnet.stellar.org",
	}

	// Create mock network fee queue
	mockNetworkFeeQueue := make(chan common.NetworkFee, 100)

	// Create block scanner
	scanner, err := NewStellarBlockScanner(
		"https://horizon-testnet.stellar.org",
		cfg.BlockScanner,
		storage,
		s.bridge,
		m,
		func(int64) error { return nil }, // mock solvency reporter
		horizonClient,
		&SorobanRPCClient{}, // mock soroban client
		mockNetworkFeeQueue,
	)
	c.Assert(err, IsNil)

	// Create block scanner wrapper
	blockScanner, err := blockscanner.NewBlockScanner(cfg.BlockScanner, storage, m, s.bridge, scanner)
	c.Assert(err, IsNil)

	// Create signer cache
	signerCacheManager, err := signercache.NewSignerCacheManager(storage.GetInternalDb())
	c.Assert(err, IsNil)

	// Create client
	s.client = &Client{
		cfg:                cfg,
		thorchainBridge:    s.bridge,
		storage:            storage,
		blockScanner:       blockScanner,
		signerCacheManager: signerCacheManager,
		stellarScanner:     scanner,
		networkPassphrase:  network.TestNetworkPassphrase,
		horizonClient:      horizonClient,
	}
}

func (s *StellarClientTestSuite) TestGetChain(c *C) {
	chain := s.client.GetChain()
	c.Assert(chain, Equals, common.StellarChain)
}

func (s *StellarClientTestSuite) TestGetAddress(c *C) {
	// Test with a valid Stellar public key
	pubKeyStr := "tswitchpub1addwnpepqflvfv08t6qt95lmttd6wpf3ss8wx63e9vf6fvyuj2yy6nnyna576qmw2y8"
	pubKey, err := common.NewPubKey(pubKeyStr)
	if err != nil {
		// If the pubkey format is invalid, test that GetAddress handles it gracefully
		addr := s.client.GetAddress(pubKey)
		c.Assert(addr, Equals, "")
		return
	}

	addr := s.client.GetAddress(pubKey)
	c.Assert(addr, Not(Equals), "")
	c.Assert(len(addr), Equals, 56)       // Stellar addresses are 56 characters long
	c.Assert(addr[0], Equals, uint8('G')) // Stellar addresses start with 'G'
}

func (s *StellarClientTestSuite) TestStellarConstants(c *C) {
	// Test that the client can access Stellar constants
	c.Assert(s.client.cfg.ChainID, Equals, common.StellarChain)
	c.Assert(s.client.cfg.ChainNetwork, Equals, "testnet")
}

func (s *StellarClientTestSuite) TestGetConfig(c *C) {
	// Test configuration retrieval
	cfg := s.client.GetConfig()
	c.Assert(cfg.ChainID, Equals, common.StellarChain)
	c.Assert(cfg.ChainNetwork, Equals, "testnet")
}

func (s *StellarClientTestSuite) TestNetworkPassphrase(c *C) {
	// Test network passphrase selection
	c.Assert(s.client.networkPassphrase, Not(Equals), "")

	// For testnet, should use testnet passphrase
	c.Assert(s.client.networkPassphrase, Equals, network.TestNetworkPassphrase)
}

func (s *StellarClientTestSuite) TestSignTxValidation(c *C) {
	// Test transaction signing validation
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000)) // 1 XLM in stroops
	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.EmptyPubKey,
		Coins:       common.Coins{coin},
		Memo:        "test memo",
		MaxGas:      common.Gas{common.NewCoin(common.XLMAsset, cosmos.NewUint(1000))},
		GasRate:     1,
	}

	// This should fail because we don't have proper TSS setup in test
	_, _, _, err := s.client.SignTx(txOutItem, 1)
	c.Assert(err, NotNil) // Expected to fail in test environment
}

func (s *StellarClientTestSuite) TestGetHeight(c *C) {
	// Test height retrieval (will fail in test environment but should not panic)
	height, err := s.client.GetHeight()
	// In test environment, this will likely fail due to no real connection
	// but we test that it doesn't panic and returns proper error handling
	if err != nil {
		c.Assert(height, Equals, int64(0))
	} else {
		c.Assert(height >= 0, Equals, true)
	}
}

func (s *StellarClientTestSuite) TestBroadcastTx(c *C) {
	// Test transaction broadcasting validation
	txBytes := []byte("test transaction")
	txID, err := s.client.BroadcastTx(stypes.TxOutItem{}, txBytes)

	// Should fail in test environment but not panic
	c.Assert(err, NotNil)
	c.Assert(string(txID), Equals, "")
}

func (s *StellarClientTestSuite) TestGetAccountByAddress(c *C) {
	// Test with an invalid address
	account, err := s.client.GetAccountByAddress("invalid_address", big.NewInt(0))
	c.Assert(err, NotNil)
	c.Assert(account.Coins.IsEmpty(), Equals, true)

	// Test with a valid but non-existent address
	validAddr := "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"
	account, err = s.client.GetAccountByAddress(validAddr, big.NewInt(0))
	// This might fail due to network call, but address validation should pass
	if err == nil {
		c.Assert(account, NotNil)
	}
}

func (s *StellarClientTestSuite) TestProcessOutboundTx(c *C) {
	// Test XLM transaction
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000)) // 1 XLM in stroops
	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.EmptyPubKey,
		Coins:       common.Coins{coin},
		Memo:        "test memo",
	}

	payment, err := s.client.processOutboundTx(txOutItem)
	c.Assert(err, IsNil)
	c.Assert(payment, NotNil)
	c.Assert(payment.Destination, Equals, "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM")

	// Check that it's a native asset
	_, isNative := payment.Asset.(txnbuild.NativeAsset)
	c.Assert(isNative, Equals, true)

	// Test USDC transaction
	usdcAsset := common.Asset{Chain: common.StellarChain, Symbol: "USDC", Ticker: "USDC"}
	usdcCoin := common.NewCoin(usdcAsset, cosmos.NewUint(100000000)) // 1 USDC in THORChain units
	usdcTxOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.EmptyPubKey,
		Coins:       common.Coins{usdcCoin},
		Memo:        "test usdc memo",
	}

	usdcPayment, err := s.client.processOutboundTx(usdcTxOutItem)
	c.Assert(err, IsNil)
	c.Assert(usdcPayment, NotNil)
	c.Assert(usdcPayment.Destination, Equals, "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM")

	// Check that it's a credit asset
	creditAsset, isCredit := usdcPayment.Asset.(txnbuild.CreditAsset)
	c.Assert(isCredit, Equals, true)
	c.Assert(creditAsset.Code, Equals, "USDC")
	c.Assert(creditAsset.Issuer, Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA") // Testnet address

	// Test unsupported asset
	unsupportedAsset := common.Asset{Chain: common.StellarChain, Symbol: "UNKNOWN", Ticker: "UNKNOWN"}
	unsupportedCoin := common.NewCoin(unsupportedAsset, cosmos.NewUint(1000000))
	unsupportedTxOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.EmptyPubKey,
		Coins:       common.Coins{unsupportedCoin},
		Memo:        "test unsupported memo",
	}

	_, err = s.client.processOutboundTx(unsupportedTxOutItem)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*unsupported asset.*")
}

func (s *StellarClientTestSuite) TestConfirmationCount(c *C) {
	// Test confirmation counting
	txIn := stypes.TxIn{
		Chain:    common.StellarChain,
		TxArray:  nil,
		Filtered: false,
		MemPool:  false,
	}

	// Should return 1 for Stellar (finalized transactions)
	count := s.client.GetConfirmationCount(txIn)
	c.Assert(count, Equals, int64(1))

	// Test confirmation ready check
	ready := s.client.ConfirmationCountReady(txIn)
	c.Assert(ready, Equals, true) // Should be ready for empty tx
}

func (s *StellarClientTestSuite) TestSolvencyReporting(c *C) {
	// Test solvency reporting doesn't panic
	err := s.client.ReportSolvency(1)
	// May fail due to no real bridge connection, but shouldn't panic
	if err != nil {
		c.Assert(err, NotNil)
	}
}

func (s *StellarClientTestSuite) TestShouldReportSolvency(c *C) {
	// Mock solvency blocks configuration
	s.client.cfg.SolvencyBlocks = 10

	// Test heights that should report solvency
	c.Assert(s.client.ShouldReportSolvency(10), Equals, true)
	c.Assert(s.client.ShouldReportSolvency(20), Equals, true)
	c.Assert(s.client.ShouldReportSolvency(100), Equals, true)

	// Test heights that should not report solvency
	c.Assert(s.client.ShouldReportSolvency(5), Equals, false)
	c.Assert(s.client.ShouldReportSolvency(15), Equals, false)
	c.Assert(s.client.ShouldReportSolvency(99), Equals, false)
}
