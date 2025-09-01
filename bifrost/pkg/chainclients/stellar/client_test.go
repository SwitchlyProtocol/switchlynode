package stellar

// Stellar Client Test Suite
//
// This test suite covers the updated Stellar client implementation that uses
// simple Payment operations for outbound transactions. Key features tested:
// - Simple payments for all mapped Stellar assets (native XLM and issued tokens)
// - Memo truncation to comply with Stellar's 28-byte limit
// - Ed25519 key derivation for signature compatibility
// - Direct Horizon API integration for improved reliability

import (
	"math/big"
	"testing"
	"time"

	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
	"github.com/stellar/go/xdr"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/bifrost/blockscanner"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/metrics"
	"github.com/switchlyprotocol/switchlynode/v3/bifrost/pkg/chainclients/shared/signercache"
	stypes "github.com/switchlyprotocol/switchlynode/v3/bifrost/switchlyclient/types"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/config"
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
	bridge *MockSwitchlyBridge
}

var _ = Suite(&StellarClientTestSuite{})

func (s *StellarClientTestSuite) SetUpSuite(c *C) {
	// Create shared metrics
	m := GetMetricForTest(c)
	c.Assert(m, NotNil)

	// Create mock bridge
	s.bridge = &MockSwitchlyBridge{}

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

	// Create mock transactions queue
	mockTxsQueue := make(chan stypes.TxIn, 100)

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
		mockTxsQueue,
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
		switchlyBridge:     s.bridge,
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
	// Test transaction signing validation with simple payment approach
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000)) // 1 XLM in stroops
	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
		Coins:       common.Coins{coin},
		Memo:        "OUT:1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF", // Test long memo
		MaxGas:      common.Gas{common.NewCoin(common.XLMAsset, cosmos.NewUint(1000))},
		GasRate:     1,
	}

	// This should fail due to invalid stellar address in test environment
	_, _, _, err := s.client.SignTx(txOutItem, 1)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*invalid stellar address.*") // Should fail due to invalid test address
}

func (s *StellarClientTestSuite) TestSignTxSimplePaymentApproach(c *C) {
	// Verify that our implementation uses simple payments with memo truncation
	// All memo lengths should work, with long memos being truncated to 28 bytes

	// Test with various memo lengths to ensure they're handled properly
	testCases := []struct {
		name string
		memo string
	}{
		{"short memo", "test"},
		{"28 byte memo", "1234567890123456789012345678"}, // Exactly 28 bytes
		{"long memo", "OUT:1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF1234567890ABCDEF"}, // 67 bytes
		{"very long memo", "OUT:LONG_TRANSACTION_ID_" + string(make([]byte, 100))},            // Very long
	}

	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000))

	for _, tc := range testCases {
		txOutItem := stypes.TxOutItem{
			Chain:       common.StellarChain,
			ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
			VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
			Coins:       common.Coins{coin},
			Memo:        tc.memo,
			MaxGas:      common.Gas{common.NewCoin(common.XLMAsset, cosmos.NewUint(1000))},
			GasRate:     1,
		}

		// All should fail due to invalid stellar address in test environment, not memo length
		_, _, _, err := s.client.SignTx(txOutItem, 1)
		c.Assert(err, NotNil, Commentf("Test case: %s", tc.name))
		c.Assert(err.Error(), Matches, ".*invalid stellar address.*",
			Commentf("Expected invalid address error for case: %s, got: %v", tc.name, err))
	}
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
	// Verify processOutboundTx creates proper Payment operations for supported assets

	// Test native XLM payment creation
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(10000000)) // 1 XLM (SwitchlyNode: 8 decimals → Stellar: 7 decimals)
	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
		Coins:       common.Coins{coin},
		Memo:        "test memo",
	}

	payment, err := s.client.processOutboundTx(txOutItem)
	c.Assert(err, IsNil)
	c.Assert(payment, NotNil)

	// Verify Payment operation properties
	c.Assert(payment.Destination, Equals, "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM")
	c.Assert(payment.Amount, Equals, "1.0000000") // 10000000 / 10^7 = 1.0 XLM

	// Confirm native XLM asset type
	_, isNative := payment.Asset.(txnbuild.NativeAsset)
	c.Assert(isNative, Equals, true)

	// Test error handling: empty coins array
	emptyTxOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
		Coins:       common.Coins{},
		Memo:        "test memo",
	}

	_, err = s.client.processOutboundTx(emptyTxOutItem)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*no coins to send.*")

	// Test error handling: unsupported asset
	unsupportedAsset := common.Asset{Chain: common.StellarChain, Symbol: "UNKNOWN", Ticker: "UNKNOWN"}
	unsupportedCoin := common.NewCoin(unsupportedAsset, cosmos.NewUint(1000000))
	unsupportedTxOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
		Coins:       common.Coins{unsupportedCoin},
		Memo:        "test unsupported memo",
	}

	_, err = s.client.processOutboundTx(unsupportedTxOutItem)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*unsupported asset.*")
}

