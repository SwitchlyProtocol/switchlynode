package stellar

import (
	"testing"

	"github.com/stellar/go/txnbuild"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	. "gopkg.in/check.v1"
)

type AssetMappingTestSuite struct{}

var _ = Suite(&AssetMappingTestSuite{})

func TestAssetMapping(t *testing.T) {
	TestingT(t)
}

func (s *AssetMappingTestSuite) TestGetAssetByStellarAsset(c *C) {
	// Test native XLM
	mapping, found := GetAssetByStellarAsset("native", "", "")
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "native")
	c.Assert(mapping.THORChainAsset.Equals(common.XLMAsset), Equals, true)

	// Test USDC
	mapping, found = GetAssetByStellarAsset("credit_alphanum4", "USDC", "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(mapping.StellarAssetIssuer, Equals, "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")

	// Test unsupported asset
	mapping, found = GetAssetByStellarAsset("credit_alphanum4", "UNKNOWN", "UNKNOWN_ISSUER")
	c.Assert(found, Equals, false)
}

func (s *AssetMappingTestSuite) TestGetAssetByTHORChainAsset(c *C) {
	// Test XLM asset
	mapping, found := GetAssetByTHORChainAsset(common.XLMAsset)
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetType, Equals, "native")

	// Test USDC asset
	usdcAsset := common.Asset{Chain: common.StellarChain, Symbol: "USDC-GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN", Ticker: "USDC"}
	mapping, found = GetAssetByTHORChainAsset(usdcAsset)
	c.Assert(found, Equals, true)
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")

	// Test unsupported asset
	unknownAsset := common.Asset{Chain: common.StellarChain, Symbol: "UNKNOWN", Ticker: "UNKNOWN"}
	mapping, found = GetAssetByTHORChainAsset(unknownAsset)
	c.Assert(found, Equals, false)
}

func (s *AssetMappingTestSuite) TestToStellarAsset(c *C) {
	// Test native asset conversion
	xlmMapping, _ := GetAssetByStellarAsset("native", "", "")
	stellarAsset := xlmMapping.ToStellarAsset()
	_, isNative := stellarAsset.(txnbuild.NativeAsset)
	c.Assert(isNative, Equals, true)

	// Test credit asset conversion
	usdcMapping, _ := GetAssetByStellarAsset("credit_alphanum4", "USDC", "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")
	stellarAsset = usdcMapping.ToStellarAsset()
	creditAsset, isCredit := stellarAsset.(txnbuild.CreditAsset)
	c.Assert(isCredit, Equals, true)
	c.Assert(creditAsset.Code, Equals, "USDC")
	c.Assert(creditAsset.Issuer, Equals, "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")
}

func (s *AssetMappingTestSuite) TestFromStellarAsset(c *C) {
	// Test native asset
	nativeAsset := txnbuild.NativeAsset{}
	mapping, err := FromStellarAsset(nativeAsset)
	c.Assert(err, IsNil)
	c.Assert(mapping.StellarAssetType, Equals, "native")
	c.Assert(mapping.THORChainAsset.Equals(common.XLMAsset), Equals, true)

	// Test credit asset
	creditAsset := txnbuild.CreditAsset{
		Code:   "USDC",
		Issuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
	}
	mapping, err = FromStellarAsset(creditAsset)
	c.Assert(err, IsNil)
	c.Assert(mapping.StellarAssetCode, Equals, "USDC")
	c.Assert(mapping.StellarAssetIssuer, Equals, "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")

	// Test unsupported asset
	unsupportedAsset := txnbuild.CreditAsset{
		Code:   "UNKNOWN",
		Issuer: "UNKNOWN_ISSUER",
	}
	mapping, err = FromStellarAsset(unsupportedAsset)
	c.Assert(err, NotNil)
}

func (s *AssetMappingTestSuite) TestConvertToTHORChainAmount(c *C) {
	// Test XLM conversion (7 decimals to 8 decimals)
	xlmMapping, _ := GetAssetByStellarAsset("native", "", "")
	coin, err := xlmMapping.ConvertToTHORChainAmount("10000000") // 1 XLM in stroops
	c.Assert(err, IsNil)
	c.Assert(coin.Asset.Equals(common.XLMAsset), Equals, true)
	c.Assert(coin.Amount.Equal(cosmos.NewUint(100000000)), Equals, true) // 1 XLM in THORChain units

	// Test USDC conversion (7 decimals to 8 decimals)
	usdcMapping, _ := GetAssetByStellarAsset("credit_alphanum4", "USDC", "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")
	coin, err = usdcMapping.ConvertToTHORChainAmount("10000000") // 1 USDC in Stellar units
	c.Assert(err, IsNil)
	c.Assert(coin.Amount.Equal(cosmos.NewUint(100000000)), Equals, true) // 1 USDC in THORChain units
}

func (s *AssetMappingTestSuite) TestConvertFromTHORChainAmount(c *C) {
	// Test XLM conversion (8 decimals to 7 decimals)
	xlmMapping, _ := GetAssetByStellarAsset("native", "", "")
	stellarAmount := xlmMapping.ConvertFromTHORChainAmount(cosmos.NewUint(100000000)) // 1 XLM in THORChain units
	c.Assert(stellarAmount, Equals, "10000000")                                       // 1 XLM in stroops

	// Test USDC conversion (8 decimals to 7 decimals)
	usdcMapping, _ := GetAssetByStellarAsset("credit_alphanum4", "USDC", "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN")
	stellarAmount = usdcMapping.ConvertFromTHORChainAmount(cosmos.NewUint(100000000)) // 1 USDC in THORChain units
	c.Assert(stellarAmount, Equals, "10000000")                                       // 1 USDC in Stellar units
}
