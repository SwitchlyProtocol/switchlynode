package types

import (
	. "gopkg.in/check.v1"

	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type ProtocolOwnedLiquiditySuite struct{}

var _ = Suite(&ProtocolOwnedLiquiditySuite{})

func (s *ProtocolOwnedLiquiditySuite) TestCalcNodeRewards(c *C) {
	pol := NewProtocolOwnedLiquidity()
	c.Check(pol.SwitchDeposited.Uint64(), Equals, cosmos.ZeroUint().Uint64())
	c.Check(pol.SwitchWithdrawn.Uint64(), Equals, cosmos.ZeroUint().Uint64())
}

func (s *ProtocolOwnedLiquiditySuite) TestCurrentDeposit(c *C) {
	pol := NewProtocolOwnedLiquidity()
	pol.SwitchDeposited = cosmos.NewUint(100)
	pol.SwitchWithdrawn = cosmos.NewUint(25)
	c.Check(pol.CurrentDeposit().Int64(), Equals, int64(75))

	pol = NewProtocolOwnedLiquidity()
	pol.SwitchDeposited = cosmos.NewUint(25)
	pol.SwitchWithdrawn = cosmos.NewUint(100)
	c.Check(pol.CurrentDeposit().Int64(), Equals, int64(-75))
}

func (s *ProtocolOwnedLiquiditySuite) PnL(c *C) {
	pol := NewProtocolOwnedLiquidity()
	pol.SwitchDeposited = cosmos.NewUint(100)
	pol.SwitchWithdrawn = cosmos.NewUint(25)
	c.Check(pol.PnL(cosmos.NewUint(30)).Int64(), Equals, int64(-45))

	pol = NewProtocolOwnedLiquidity()
	pol.SwitchDeposited = cosmos.NewUint(25)
	pol.SwitchWithdrawn = cosmos.NewUint(10)
	c.Check(pol.PnL(cosmos.NewUint(30)).Int64(), Equals, int64(15))
}
