package keeperv1

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

type KeeperOutboundFeesSuite struct{}

var _ = Suite(&KeeperOutboundFeesSuite{})

func (s *KeeperOutboundFeesSuite) TestOutboundSWITCHRecords(c *C) {
	ctx, k := setupKeeperForTest(c)

	// Nothing set returns 0.
	feeWithheldSWITCH, err := k.GetOutboundFeeWithheldSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	feeSpentSWITCH, err := k.GetOutboundFeeSpentSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	c.Check(feeWithheldSWITCH.String(), Equals, "0")
	c.Check(feeSpentSWITCH.String(), Equals, "0")

	// Adding sets.
	err = k.AddToOutboundFeeWithheldSwitch(ctx, common.BTCAsset, cosmos.NewUint(uint64(200)))
	c.Assert(err, IsNil)
	err = k.AddToOutboundFeeSpentSwitch(ctx, common.BTCAsset, cosmos.NewUint(uint64(100)))
	c.Assert(err, IsNil)

	feeWithheldSWITCH, err = k.GetOutboundFeeWithheldSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	feeSpentSWITCH, err = k.GetOutboundFeeSpentSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	c.Check(feeWithheldSWITCH.String(), Equals, "200")
	c.Check(feeSpentSWITCH.String(), Equals, "100")

	// Adding again adds.
	err = k.AddToOutboundFeeWithheldSwitch(ctx, common.BTCAsset, cosmos.NewUint(uint64(400)))
	c.Assert(err, IsNil)
	err = k.AddToOutboundFeeSpentSwitch(ctx, common.BTCAsset, cosmos.NewUint(uint64(300)))
	c.Assert(err, IsNil)

	feeWithheldSWITCH, err = k.GetOutboundFeeWithheldSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	feeSpentSWITCH, err = k.GetOutboundFeeSpentSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	c.Check(feeWithheldSWITCH.String(), Equals, "600")
	c.Check(feeSpentSWITCH.String(), Equals, "400")

	// Set values are distinct by Asset.
	feeWithheldSWITCH, err = k.GetOutboundFeeWithheldSwitch(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	feeSpentSWITCH, err = k.GetOutboundFeeSpentSwitch(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	c.Check(feeWithheldSWITCH.String(), Equals, "0")
	c.Check(feeSpentSWITCH.String(), Equals, "0")

	err = k.AddToOutboundFeeWithheldSwitch(ctx, common.ETHAsset, cosmos.NewUint(uint64(50)))
	c.Assert(err, IsNil)
	err = k.AddToOutboundFeeSpentSwitch(ctx, common.BTCAsset, cosmos.NewUint(uint64(30)))
	c.Assert(err, IsNil)

	feeWithheldSWITCH, err = k.GetOutboundFeeWithheldSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	feeSpentSWITCH, err = k.GetOutboundFeeSpentSwitch(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	c.Check(feeWithheldSWITCH.String(), Equals, "600")
	c.Check(feeSpentSWITCH.String(), Equals, "430")

	feeWithheldSWITCH, err = k.GetOutboundFeeWithheldSwitch(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	feeSpentSWITCH, err = k.GetOutboundFeeSpentSwitch(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	c.Check(feeWithheldSWITCH.String(), Equals, "50")
	c.Check(feeSpentSWITCH.String(), Equals, "0")
}
