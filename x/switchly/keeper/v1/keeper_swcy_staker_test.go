package keeperv1

import (
	"cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	. "gopkg.in/check.v1"
)

type KeeperSWCYStakerSuite struct{}

var _ = Suite(&KeeperSWCYStakerSuite{})

func (s *KeeperSWCYStakerSuite) TestSWCYStaker(c *C) {
	ctx, k := setupKeeperForTest(c)
	initStakers := []SWCYStaker{
		{
			Address: GetRandomSwitchAddress(),
			Amount:  math.NewUint(1 * common.One),
		},
		{
			Address: GetRandomSwitchAddress(),
			Amount:  math.NewUint(10 * common.One),
		},
		{
			Address: GetRandomSwitchAddress(),
			Amount:  math.NewUint(100 * common.One),
		},
		{
			Address: GetRandomSwitchAddress(),
			Amount:  math.NewUint(1000 * common.One),
		},
		{
			Address: GetRandomSwitchAddress(),
			Amount:  math.NewUint(10000 * common.One),
		},
	}

	// Set stakers
	for _, staker := range initStakers {
		c.Assert(k.SetSWCYStaker(ctx, staker), IsNil)
	}

	stakers, err := k.ListSWCYStakers(ctx)
	c.Assert(err, IsNil)

	// Include SWCY smart contract staker
	expectedLen := len(initStakers) + 1
	c.Assert(len(stakers), Equals, expectedLen)

	var staker SWCYStaker
	for _, initStaker := range initStakers {
		c.Assert(k.SWCYStakerExists(ctx, initStaker.Address), Equals, true)
		staker, err = k.GetSWCYStaker(ctx, initStaker.Address)
		c.Assert(err, IsNil)
		c.Assert(staker.Amount.Equal(initStaker.Amount), Equals, true)
	}

	// Delete stakers
	for _, staker := range initStakers {
		k.DeleteSWCYStaker(ctx, staker.Address)
	}

	stakers, err = k.ListSWCYStakers(ctx)
	c.Assert(err, IsNil)

	// Just SWCY smart contract staker
	c.Assert(len(stakers), Equals, 1)

	for _, staker := range initStakers {
		c.Assert(k.SWCYStakerExists(ctx, staker.Address), Equals, false)
	}
}
