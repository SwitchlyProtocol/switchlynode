package switchly

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

type HandlerManageSWITCHNameSuite struct{}

var _ = Suite(&HandlerManageSWITCHNameSuite{})

func (s *HandlerManageSWITCHNameSuite) TestValidator(c *C) {
	ctx, mgr := setupManagerForTest(c)

	handler := NewManageSWITCHNameHandler(mgr)
	coin := common.NewCoin(common.SwitchNative, cosmos.NewUint(100*common.One))
	addr := GetRandomSWITCHLYAddress()
	acc, _ := addr.AccAddress()
	name := NewSWITCHName("hello", 50, []SWITCHNameAlias{{Chain: common.SWITCHLYChain, Address: addr}})
	mgr.Keeper().SetSWITCHName(ctx, name)

	// set pool for preferred asset
	pool, err := mgr.Keeper().GetPool(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	pool.Asset = common.ETHAsset
	err = mgr.Keeper().SetPool(ctx, pool)
	c.Assert(err, IsNil)

	// happy path
	msg := NewMsgManageSWITCHName("I-am_the_99th_walrus+", common.SWITCHLYChain, addr, coin, 0, common.ETHAsset, acc, acc)
	c.Assert(handler.validate(ctx, *msg), IsNil)

	// fail: address is wrong chain
	msg.Chain = common.ETHChain
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// fail: address is wrong network
	mainnetBTCAddr, err := common.NewAddress("bc1qy0tj9fh0u6fgz0mejjp6776z6kugych0zwrkwr")
	c.Assert(err, IsNil)
	msg.Address = mainnetBTCAddr
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// restore to happy path
	msg.Chain = common.SWITCHLYChain
	msg.Address = addr

	// fail: name is too long
	msg.Name = "this_name_is_way_too_long_to_be_a_valid_name"
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// fail: bad characters
	msg.Name = "i am the walrus"
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// fail: bad attempt to inflate expire block height
	msg.Name = "hello"
	msg.ExpireBlockHeight = 100
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// fail: bad auth
	msg.ExpireBlockHeight = 0
	msg.Signer = GetRandomBech32Addr()
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// fail: not enough funds for new SWITCHName
	msg.Name = "bang"
	msg.Coin.Amount = cosmos.ZeroUint()
	c.Assert(handler.validate(ctx, *msg), NotNil)
}

func (s *HandlerManageSWITCHNameSuite) TestHandler(c *C) {
	ver := GetCurrentVersion()
	constAccessor := constants.GetConstantValues(ver)
	feePerBlock := constAccessor.GetInt64Value(constants.TNSFeePerBlock)
	registrationFee := constAccessor.GetInt64Value(constants.TNSRegisterFee)
	ctx, mgr := setupManagerForTest(c)

	blocksPerYear := mgr.GetConstants().GetInt64Value(constants.BlocksPerYear)
	handler := NewManageSWITCHNameHandler(mgr)
	coin := common.NewCoin(common.SwitchNative, cosmos.NewUint(100*common.One))
	addr := GetRandomSWITCHLYAddress()
	acc, _ := addr.AccAddress()
	tnName := "hello"

	// add switch to addr for gas
	FundAccount(c, ctx, mgr.Keeper(), acc, 10*common.One)

	// happy path, register new name
	msg := NewMsgManageSWITCHName(tnName, common.SWITCHLYChain, addr, coin, 0, common.SwitchNative, acc, acc)
	_, err := handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	name, err := mgr.Keeper().GetSWITCHName(ctx, tnName)
	c.Assert(err, IsNil)
	c.Check(name.Owner.Empty(), Equals, false)
	c.Check(name.ExpireBlockHeight, Equals, ctx.BlockHeight()+blocksPerYear+(int64(coin.Amount.Uint64())-registrationFee)/feePerBlock)

	// happy path, set alt chain address
	ethAddr := GetRandomETHAddress()
	msg = NewMsgManageSWITCHName(tnName, common.ETHChain, ethAddr, coin, 0, common.SwitchNative, acc, acc)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	name, err = mgr.Keeper().GetSWITCHName(ctx, tnName)
	c.Assert(err, IsNil)
	c.Check(name.GetAlias(common.ETHChain).Equals(ethAddr), Equals, true)

	// happy path, update alt chain address
	ethAddr = GetRandomETHAddress()
	msg = NewMsgManageSWITCHName(tnName, common.ETHChain, ethAddr, coin, 0, common.SwitchNative, acc, acc)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	name, err = mgr.Keeper().GetSWITCHName(ctx, tnName)
	c.Assert(err, IsNil)
	c.Check(name.GetAlias(common.ETHChain).Equals(ethAddr), Equals, true)

	// update preferred asset
	msg = NewMsgManageSWITCHName(tnName, common.ETHChain, ethAddr, coin, 0, common.ETHAsset, acc, acc)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	name, err = mgr.Keeper().GetSWITCHName(ctx, tnName)
	c.Assert(err, IsNil)
	c.Check(name.PreferredAsset, Equals, common.ETHAsset)

	// transfer switchlyname to new owner, should reset preferred asset/external aliases
	addr2 := GetRandomSWITCHLYAddress()
	acc2, _ := addr2.AccAddress()
	msg = NewMsgManageSWITCHName(tnName, common.SWITCHLYChain, addr, coin, 0, common.SwitchNative, acc2, acc)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	name, err = mgr.Keeper().GetSWITCHName(ctx, tnName)
	c.Assert(err, IsNil)
	c.Check(len(name.GetAliases()), Equals, 0)
	c.Check(name.PreferredAsset.IsEmpty(), Equals, true)
	c.Check(name.GetOwner().Equals(acc2), Equals, true)

	// happy path, release switchlyname back into the wild
	msg = NewMsgManageSWITCHName(tnName, common.SWITCHLYChain, addr, common.NewCoin(common.SwitchNative, cosmos.ZeroUint()), 1, common.SwitchNative, acc, acc)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	name, err = mgr.Keeper().GetSWITCHName(ctx, tnName)
	c.Assert(err, IsNil)
	c.Check(name.Owner.Empty(), Equals, true)
	c.Check(name.ExpireBlockHeight, Equals, int64(0))
}
