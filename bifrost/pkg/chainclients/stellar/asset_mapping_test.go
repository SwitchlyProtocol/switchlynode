package stellar

import (
	"github.com/stellar/go/txnbuild"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	. "gopkg.in/check.v1"
)

type AssetMappingTestSuite struct{}

var _ = Suite(&AssetMappingTestSuite{})

func (s *AssetMappingTestSuite) SetUpTest(c *C) {
	// Reset to testnet for consistent testing
	SetNetwork(StellarTestnet)
}

func (s *AssetMappingTestSuite) TestNetworkConfiguration(c *C) {
	// Test network setting and retrieval
	SetNetwork(StellarMainnet)
	c.Assert(GetCurrentNetwork(), Equals, StellarMainnet)

	SetNetwork(StellarTestnet)
	c.Assert(GetCurrentNetwork(), Equals, StellarTestnet)
}

func (s *AssetMappingTestSuite) TestGetAssetByStellarAsset(c *C) {
	// Test native XLM
	mapping, found := GetAssetByStellarAsset("native", "XLM", "")
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "native")
	c.Assert(mapping.StellarAssetCode, Equals, "XLM")
	c.Assert(mapping.StellarAssetIssuer, Equals, "")
	c.Assert(mapping.StellarDecimals, Equals, 7)
	c.Assert(string(mapping.SwitchlyProtocolAsset.Symbol), Equals, "XLM")

	// Test Soroban USDC token - should use testnet address after network setup
	SetNetwork(StellarTestnet)
	mapping, found = GetAssetByStellarAsset("contract", "USDC", "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "contract")
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(mapping.StellarAssetIssuer, Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	c.Assert(mapping.StellarDecimals, Equals, 7)
	c.Assert(string(mapping.SwitchlyProtocolAsset.Symbol), Equals, "USDC")

	// Test unknown asset
	_, found = GetAssetByStellarAsset("contract", "UNKNOWN", "UNKNOWN_ISSUER")
	c.Assert(found, Equals, false)
}

func (s *AssetMappingTestSuite) TestGetAssetByTHORChainAsset(c *C) {
	// Test native XLM
	mapping, found := GetAssetBySwitchlyProtocolAsset(common.Asset{Chain: common.StellarChain, Symbol: "XLM", Ticker: "XLM"})
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "native")

	// Test Soroban USDC with network-agnostic symbol
	usdcAsset := common.Asset{Chain: common.StellarChain, Symbol: "USDC", Ticker: "USDC"}
	mapping, found = GetAssetBySwitchlyProtocolAsset(usdcAsset)
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "contract")
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")

	// Test unknown asset
	unknownAsset := common.Asset{Chain: common.StellarChain, Symbol: "UNKNOWN", Ticker: "UNKNOWN"}
	mapping, found = GetAssetBySwitchlyProtocolAsset(unknownAsset)
	c.Assert(found, Equals, false)
}

func (s *AssetMappingTestSuite) TestGetAssetByAddress(c *C) {
	// Test native XLM by "native" address
	mapping, found := GetAssetByAddress("native")
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "native")
	c.Assert(mapping.StellarAssetCode, Equals, "XLM")
	c.Assert(string(mapping.SwitchlyProtocolAsset.Symbol), Equals, "XLM")

	// Test Soroban USDC by testnet contract address
	SetNetwork(StellarTestnet)
	mapping, found = GetAssetByAddress("CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "contract")
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(string(mapping.SwitchlyProtocolAsset.Symbol), Equals, "USDC")

	// Test unknown address
	_, found = GetAssetByAddress("UNKNOWN_ADDRESS")
	c.Assert(found, Equals, false)
}

func (s *AssetMappingTestSuite) TestGetAssetByAddressAndNetwork(c *C) {
	// Test Soroban USDC by testnet contract address
	mapping, found := GetAssetByAddressAndNetwork("CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA", StellarTestnet)
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "contract")
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(mapping.StellarAssetIssuer, Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")

	// Test Soroban USDC by mainnet contract address
	mapping, found = GetAssetByAddressAndNetwork("CCW67TSZV3SSS2HXMBQ5JFGCKJNXKZM7UQUWUZPUTHXSTZLEO7SJMI75", StellarMainnet)
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "contract")
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(mapping.StellarAssetIssuer, Equals, "CCW67TSZV3SSS2HXMBQ5JFGCKJNXKZM7UQUWUZPUTHXSTZLEO7SJMI75")

	// Test unknown address
	mapping, found = GetAssetByAddressAndNetwork("UNKNOWN_ADDRESS", StellarTestnet)
	c.Assert(found, Equals, false)
}

