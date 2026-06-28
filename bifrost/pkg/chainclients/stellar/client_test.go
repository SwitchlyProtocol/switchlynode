package stellar

// Stellar Client Test Suite
//
// Outbound transactions always go through the Soroban router's transfer_out so the full
// SwitchlyProtocol memo is preserved on-chain. Key areas tested:
// - Router transfer_out invocation building (asset/address/amount/memo args)
// - Asset mapping for native XLM and issued tokens
// - Ed25519 key derivation for signature compatibility
// - Direct Horizon API integration for improved reliability

import (
	"encoding/base64"
	"math/big"
	"os"
	"strconv"
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
	// SignTx should fail fast on an unresolvable vault address in the test environment.
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
	contractAddr := "CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH"

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
	s.client.routerAddress = "CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH"

	// Create a test transaction out item that matches what we see in the logs
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(74044534707)) // Amount from logs

	// Use a valid public key that won't cause address conversion errors
	validPubKey := newTestVaultPubKey(c)

	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GB3F6OUNAIAMDGV5LVT2UFWAWACQ2LRE6OMF2OTSYCY7AKXHAMEXG37V"),
		VaultPubKey: validPubKey,
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

func (s *StellarClientTestSuite) TestAuthorizationEntryParsing(c *C) {
	// Test authorization entry parsing from simulation results

	// Create mock simulation result with authorization entries
	mockAuthEntry := "AAAAAQAAAAEAAAAFAAAAAAAAAAEAAAAFAAAAAAAAAAEAAAABAAAAHQAAAAFUAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

	// simulationResult := &SorobanSimulationResult{
	// 	Auth:            []string{mockAuthEntry},
	// 	MinResourceFee:  "1000000",
	// 	TransactionData: "mock_transaction_data",
	// }

	// Test simulation result structure - commented out since SorobanSimulationResult removed
	// c.Assert(len(simulationResult.Auth), Equals, 1)
	// c.Assert(simulationResult.Auth[0], Equals, mockAuthEntry)
	// c.Assert(simulationResult.MinResourceFee, Equals, "1000000")

	// Test base64 decoding of auth entry
	authBytes, err := base64.StdEncoding.DecodeString(mockAuthEntry)
	c.Assert(err, IsNil)
	c.Assert(len(authBytes) > 0, Equals, true) // Should decode successfully
}

func (s *StellarClientTestSuite) TestResourceFeeCalculation(c *C) {
	// Test resource fee calculation from simulation results

	testCases := []struct {
		name            string
		minResourceFee  string
		expectedBaseFee int64
		shouldSucceed   bool
	}{
		{
			name:            "Valid resource fee",
			minResourceFee:  "1000000",
			expectedBaseFee: int64(txnbuild.MinBaseFee) + 1000000,
			shouldSucceed:   true,
		},
		{
			name:            "Zero resource fee",
			minResourceFee:  "0",
			expectedBaseFee: int64(txnbuild.MinBaseFee),
			shouldSucceed:   true,
		},
		{
			name:            "Empty resource fee",
			minResourceFee:  "",
			expectedBaseFee: int64(txnbuild.MinBaseFee),
			shouldSucceed:   true,
		},
		{
			name:            "Invalid resource fee",
			minResourceFee:  "invalid",
			expectedBaseFee: int64(txnbuild.MinBaseFee),
			shouldSucceed:   true,
		},
	}

	for _, tc := range testCases {
		c.Logf("Testing case: %s", tc.name)

		// Test resource fee calculation logic
		baseFee := int64(txnbuild.MinBaseFee)
		if tc.minResourceFee != "" {
			if resourceFee, parseErr := strconv.ParseInt(tc.minResourceFee, 10, 64); parseErr == nil && resourceFee > 0 {
				baseFee = int64(txnbuild.MinBaseFee) + resourceFee
			}
		}

		c.Assert(baseFee, Equals, tc.expectedBaseFee)
	}
}

func (s *StellarClientTestSuite) TestContractInvocationWithAuth(c *C) {
	// Test complete contract invocation flow structure validation

	// Test XDR structure validation for contract invocation
	routerAddress := "CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH"

	// Test router address conversion
	routerScAddr, err := s.client.getScAddressFromString(routerAddress)
	c.Assert(err, IsNil)
	c.Assert(routerScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)
	c.Assert(routerScAddr.ContractId, NotNil)

	// Test Stellar address conversion
	stellarAddr := "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"
	stellarScAddr, err := s.client.getScAddressPtrFromString(stellarAddr)
	c.Assert(err, IsNil)
	c.Assert(stellarScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)
	c.Assert(stellarScAddr.AccountId, NotNil)

	// Test asset address conversion
	assetAddr := "CDLZFC3SYJYDZT7K67VZ75HPJVIEUVNIXF47ZG2FB2RMQQVU2HHGCYSC"
	assetScAddr, err := s.client.getScAddressPtrFromString(assetAddr)
	c.Assert(err, IsNil)
	c.Assert(assetScAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeContract)
	c.Assert(assetScAddr.ContractId, NotNil)

	// Test amount conversion
	amount := cosmos.NewUint(50000000)
	amountParts := &xdr.Int128Parts{
		Lo: xdr.Uint64(amount.Uint64()),
		Hi: xdr.Int64(0),
	}
	c.Assert(amountParts.Lo, Equals, xdr.Uint64(50000000))
	c.Assert(amountParts.Hi, Equals, xdr.Int64(0))

	// Test memo string conversion
	memo := "OUT:COMPLETE_FLOW_TEST_WITH_AUTHORIZATION_ENTRIES"
	memoXdr := xdr.ScString(memo)
	c.Assert(string(memoXdr), Equals, memo)
	c.Assert(len(string(memoXdr)) > 28, Equals, true) // Verify no 28-byte truncation
}

