package switchly

import (
	"cosmossdk.io/math"
	"github.com/switchlyprotocol/switchlynode/v3/common"
	cosmos "github.com/switchlyprotocol/switchlynode/v3/common/cosmos"

	. "gopkg.in/check.v1"
)

type HandlerSWCYUnstake struct{}

var _ = Suite(&HandlerSWCYUnstake{})

func (s *HandlerSWCYUnstake) TestValidate(c *C) {
	ctx, k := setupKeeperForTest(c)

	// happy path
	k.SetMimir(ctx, "SWCYUnstakingHalt", 0)
	fromAddr := GetRandomSwitchAddress()
	toAddr := GetRandomSwitchAddress()
	accSignerAddr, err := fromAddr.AccAddress()
	c.Assert(err, IsNil)
	bps := cosmos.NewUint(100_00)

	tx := common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.Coins{},
		common.Gas{},
		"",
	)

	msg := NewMsgSWCYUnstake(tx, bps, accSignerAddr)
	handler := NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, IsNil)

	// invalid msgs
	// empty signer
	msg = NewMsgSWCYUnstake(tx, bps, cosmos.AccAddress{})
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// empty bps
	msg = NewMsgSWCYUnstake(tx, cosmos.ZeroUint(), cosmos.AccAddress{})
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// bps more than 100%
	msg = NewMsgSWCYUnstake(tx, cosmos.NewUint(200_00), cosmos.AccAddress{})
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// tx from address not switch address
	tx = common.NewTx(
		common.BlankTxID,
		GetRandomBTCAddress(),
		toAddr,
		common.Coins{},
		common.Gas{},
		"",
	)
	msg = NewMsgSWCYUnstake(tx, bps, accSignerAddr)
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)

	// happy path
	k.SetMimir(ctx, "SWCYUnstakingHalt", 1)
	bps = cosmos.NewUint(100_00)

	tx = common.NewTx(
		common.BlankTxID,
		fromAddr,
		toAddr,
		common.Coins{},
		common.Gas{},
		"",
	)

	msg = NewMsgSWCYUnstake(tx, bps, accSignerAddr)
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	err = handler.validate(ctx, *msg)
	c.Assert(err.Error(), Equals, "tcy unstaking is halt")

	// empty msg
	k.SetMimir(ctx, "SWCYUnstakingHalt", 0)
	msg = &MsgSWCYUnstake{}
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)
}

