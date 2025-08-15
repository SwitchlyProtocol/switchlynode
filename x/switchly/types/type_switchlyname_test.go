package types

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
)

type SWITCHNameSuite struct{}

var _ = Suite(&SWITCHNameSuite{})

func (SWITCHNameSuite) TestSWITCHName(c *C) {
	// happy path
	n := NewSWITCHName("iamthewalrus", 0, []SWITCHNameAlias{{Chain: common.SWITCHLYChain, Address: GetRandomSWITCHLYAddress()}})
	c.Check(n.Valid(), IsNil)

	// unhappy path
	n1 := NewSWITCHName("", 0, []SWITCHNameAlias{{Chain: common.ETHChain, Address: GetRandomSWITCHLYAddress()}})
	c.Check(n1.Valid(), NotNil)
	n2 := NewSWITCHName("hello", 0, []SWITCHNameAlias{{Chain: common.EmptyChain, Address: GetRandomSWITCHLYAddress()}})
	c.Check(n2.Valid(), NotNil)
	n3 := NewSWITCHName("hello", 0, []SWITCHNameAlias{{Chain: common.SWITCHLYChain, Address: common.Address("")}})
	c.Check(n3.Valid(), NotNil)

	// set/get alias
	eth1 := GetRandomETHAddress()
	n1.SetAlias(common.ETHChain, eth1)
	c.Check(n1.GetAlias(common.ETHChain), Equals, eth1)
}
