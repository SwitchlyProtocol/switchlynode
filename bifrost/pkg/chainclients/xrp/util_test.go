package xrp

import (
	sdkmath "cosmossdk.io/math"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	. "gopkg.in/check.v1"

	txtypes "github.com/Peersyst/xrpl-go/xrpl/transaction/types"
)

type UtilTestSuite struct{}

var _ = Suite(&UtilTestSuite{})

func (s *UtilTestSuite) SetUpSuite(c *C) {}

func (s *UtilTestSuite) TestFromXrpToSwitchly(c *C) {
	// 5 XRP, 6 decimals
	switchlyCoin, err := fromXrpToSwitchly(txtypes.XRPCurrencyAmount(uint64(5000000)))
	c.Assert(err, IsNil)

	// 5 XRP, 8 decimals
	expectedCoin := common.Coin{
		Asset:    common.XRPAsset,
		Amount:   sdkmath.NewUint(500000000),
		Decimals: 6,
	}
	c.Check(switchlyCoin.Asset.Equals(expectedCoin.Asset), Equals, true)
	c.Check(switchlyCoin.Amount.String(), Equals, expectedCoin.Amount.String())
	c.Check(switchlyCoin.Decimals, Equals, expectedCoin.Decimals)
}

func (s *UtilTestSuite) TestFromSwitchlyToXrp(c *C) {
	// 6 XRP, 8 decimals
	switchlyCoin := common.NewCoin(common.XRPAsset, sdkmath.NewUint(600000000))
	xrpCurrency, err := fromSwitchlyToXrp(switchlyCoin)
	c.Assert(err, IsNil)

	// 6 XRP, 6 decimals
	xrpCoin, ok := xrpCurrency.(txtypes.XRPCurrencyAmount)
	c.Check(ok, Equals, true)
	c.Check(xrpCoin, Equals, txtypes.XRPCurrencyAmount(6000000))
}
