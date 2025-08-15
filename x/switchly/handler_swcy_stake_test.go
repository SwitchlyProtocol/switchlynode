package switchly

import (
	math "cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"

	. "gopkg.in/check.v1"
)

type HandlerSWCYStake struct{}

var _ = Suite(&HandlerSWCYStake{})

func (s *HandlerSWCYStake) TestValidate(c *C) {
	ctx, k := setupKeeperForTest(c)

	// happy path
	k.SetMimir(ctx, "SWCYStakingHalt", 0)
	fromAddr := GetRandomSwitchAddress()
	toAddr := GetRandomSwitchAddress()
	accSignerAddr, err := fromAddr.AccAddress()
	c.Assert(err, IsNil)
	coin := common.NewCoin(common.SWCY, cosmos.NewUint(100))

	tx := common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	msg := NewMsgSWCYStake(tx, accSignerAddr)
	handler := NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, IsNil)

	// invalid msgs
	// invalid coin
	coin = common.NewCoin(common.ETHAsset, cosmos.NewUint(100))
	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYStake(tx, accSignerAddr)
	handler = NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// multiple coins send
	coin = common.NewCoin(common.SWCY, cosmos.NewUint(100))
	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin, coin),
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYStake(tx, accSignerAddr)
	handler = NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// coin is not tcy
	coin = common.NewCoin(common.SwitchNative, cosmos.NewUint(100))
	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYStake(tx, accSignerAddr)
	handler = NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// coin amount is zero
	coin = common.NewCoin(common.SWCY, cosmos.ZeroUint())
	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYStake(tx, cosmos.AccAddress{})
	handler = NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// signer is empty
	coin = common.NewCoin(common.SWCY, cosmos.NewUint(100))
	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYStake(tx, cosmos.AccAddress{})
	handler = NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// staking halt
	k.SetMimir(ctx, "SWCYStakingHalt", 1)
	coin = common.NewCoin(common.SWCY, cosmos.NewUint(100))

	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYStake(tx, accSignerAddr)
	handler = NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err.Error(), Equals, "tcy staking is halted")

	// empty msg
	k.SetMimir(ctx, "SWCYStakingHalt", 0)
	msg = &MsgSWCYStake{}
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)
}

func (s *HandlerSWCYUnstake) TestHandle(c *C) {
	ctx, k := setupKeeperForTest(c)

	// first time stake
	fromAddr := GetRandomSwitchAddress()
	toAddr := GetRandomSwitchAddress()
	accSignerAddr, err := fromAddr.AccAddress()
	c.Assert(err, IsNil)
	amount := cosmos.NewUint(100)
	coin := common.NewCoin(common.SWCY, amount)

	tx := common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.NewCoins(coin),
		common.Gas{},
		"",
	)

	// check before state
	c.Assert(k.SWCYStakerExists(ctx, fromAddr), Equals, false)

	msg := NewMsgSWCYStake(tx, accSignerAddr)
	handler := NewSWCYStakeHandler(NewDummyMgrWithKeeper(k))
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)

	// check after state
	c.Assert(k.SWCYStakerExists(ctx, fromAddr), Equals, true)

	staker, err := k.GetSWCYStaker(ctx, fromAddr)
	c.Assert(err, IsNil)
	c.Assert(staker.Address.Equals(fromAddr), Equals, true)
	c.Assert(staker.Amount.Equal(amount), Equals, true)

	// second time stake
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)

	// the amount should be twice since the msg was send twice
	staker, err = k.GetSWCYStaker(ctx, fromAddr)
	c.Assert(err, IsNil)
	c.Assert(staker.Address.Equals(fromAddr), Equals, true)
	c.Assert(staker.Amount.Equal(amount.Mul(math.NewUint(2))), Equals, true)
}
