package gaia

import (
	sdkmath "cosmossdk.io/math"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/config"
	. "gopkg.in/check.v1"
)

type UtilTestSuite struct {
	scanner CosmosBlockScanner
}

var _ = Suite(&UtilTestSuite{})

func (s *UtilTestSuite) SetUpSuite(c *C) {
	cfg := config.BifrostBlockScannerConfiguration{
		ChainID:            common.GAIAChain,
		GasPriceResolution: 100_000,
		WhitelistCosmosAssets: []config.WhitelistCosmosAsset{
			{Denom: "uatom", Decimals: 6, SwitchlySymbol: "ATOM"},
		},
	}
	s.scanner = CosmosBlockScanner{cfg: cfg}
}

func (s *UtilTestSuite) TestFromCosmosToSwitchly(c *C) {
	// 5 ATOM, 6 decimals
	cosmosCoin := cosmos.NewCoin("uatom", sdkmath.NewInt(5000000))
	switchlyCoin, err := s.scanner.fromCosmosToSwitchly(cosmosCoin)
	c.Assert(err, IsNil)

	// 5 ATOM, 8 decimals
	expectedSwitchlyAsset, err := common.NewAsset("GAIA.ATOM")
	c.Assert(err, IsNil)
	expectedSwitchlyAmount := sdkmath.NewUint(500000000)
	c.Check(switchlyCoin.Asset.Equals(expectedSwitchlyAsset), Equals, true)
	c.Check(switchlyCoin.Amount.BigInt().Int64(), Equals, expectedSwitchlyAmount.BigInt().Int64())
	c.Check(switchlyCoin.Decimals, Equals, int64(6))
}

func (s *UtilTestSuite) TestFromSwitchlyToCosmos(c *C) {
	// 6 GAIA.ATOM, 8 decimals
	switchlyAsset, err := common.NewAsset("GAIA.ATOM")
	c.Assert(err, IsNil)
	switchlyCoin := common.Coin{
		Asset:    switchlyAsset,
		Amount:   cosmos.NewUint(600000000),
		Decimals: 6,
	}
	cosmosCoin, err := s.scanner.fromSwitchlyToCosmos(switchlyCoin)
	c.Assert(err, IsNil)

	// 6 uatom, 6 decimals
	expectedCosmosDenom := "uatom"
	expectedCosmosAmount := int64(6000000)
	c.Check(cosmosCoin.Denom, Equals, expectedCosmosDenom)
	c.Check(cosmosCoin.Amount.Int64(), Equals, expectedCosmosAmount)
}