func (s *HandlerSWCYStake) TestHandle(c *C) {
	ctx, k := setupKeeperForTest(c)

	addr1 := GetRandomSwitchAddress()
	accAddr1, err := addr1.AccAddress()
	c.Assert(err, IsNil)
	addr1Amount := cosmos.NewUint(100)
	addr2 := GetRandomSwitchAddress()
	accAddr2, err := addr2.AccAddress()
	c.Assert(err, IsNil)
	addr2Amount := cosmos.NewUint(200)

	tx := common.NewTx(
		common.BlankTxID,
		common.NoAddress,
		GetRandomSwitchAddress(),
		common.NewCoins(common.NewCoin(common.SwitchNative, cosmos.NewUint(1))),
		common.Gas{},
		"",
	)

	// set stakers and staking module
	err = k.SetSWCYStaker(ctx, SWCYStaker{
		Address: addr1,
		Amount:  addr1Amount,
	})
	c.Assert(err, IsNil)
	err = k.SetSWCYStaker(ctx, SWCYStaker{
		Address: addr2,
		Amount:  addr2Amount,
	})
	c.Assert(err, IsNil)

	stakeAmount := addr1Amount.Add(addr2Amount)
	coin := common.NewCoin(common.SWCY, stakeAmount)
	err = k.MintToModule(ctx, ModuleName, coin)
	c.Assert(err, IsNil)
	err = k.SendFromModuleToModule(ctx, ModuleName, SWCYStakeName, common.NewCoins(coin))
	c.Assert(err, IsNil)
	tcyStakeAmount := k.GetBalanceOfModule(ctx, SWCYStakeName, common.SWCY.Native())
	c.Assert(tcyStakeAmount.Equal(stakeAmount), Equals, true)

	// check state before
	c.Assert(k.SWCYStakerExists(ctx, addr1), Equals, true)
	c.Assert(k.SWCYStakerExists(ctx, addr2), Equals, true)

	addr1Coin := k.GetBalanceOf(ctx, accAddr1, common.SWCY)
	c.Assert(addr1Coin.IsZero(), Equals, true)
	addr2Coin := k.GetBalanceOf(ctx, accAddr2, common.SWCY)
	c.Assert(addr2Coin.IsZero(), Equals, true)

	// unstake 100% from addr1
	tx.FromAddress = addr1
	bps := cosmos.NewUint(100_00)
	msg := NewMsgSWCYUnstake(tx, bps, accAddr1)
	handler := NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)

	c.Assert(k.SWCYStakerExists(ctx, addr1), Equals, false)
	c.Assert(k.SWCYStakerExists(ctx, addr2), Equals, true)

	addr1Coin = k.GetBalanceOf(ctx, accAddr1, common.SWCY)
	c.Assert(addr1Coin.Amount.Equal(math.NewInt(100)), Equals, true)
	addr2Coin = k.GetBalanceOf(ctx, accAddr2, common.SWCY)
	c.Assert(addr2Coin.IsZero(), Equals, true)
	tcyStakeAmount = k.GetBalanceOfModule(ctx, SWCYStakeName, common.SWCY.Native())
	c.Assert(tcyStakeAmount.Equal(addr2Amount), Equals, true)

	// unstake 25% from addr2
	tx.FromAddress = addr2
	bps = cosmos.NewUint(25_00)
	msg = NewMsgSWCYUnstake(tx, bps, accAddr2)
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)

	c.Assert(k.SWCYStakerExists(ctx, addr1), Equals, false)
	c.Assert(k.SWCYStakerExists(ctx, addr2), Equals, true)

	staker, err := k.GetSWCYStaker(ctx, addr2)
	c.Assert(err, IsNil)
	c.Assert(staker.Address.Equals(addr2), Equals, true)
	c.Assert(staker.Amount.Equal(math.NewUint(150)), Equals, true)

	addr1Coin = k.GetBalanceOf(ctx, accAddr1, common.SWCY)
	c.Assert(addr1Coin.Amount.Equal(math.NewInt(100)), Equals, true)
	addr2Coin = k.GetBalanceOf(ctx, accAddr2, common.SWCY)
	c.Assert(addr2Coin.Amount.Equal(math.NewInt(50)), Equals, true)
	tcyStakeAmount = k.GetBalanceOfModule(ctx, SWCYStakeName, common.SWCY.Native())
	c.Assert(tcyStakeAmount.Equal(math.NewUint(150)), Equals, true)

	// unstake 100% from addr2
	tx.FromAddress = addr2
	bps = cosmos.NewUint(100_00)
	msg = NewMsgSWCYUnstake(tx, bps, accAddr2)
	handler = NewSWCYUnstakeHandler(NewDummyMgrWithKeeper(k))
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)

	c.Assert(k.SWCYStakerExists(ctx, addr1), Equals, false)
	c.Assert(k.SWCYStakerExists(ctx, addr2), Equals, false)

	addr1Coin = k.GetBalanceOf(ctx, accAddr1, common.SWCY)
	c.Assert(addr1Coin.Amount.Equal(math.NewInt(100)), Equals, true)
	addr2Coin = k.GetBalanceOf(ctx, accAddr2, common.SWCY)
	c.Assert(addr2Coin.Amount.Equal(math.NewInt(200)), Equals, true)
	tcyStakeAmount = k.GetBalanceOfModule(ctx, SWCYStakeName, common.SWCY.Native())
	c.Assert(tcyStakeAmount.IsZero(), Equals, true)
}