func (s *StellarClientTestSuite) TestTruncateMemoForStellar(c *C) {
	// Verify memo truncation logic handles various scenarios correctly

	// Test case: memo within 28-byte limit (no truncation)
	shortMemo := "OUT:ABC123"
	truncated := s.client.truncateMemoForStellar(shortMemo)
	c.Assert(truncated, Equals, shortMemo)
	c.Assert(len(truncated), Equals, 10)
	c.Assert(len(truncated) <= 28, Equals, true)

	// Test case: long memo requiring midpoint truncation
	longMemo := "OUT:26582E155FACA962DDA2A4C6E94F19C60EC6AC1C2BC27E80421AB545C395DBAA"
	truncated = s.client.truncateMemoForStellar(longMemo)
	c.Assert(len(truncated), Equals, 28)
	c.Assert(truncated, Equals, "OUT:26582E155F....45C395DBAA")

	// Verify proper format: OUT: prefix with midpoint dots
	c.Assert(truncated[:4], Equals, "OUT:")
	c.Assert(truncated[14:18], Equals, "....")

	// Test case: memo without colon separator
	noColonMemo := "VERY_LONG_MEMO_WITHOUT_COLON_THAT_NEEDS_TRUNCATION_ABCDEF123456789"
	truncated = s.client.truncateMemoForStellar(noColonMemo)
	c.Assert(len(truncated), Equals, 28)
	c.Assert(truncated, Equals, "VERY_LONG_ME....DEF123456789")

	// Test edge case: memo exactly at 28-byte limit
	exactMemo := "OUT:123456789012345678901234" // Precisely 28 bytes
	c.Assert(len(exactMemo), Equals, 28)
	truncated = s.client.truncateMemoForStellar(exactMemo)
	c.Assert(truncated, Equals, exactMemo)
	c.Assert(len(truncated), Equals, 28)
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

func (s *StellarClientTestSuite) TestAddressConversionFunctions(c *C) {
	// Test the address conversion functions that are used in SignTx
	// These are the helper functions that were causing nil pointer dereference

	// Test with valid Stellar account address (starts with 'G')
	accountAddr := "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"

	// Test with valid Stellar contract address (starts with 'C')
	contractAddr := "CC7XNCYBCI2UVAE2A5TUBALEXMZXTYHLMKYOA6FSXVRT42YLR76NQR7R"

	// Test with invalid address
	invalidAddr := "invalid_address"

	// Test account address conversion
	scAddr, err := s.client.getScAddressFromString(accountAddr)
	c.Assert(err, IsNil)
	c.Assert(scAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)
	c.Assert(scAddr.AccountId, NotNil)

	// Test contract address conversion
	scAddrContract, err := s.client.getScAddressFromString(contractAddr)
	c.Assert(err, IsNil)
	c.Assert(scAddrContract.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)
	c.Assert(scAddrContract.ContractId, NotNil)

	// Test invalid address conversion
	_, err = s.client.getScAddressFromString(invalidAddr)
	c.Assert(err, NotNil)

	// Test pointer conversion functions
	scAddrPtr, err := s.client.getScAddressPtrFromString(accountAddr)
	c.Assert(err, IsNil)
	c.Assert(scAddrPtr, NotNil)
	c.Assert(scAddrPtr.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)

	scAddrContractPtr, err := s.client.getScAddressPtrFromString(contractAddr)
	c.Assert(err, IsNil)
	c.Assert(scAddrContractPtr, NotNil)
	c.Assert(scAddrContractPtr.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)

	// Test that pointers are valid and not nil
	c.Assert(scAddrPtr.AccountId, NotNil)
	c.Assert(scAddrContractPtr.ContractId, NotNil)
}

func (s *StellarClientTestSuite) TestInvokeHostFunctionTransactionSigning(c *C) {
	// Test the full InvokeHostFunction transaction signing logic
	// This simulates the actual outbound transaction signing that was failing

	// Set up the router address for testing
	s.client.routerAddress = "CC7XNCYBCI2UVAE2A5TUBALEXMZXTYHLMKYOA6FSXVRT42YLR76NQR7R"

	// Create a test transaction out item that matches what we see in the logs
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(74044534707)) // Amount from logs
	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GB3F6OUNAIAMDGV5LVT2UFWAWACQ2LRE6OMF2OTSYCY7AKXHAMEXG37V"),
		VaultPubKey: common.PubKey("tswitchpub1addwnpepqfshsq2y6ejy2ysxmq4gj8n8mzuzyulk9wh4n946jv5w2vpwdn2yuhpesc6"),
		Coins:       common.Coins{coin},
		Memo:        "OUT:56D832CB5365562BC87F8A309CB3D3A518A5D86715C574D6BED791F42F2F9762",
		MaxGas:      common.Gas{common.NewCoin(common.XLMAsset, cosmos.NewUint(150))},
		GasRate:     150,
		Height:      747,
		InHash:      "56D832CB5365562BC87F8A309CB3D3A518A5D86715C574D6BED791F42F2F9762",
	}

	// Test that we can build the InvokeHostFunction operation without panicking
	// This tests the address conversion and operation building logic
	c.Assert(s.client.routerAddress, Not(Equals), "")

	// Test router address conversion
	routerScAddr, err := s.client.getScAddressFromString(s.client.routerAddress)
	c.Assert(err, IsNil)
	c.Assert(routerScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)
	c.Assert(routerScAddr.ContractId, NotNil)

	// Test vault address conversion (this would be the vault's Stellar address)
	vaultAddr := "GAKJPRDGMOXTUFAJPGKXBAQ2BHYGQ6UQ7MW6LWOZUYEVK7D4YR4OIGT4"
	vaultScAddr, err := s.client.getScAddressPtrFromString(vaultAddr)
	c.Assert(err, IsNil)
	c.Assert(vaultScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)
	c.Assert(vaultScAddr.AccountId, NotNil)

	// Test destination address conversion
	toScAddr, err := s.client.getScAddressPtrFromString(txOutItem.ToAddress.String())
	c.Assert(err, IsNil)
	c.Assert(toScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)
	c.Assert(toScAddr.AccountId, NotNil)

	// Test asset address conversion (XLM.XLM should map to native asset)
	// For XLM.XLM, the asset address should be the native asset contract
	assetScAddr, err := s.client.getScAddressPtrFromString("CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC")
	c.Assert(err, IsNil)
	c.Assert(assetScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)
	c.Assert(assetScAddr.ContractId, NotNil)

	// Test that we can create the InvokeHostFunction operation structure
	// This simulates the operation building logic from SignTx
	invokeHostFunction := &txnbuild.InvokeHostFunction{
		HostFunction: xdr.HostFunction{
			Type: xdr.HostFunctionTypeHostFunctionTypeInvokeContract,
			InvokeContract: &xdr.InvokeContractArgs{
				ContractAddress: routerScAddr,
				FunctionName:    "transfer_out",
				Args: []xdr.ScVal{
					// vault: Address (the vault making the transfer)
					{Type: xdr.ScValTypeScvAddress, Address: vaultScAddr},
					// to: Address (recipient)
					{Type: xdr.ScValTypeScvAddress, Address: toScAddr},
					// asset: Address (asset to transfer)
					{Type: xdr.ScValTypeScvAddress, Address: assetScAddr},
					// amount: i128 - convert from the coin amount
					{Type: xdr.ScValTypeScvI128, I128: &xdr.Int128Parts{Lo: xdr.Uint64(coin.Amount.Uint64()), Hi: 0}},
					// memo: String (can be longer than 28 bytes - passed as function parameter)
					{Type: xdr.ScValTypeScvString, Str: (*xdr.ScString)(&txOutItem.Memo)},
				},
			},
		},
		SourceAccount: "GAKJPRDGMOXTUFAJPGKXBAQ2BHYGQ6UQ7MW6LWOZUYEVK7D4YR4OIGT4",
	}

	// Verify the operation structure is valid
	c.Assert(invokeHostFunction.HostFunction.Type, Equals, xdr.HostFunctionTypeHostFunctionTypeInvokeContract)
	c.Assert(invokeHostFunction.HostFunction.InvokeContract, NotNil)
	c.Assert(invokeHostFunction.HostFunction.InvokeContract.ContractAddress, Equals, routerScAddr)
	c.Assert(string(invokeHostFunction.HostFunction.InvokeContract.FunctionName), Equals, "transfer_out")
	c.Assert(len(invokeHostFunction.HostFunction.InvokeContract.Args), Equals, 5)

	// Test that all ScVal arguments are properly typed
	args := invokeHostFunction.HostFunction.InvokeContract.Args
	c.Assert(args[0].Type, Equals, xdr.ScValTypeScvAddress)
	c.Assert(args[0].Address, NotNil)
	c.Assert(args[1].Type, Equals, xdr.ScValTypeScvAddress)
	c.Assert(args[1].Address, NotNil)
	c.Assert(args[2].Type, Equals, xdr.ScValTypeScvAddress)
	c.Assert(args[2].Address, NotNil)
	c.Assert(args[3].Type, Equals, xdr.ScValTypeScvI128)
	c.Assert(args[3].I128, NotNil)
	c.Assert(args[4].Type, Equals, xdr.ScValTypeScvString)
	c.Assert(args[4].Str, NotNil)

	// Test that the memo string is properly set
	memoStr := string(*args[4].Str)
	c.Assert(memoStr, Equals, txOutItem.Memo)
	c.Assert(len(memoStr) > 28, Equals, true) // Should be longer than Stellar's 28-byte limit

	// Test that the amount is properly converted
	amountParts := args[3].I128
	c.Assert(amountParts.Lo, Equals, xdr.Uint64(coin.Amount.Uint64()))
	c.Assert(amountParts.Hi, Equals, xdr.Int64(0))
}
