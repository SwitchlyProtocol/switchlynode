package types

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
)

type THORNameSuite struct{}

var _ = Suite(&THORNameSuite{})

func (THORNameSuite) TestTHORName(c *C) {
	// happy path
	n := NewTHORName("iamthewalrus", 0, []THORNameAlias{{Chain: common.SWITCHLYChain, Address: GetRandomTHORAddress()}})
	c.Check(n.Valid(), IsNil)

	// unhappy path
	n1 := NewTHORName("", 0, []THORNameAlias{{Chain: common.ETHChain, Address: GetRandomTHORAddress()}})
	c.Check(n1.Valid(), NotNil)
	n2 := NewTHORName("hello", 0, []THORNameAlias{{Chain: common.EmptyChain, Address: GetRandomTHORAddress()}})
	c.Check(n2.Valid(), NotNil)
	n3 := NewTHORName("hello", 0, []THORNameAlias{{Chain: common.SWITCHLYChain, Address: common.Address("")}})
	c.Check(n3.Valid(), NotNil)

	// set/get alias
	eth1 := GetRandomETHAddress()
	n1.SetAlias(common.ETHChain, eth1)
	c.Check(n1.GetAlias(common.ETHChain), Equals, eth1)
}
