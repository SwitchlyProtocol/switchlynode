package types

import (
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	. "gopkg.in/check.v1"
)

type MsgManageTHORNameSuite struct{}

var _ = Suite(&MsgManageTHORNameSuite{})

func (MsgManageTHORNameSuite) TestMsgManageTHORNameSuite(c *C) {
	owner := GetRandomBech32Addr()
	signer := GetRandomBech32Addr()
	coin := common.NewCoin(common.SwitchNative, cosmos.NewUint(10*common.One))
	msg := NewMsgManageTHORName("myname", common.ETHChain, GetRandomETHAddress(), coin, 0, common.ETHAsset, owner, signer)
	c.Assert(msg.ValidateBasic(), IsNil)
	c.Assert(msg.GetSigners(), NotNil)
	c.Assert(msg.GetSigners()[0].String(), Equals, signer.String())

	// unhappy paths
	msg = NewMsgManageTHORName("myname", common.ETHChain, GetRandomETHAddress(), coin, 0, common.ETHAsset, owner, cosmos.AccAddress{})
	c.Assert(msg.ValidateBasic(), NotNil)
	msg = NewMsgManageTHORName("myname", common.EmptyChain, GetRandomETHAddress(), coin, 0, common.ETHAsset, owner, signer)
	c.Assert(msg.ValidateBasic(), NotNil)
	msg = NewMsgManageTHORName("myname", common.ETHChain, common.NoAddress, coin, 0, common.ETHAsset, owner, signer)
	c.Assert(msg.ValidateBasic(), NotNil)
	msg = NewMsgManageTHORName("myname", common.ETHChain, GetRandomBTCAddress(), coin, 0, common.ETHAsset, owner, signer)
	c.Assert(msg.ValidateBasic(), NotNil)
	msg = NewMsgManageTHORName("myname", common.ETHChain, GetRandomETHAddress(), common.NewCoin(common.ETHAsset, cosmos.NewUint(10*common.One)), 0, common.ETHAsset, owner, signer)
	c.Assert(msg.ValidateBasic(), NotNil)
}

func (s *MsgManageTHORNameSuite) TestMsgManageTHORName(c *C) {
	acc1 := GetRandomBech32Addr()
	coin := common.NewCoin(common.SwitchNative, cosmos.NewUint(10*common.One))
	m := NewMsgManageTHORName("hello", common.BSCChain, GetRandomBNBAddress(), coin, 100, common.SwitchNative, acc1, acc1)
	EnsureMsgBasicCorrect(m, c)
	c.Check(m.ValidateBasic(), IsNil)
}