func (s *AssetMappingTestSuite) TestGetContractAddressForNetwork(c *C) {
	// Test USDC contract address retrieval
	addr, found := GetContractAddressForNetwork("USDC", StellarMainnet)
	c.Assert(found, Equals, true)
	c.Assert(addr, Equals, "CCW67TSZV3SSS2HXMBQ5JFGCKJNXKZM7UQUWUZPUTHXSTZLEO7SJMI75")

	addr, found = GetContractAddressForNetwork("USDC", StellarTestnet)
	c.Assert(found, Equals, true)
	c.Assert(addr, Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")

	// Test unknown asset
	addr, found = GetContractAddressForNetwork("UNKNOWN", StellarMainnet)
	c.Assert(found, Equals, false)
	c.Assert(addr, Equals, "")
}

func (s *AssetMappingTestSuite) TestIsAssetAvailableOnNetwork(c *C) {
	// Test native asset (available on all networks)
	c.Assert(IsAssetAvailableOnNetwork("XLM", StellarMainnet), Equals, true)
	c.Assert(IsAssetAvailableOnNetwork("XLM", StellarTestnet), Equals, true)

	// Test Soroban token (network-specific)
	c.Assert(IsAssetAvailableOnNetwork("USDC", StellarMainnet), Equals, true)
	c.Assert(IsAssetAvailableOnNetwork("USDC", StellarTestnet), Equals, true)

	// Test unknown asset
	c.Assert(IsAssetAvailableOnNetwork("UNKNOWN", StellarMainnet), Equals, false)
}

func (s *AssetMappingTestSuite) TestGetAllNetworksForAsset(c *C) {
	// Test USDC networks
	networks := GetAllNetworksForAsset("USDC")
	c.Assert(len(networks), Equals, 2)
	c.Assert(contains(networks, StellarMainnet), Equals, true)
	c.Assert(contains(networks, StellarTestnet), Equals, true)

	// Test unknown asset
	networks = GetAllNetworksForAsset("UNKNOWN")
	c.Assert(len(networks), Equals, 0)
}

// Helper function to check if slice contains value
func contains(slice []StellarNetwork, value StellarNetwork) bool {
	for _, item := range slice {
		if item == value {
			return true
		}
	}
	return false
}

func (s *AssetMappingTestSuite) TestToStellarAsset(c *C) {
	// Test native asset conversion
	xlmMapping, _ := GetAssetByStellarAsset("native", "XLM", "")
	stellarAsset := xlmMapping.ToStellarAsset()
	_, isNative := stellarAsset.(txnbuild.NativeAsset)
	c.Assert(isNative, Equals, true)

	// Test contract asset conversion
	SetNetwork(StellarTestnet)
	usdcMapping, _ := GetAssetByStellarAsset("contract", "USDC", "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	stellarAsset = usdcMapping.ToStellarAsset()
	creditAsset, isCredit := stellarAsset.(txnbuild.CreditAsset)
	c.Assert(isCredit, Equals, true)
	c.Assert(creditAsset.Code, Equals, "USDC")
	c.Assert(creditAsset.Issuer, Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
}

func (s *AssetMappingTestSuite) TestFromStellarAsset(c *C) {
	// Test native asset
	nativeAsset := txnbuild.NativeAsset{}
	mapping, err := FromStellarAsset(nativeAsset)
	c.Assert(err, IsNil)
	c.Assert(mapping.StellarAssetType, Equals, "native")
	c.Assert(mapping.StellarAssetCode, Equals, "XLM")
	c.Assert(mapping.StellarAssetIssuer, Equals, "")

	// Test contract asset (Soroban token)
	SetNetwork(StellarTestnet)
	contractAsset := txnbuild.CreditAsset{
		Code:   "USDC",
		Issuer: "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA",
	}
	mapping, err = FromStellarAsset(contractAsset)
	c.Assert(err, IsNil)
	c.Assert(mapping.StellarAssetType, Equals, "contract")
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(mapping.StellarAssetIssuer, Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")

	// Test unknown asset
	unknownAsset := txnbuild.CreditAsset{
		Code:   "UNKNOWN",
		Issuer: "UNKNOWN_ISSUER",
	}
	_, err = FromStellarAsset(unknownAsset)
	c.Assert(err, NotNil)
}

func (s *AssetMappingTestSuite) TestConvertToTHORChainAmount(c *C) {
	// Test XLM conversion (7 decimals to 8 decimals)
	mapping, _ := GetAssetByStellarAsset("native", "XLM", "")
	coin, err := mapping.ConvertToSwitchlyProtocolAmount("10000000") // 1 XLM in stroops
	c.Assert(err, IsNil)
	c.Assert(coin.Amount.Uint64(), Equals, uint64(100000000)) // 1 XLM in THORChain units (1e8)

	// Test USDC conversion (7 decimals to 8 decimals) - use testnet address
	SetNetwork(StellarTestnet)
	mapping, _ = GetAssetByStellarAsset("contract", "USDC", "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	coin, err = mapping.ConvertToSwitchlyProtocolAmount("10000000") // 1 USDC in 7-decimal format
	c.Assert(err, IsNil)
	c.Assert(coin.Amount.Uint64(), Equals, uint64(100000000)) // 1 USDC in THORChain units (1e8)
}

func (s *AssetMappingTestSuite) TestConvertFromTHORChainAmount(c *C) {
	// Test XLM conversion (8 decimals to 7 decimals)
	mapping, _ := GetAssetByStellarAsset("native", "XLM", "")
	amount := mapping.ConvertFromSwitchlyProtocolAmount(cosmos.NewUint(100000000)) // 1 XLM in THORChain units
	c.Assert(amount, Equals, "10000000")                                    // 1 XLM in stroops

	// Test USDC conversion (8 decimals to 7 decimals) - use testnet address
	SetNetwork(StellarTestnet)
	mapping, _ = GetAssetByStellarAsset("contract", "USDC", "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	amount = mapping.ConvertFromSwitchlyProtocolAmount(cosmos.NewUint(100000000)) // 1 USDC in THORChain units
	c.Assert(amount, Equals, "10000000")                                   // 1 USDC in 7-decimal format
}

func (s *AssetMappingTestSuite) TestSorobanTokenHelpers(c *C) {
	// Test native asset
	nativeMapping, _ := GetAssetByStellarAsset("native", "XLM", "")
	c.Assert(IsNativeAsset(nativeMapping), Equals, true)
	c.Assert(IsSorobanToken(nativeMapping), Equals, false)
	c.Assert(IsClassicAsset(nativeMapping), Equals, false)
	c.Assert(GetTokenContractAddress(nativeMapping), Equals, "")

	// Test Soroban token
	SetNetwork(StellarTestnet)
	sorobanMapping, _ := GetAssetByStellarAsset("contract", "USDC", "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
	c.Assert(IsNativeAsset(sorobanMapping), Equals, false)
	c.Assert(IsSorobanToken(sorobanMapping), Equals, true)
	c.Assert(IsClassicAsset(sorobanMapping), Equals, false)
	c.Assert(GetTokenContractAddress(sorobanMapping), Equals, "CBIELTK6YBZJU5UP2WWQEUCYKLPU6AUNZ2BQ4WWFEIE3USCIHMXQDAMA")
}

func (s *AssetMappingTestSuite) TestUpdateContractAddress(c *C) {
	// Test updating contract address
	success := UpdateContractAddress("USDC", StellarMainnet, "NEW_MAINNET_ADDRESS")
	c.Assert(success, Equals, true)

	// Verify the address was updated
	addr, found := GetContractAddressForNetwork("USDC", StellarMainnet)
	c.Assert(found, Equals, true)
	c.Assert(addr, Equals, "NEW_MAINNET_ADDRESS")

	// Test updating unknown asset
	success = UpdateContractAddress("UNKNOWN", StellarMainnet, "SOME_ADDRESS")
	c.Assert(success, Equals, false)
}
