package types

import (
	"errors"

	se "github.com/cosmos/cosmos-sdk/types/errors"
	. "gopkg.in/check.v1"

	common "github.com/switchlyprotocol/switchlynode/v1/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

type MsgDepositSuite struct{}

var _ = Suite(&MsgDepositSuite{})

func (MsgDepositSuite) TestMsgDepositSuite(c *C) {
	acc1 := GetRandomBech32Addr()
	c.Assert(acc1.Empty(), Equals, false)

	coins := common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(12*common.One)),
	}
	memo := "hello"
	msg := NewMsgDeposit(coins, memo, acc1)
	c.Assert(msg.ValidateBasic(), IsNil)
	c.Assert(msg.GetSigners(), NotNil)
	c.Assert(msg.GetSigners()[0].String(), Equals, acc1.String())

	// ensure non-native assets are blocked
	coins = common.Coins{
		common.NewCoin(common.BTCAsset, cosmos.NewUint(12*common.One)),
	}
	msg = NewMsgDeposit(coins, memo, acc1)
	c.Assert(msg.ValidateBasic(), NotNil)

	msg1 := NewMsgDeposit(coins, "memo", cosmos.AccAddress{})
	err1 := msg1.ValidateBasic()
	c.Assert(err1, NotNil)
	c.Assert(errors.Is(err1, se.ErrInvalidAddress), Equals, true)

	msg2 := NewMsgDeposit(common.Coins{
		common.NewCoin(common.EmptyAsset, cosmos.ZeroUint()),
	}, "memo", acc1)
	err2 := msg2.ValidateBasic()
	c.Assert(err2, NotNil)
	c.Assert(errors.Is(err2, se.ErrUnknownRequest), Equals, true)

	msg3 := NewMsgDeposit(common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(12*common.One)),
	}, "asdfsdkljadslfasfaqcvbncvncvbncvbncvbncvbncvbncvbncvbncvbncvbnsdfasdfasfasdfkjqwerqlkwerqlerqwlkerjqlwkerjqwlkerjqwlkerjqlkwerjklqwerjqwlkerjqlwkerjwqelrasdfsdkljadslfasfaqcvbncvncvbncvbncvbncvbncvbncvbncvbncvbncvbnsdfasdfasfasdfkjqwerqlkwerqlerqwlkerjqlwkerjqwlkerjqwlkerjqlkwerjklqwerjqwlkerjqlwkerjwqelr", acc1)
	err3 := msg3.ValidateBasic()
	c.Assert(err3, NotNil)
	c.Assert(errors.Is(err3, se.ErrUnknownRequest), Equals, true)
}

func (s *MsgDepositSuite) TestMsgDeposit(c *C) {
	acc1 := GetRandomBech32Addr()
	coins := common.NewCoins(
		common.NewCoin(common.SwitchNative, cosmos.NewUint(12*common.One)),
	)
	memo := "hello"

	m := NewMsgDeposit(coins, memo, acc1)
	EnsureMsgBasicCorrect(m, c)

	m1 := NewMsgDeposit(coins, memo, acc1)
	c.Check(m.Coins.EqualsEx(m1.Coins), Equals, true)
	c.Check(m.Memo, Equals, m1.Memo)
	c.Check(m.Signer.Equals(m1.Signer), Equals, true)

	// ensure we can set the signer
	m.Signer = GetRandomBech32Addr()
	c.Check(m.Signer.Equals(m1.Signer), Equals, false)
}

func (s *MsgDepositSuite) TestMsgDepositValidation(c *C) {
	acc1 := GetRandomBech32Addr()
	coins := common.NewCoins(
		common.NewCoin(common.SwitchNative, cosmos.NewUint(12*common.One)),
	)
	memo := "hello"

	m := NewMsgDeposit(coins, memo, acc1)
	c.Check(m.ValidateBasic(), IsNil)

	// test with empty coins
	m = NewMsgDeposit(common.NewCoins(), memo, acc1)
	c.Check(m.ValidateBasic(), NotNil)

	// test with empty signer
	m = NewMsgDeposit(coins, memo, cosmos.AccAddress{})
	c.Check(m.ValidateBasic(), NotNil)
}
