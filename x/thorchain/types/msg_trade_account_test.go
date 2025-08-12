package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	. "gopkg.in/check.v1"
)

type MsgTradeAccountSuite struct{}

var _ = Suite(&MsgTradeAccountSuite{})

func (MsgTradeAccountSuite) TestDeposit(c *C) {
	asset := common.ETHAsset
	amt := cosmos.NewUint(100)
	signer := GetRandomBech32Addr()
	dummyTx := common.Tx{ID: "test"}

	m := NewMsgTradeAccountDeposit(asset, amt, signer, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m = NewMsgTradeAccountDeposit(common.EmptyAsset, amt, signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgTradeAccountDeposit(common.SwitchNative, amt, signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgTradeAccountDeposit(asset, cosmos.ZeroUint(), signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)
}

func (MsgTradeAccountSuite) TestWithdrawal(c *C) {
	asset := common.ETHAsset.GetTradeAsset()
	amt := cosmos.NewUint(100)
	ethAddr := GetRandomETHAddress()
	signer := GetRandomBech32Addr()
	dummyTx := common.Tx{ID: "test"}

	m := NewMsgTradeAccountWithdrawal(asset, amt, ethAddr, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m = NewMsgTradeAccountWithdrawal(common.EmptyAsset, amt, ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgTradeAccountWithdrawal(common.SwitchNative, amt, ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgTradeAccountWithdrawal(asset, cosmos.ZeroUint(), ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgTradeAccountWithdrawal(asset, cosmos.ZeroUint(), GetRandomTHORAddress(), signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)
}

func (s *MsgTradeAccountSuite) TestMsgTradeAccountDeposit(c *C) {
	signer := GetRandomBech32Addr()
	dummyTx := GetRandomTx()
	amt := cosmos.NewUint(100 * common.One)
	m := NewMsgTradeAccountDeposit(common.ETHAsset, amt, signer, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m1 := NewMsgTradeAccountDeposit(common.ETHAsset, amt, signer, signer, dummyTx)
	c.Check(m.ValidateBasic(), IsNil)
	c.Check(m1.ValidateBasic(), IsNil)

	// ensure we can set the signer
	m.Signer = GetRandomBech32Addr()
	c.Check(m.Signer.Equals(m1.Signer), Equals, false)
}

func (s *MsgTradeAccountSuite) TestMsgTradeAccountWithdraw(c *C) {
	signer := GetRandomBech32Addr()
	ethAddr := GetRandomETHAddress()
	dummyTx := GetRandomTx()
	amt := cosmos.NewUint(100 * common.One)
	tradeAsset := common.ETHAsset.GetTradeAsset()
	m := NewMsgTradeAccountWithdrawal(tradeAsset, amt, ethAddr, signer, dummyTx)
	EnsureMsgBasicCorrect(m, c)

	m = NewMsgTradeAccountWithdrawal(tradeAsset, amt, ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), IsNil)

	m = NewMsgTradeAccountWithdrawal(tradeAsset, cosmos.ZeroUint(), ethAddr, signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)

	m = NewMsgTradeAccountWithdrawal(tradeAsset, cosmos.ZeroUint(), GetRandomTHORAddress(), signer, dummyTx)
	c.Check(m.ValidateBasic(), NotNil)
}
