package keeperv1

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type KeeperReserveContributorsSuite struct{}

var _ = Suite(&KeeperReserveContributorsSuite{})

func (KeeperReserveContributorsSuite) TestReserveContributors(c *C) {
	ctx, k := setupKeeperForTest(c)

	poolFee := cosmos.NewUint(common.One * 100)
	FundModule(c, ctx, k, AsgardName, poolFee.Uint64())
	asgardBefore := k.GetSWITCHBalanceOfModule(ctx, AsgardName)
	reserveBefore := k.GetSWITCHBalanceOfModule(ctx, ReserveName)

	c.Assert(k.AddPoolFeeToReserve(ctx, poolFee), IsNil)

	asgardAfter := k.GetSWITCHBalanceOfModule(ctx, AsgardName)
	reserveAfter := k.GetSWITCHBalanceOfModule(ctx, ReserveName)
	c.Assert(asgardAfter.String(), Equals, asgardBefore.Sub(poolFee).String())
	c.Assert(reserveAfter.String(), Equals, reserveBefore.Add(poolFee).String())

	bondFee := cosmos.NewUint(common.One * 200)
	FundModule(c, ctx, k, BondName, bondFee.Uint64())
	bondBefore := k.GetSWITCHBalanceOfModule(ctx, BondName)
	reserveBefore = reserveAfter

	c.Assert(k.AddBondFeeToReserve(ctx, bondFee), IsNil)

	bondAfter := k.GetSWITCHBalanceOfModule(ctx, BondName)
	reserveAfter = k.GetSWITCHBalanceOfModule(ctx, ReserveName)
	c.Assert(bondAfter.String(), Equals, bondBefore.Sub(bondFee).String())
	c.Assert(reserveAfter.String(), Equals, reserveBefore.Add(bondFee).String())
}