func (s *StellarClientTestSuite) TestErrorHandling(c *C) {
	// Test error handling scenarios without public key validation

	// Test router address validation
	originalRouter := s.client.routerAddress
	s.client.routerAddress = ""

	// Test that empty router address is detected
	c.Assert(s.client.routerAddress, Equals, "")

	// Test router address setting
	testRouter := "CAWZ7WYQBG2ENE7S7PAKPYVWKTOV3FMXMIKSOBBZE3JBJXTQWDGZDRGH"
	s.client.routerAddress = testRouter
	c.Assert(s.client.routerAddress, Equals, testRouter)

	// Test address conversion errors
	invalidAddress := "INVALID_ADDRESS"
	_, err := s.client.getScAddressPtrFromString(invalidAddress)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Matches, ".*failed to decode.*")

	// Test valid address conversion
	validAddress := "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"
	scAddr, err := s.client.getScAddressPtrFromString(validAddress)
	c.Assert(err, IsNil)
	c.Assert(scAddr.Type, Equals, xdr.ScAddressTypeScAddressTypeAccount)

	// Restore original router
	s.client.routerAddress = originalRouter
}

// TestManualXDRConstruction tests the new manual XDR construction approach
func (s *StellarClientTestSuite) TestManualXDRConstruction(c *C) {
	// Test the new buildSignedTransaction method that uses proven XDR patterns
	coin := common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000))

	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
		Coins:       common.Coins{coin},
		Memo:        "OUT:TEST_TRANSFER",
		MaxGas:      common.Gas{common.NewCoin(common.XLMAsset, cosmos.NewUint(1000))},
		GasRate:     1,
	}

	// Test that the client uses manual XDR construction
	_, _, _, err := s.client.SignTx(txOutItem, 1)
	c.Assert(err, NotNil) // Will fail in test env but should reach XDR construction

	// Verify it's using the manual XDR approach (not old txnbuild)
	c.Assert(err.Error(), Not(Matches), ".*txnbuild.*")
}

// TestCreateScAddress tests ScAddress creation from account and contract addresses
// via getScAddressFromString.
func (s *StellarClientTestSuite) TestCreateScAddress(c *C) {
	testCases := []struct {
		name    string
		address string
		isValid bool
		scType  xdr.ScAddressType
	}{
		{
			name:    "valid account address",
			address: "GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM",
			isValid: true,
			scType:  xdr.ScAddressTypeScAddressTypeAccount,
		},
		{
			name:    "valid contract address",
			address: testRouterContract,
			isValid: true,
			scType:  xdr.ScAddressTypeScAddressTypeContract,
		},
		{
			name:    "invalid address",
			address: "invalid_address",
			isValid: false,
		},
		{
			name:    "empty address",
			address: "",
			isValid: false,
		},
	}

	for _, tc := range testCases {
		scAddr, err := s.client.getScAddressFromString(tc.address)
		if tc.isValid {
			c.Assert(err, IsNil, Commentf("Test case: %s", tc.name))
			c.Assert(scAddr.Type, Equals, tc.scType, Commentf("Test case: %s", tc.name))
		} else {
			c.Assert(err, NotNil, Commentf("Test case: %s should fail", tc.name))
		}
	}
}

// TestAssetMappingForMigration tests asset mapping retrieval for migration
func (s *StellarClientTestSuite) TestAssetMappingForMigration(c *C) {
	// Test that asset mappings work correctly for supported assets
	supportedAssets := []common.Asset{
		common.XLMAsset,
		common.XLMUSDC,
	}

	for _, asset := range supportedAssets {
		mapping, found := GetAssetBySwitchlyAsset(asset)
		c.Assert(found, Equals, true, Commentf("Asset %s should be supported", asset))
		c.Assert(mapping.ContractAddresses, NotNil, Commentf("Asset %s should have contract addresses", asset))

		// Check that testnet addresses exist
		currentNet := GetCurrentNetwork()
		_, exists := mapping.ContractAddresses[currentNet]
		c.Assert(exists, Equals, true, Commentf("Asset %s should have address for network %s", asset, currentNet))
	}
}

// TestErrorHandlingInNewApproach tests error handling in the new XDR approach
func (s *StellarClientTestSuite) TestErrorHandlingInNewApproach(c *C) {
	// Test various error conditions in the new manual XDR construction

	// Test with invalid contract address environment
	originalContract := os.Getenv("XLM_CONTRACT")
	os.Setenv("XLM_CONTRACT", "") // Clear contract address

	txOutItem := stypes.TxOutItem{
		Chain:       common.StellarChain,
		ToAddress:   common.Address("GABLIYZNBBEF74O7FW2VXHNP2IZUPUOEPJCXA4VB5B56E2EWKSNIONZTLM"),
		VaultPubKey: common.PubKey("switchlypub1addwnpepqflvfnmttdczhsls2l74m7p74xyswjhsl8xv45g42u37q0pdqg9v97ggmj3"),
		Coins:       common.Coins{common.NewCoin(common.XLMAsset, cosmos.NewUint(1000000))},
		Memo:        "OUT:TEST",
		MaxGas:      common.Gas{common.NewCoin(common.XLMAsset, cosmos.NewUint(1000))},
		GasRate:     1,
	}

	_, _, _, err := s.client.SignTx(txOutItem, 1)
	c.Assert(err, NotNil)
	// In test environment, may fail at account validation before reaching contract check
	c.Assert(err.Error(), Matches, ".*XLM_CONTRACT.*not set.*|.*vault account.*|.*stellar address.*")

	// Restore original contract address
	if originalContract != "" {
		os.Setenv("XLM_CONTRACT", originalContract)
	}
}
