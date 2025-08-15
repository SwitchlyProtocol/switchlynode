package keeperv1

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	. "gopkg.in/check.v1"
)

type KeeperSWITCHNameSuite struct{}

var _ = Suite(&KeeperSWITCHNameSuite{})

func (s *KeeperSWITCHNameSuite) TestSWITCHName(c *C) {
	ctx, k := setupKeeperForTest(c)
	var err error
	ref := "helloworld"

	ok := k.SWITCHNameExists(ctx, ref)
	c.Assert(ok, Equals, false)

	thorAddr := GetRandomSWITCHLYAddress()
	ethAddr := GetRandomETHAddress()
	name := NewSWITCHName(ref, 50, []SWITCHNameAlias{{Chain: common.SWITCHLYChain, Address: thorAddr}, {Chain: common.ETHChain, Address: ethAddr}})
	k.SetSWITCHName(ctx, name)

	ok = k.SWITCHNameExists(ctx, ref)
	c.Assert(ok, Equals, true)
	ok = k.SWITCHNameExists(ctx, "bogus")
	c.Assert(ok, Equals, false)

	name, err = k.GetSWITCHName(ctx, ref)
	c.Assert(err, IsNil)
	c.Assert(name.GetAlias(common.SWITCHLYChain).Equals(thorAddr), Equals, true)
	c.Assert(name.GetAlias(common.ETHChain).Equals(ethAddr), Equals, true)
}
