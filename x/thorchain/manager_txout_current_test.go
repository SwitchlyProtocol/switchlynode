package thorchain

import (
	"gopkg.in/check.v1"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

type TxOutStoreVCURSuite struct{}

var _ = Suite(&TxOutStoreVCURSuite{})

func (s TxOutStoreVCURSuite) TestAddGasFees(c *C) {
	ctx, mgr := setupManagerForTest(c)
	tx := GetRandomObservedTx()

	// Set vault to satisfy VaultExists check.
	vault := NewVault(ctx.BlockHeight(), ActiveVault, AsgardVault, tx.ObservedPubKey, common.Chains{common.ETHChain}.Strings(), []ChainContract{})
	c.Assert(mgr.Keeper().SetVault(ctx, vault), IsNil)

	err := addGasFees(ctx, mgr, tx)
	c.Assert(err, IsNil)
	c.Assert(mgr.GasMgr().GetGas(), HasLen, 1)
}

func (s TxOutStoreVCURSuite) TestEndBlock(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())

	item := TxOutItem{
		Chain:     common.ETHChain,
		ToAddress: GetRandomETHAddress(),
		InHash:    GetRandomTxHash(),
		Coin:      common.NewCoin(common.ETHAsset, cosmos.NewUint(20*common.One)),
	}
	err := txOutStore.UnSafeAddTxOutItem(w.ctx, w.mgr, item, w.ctx.BlockHeight())
	c.Assert(err, IsNil)

	c.Assert(txOutStore.EndBlock(w.ctx, w.mgr), IsNil)

	items, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
	c.Check(items[0].GasRate, Equals, int64(56250))
	c.Assert(items[0].MaxGas, HasLen, 1)
	c.Check(items[0].MaxGas[0].Asset.Equals(common.ETHAsset), Equals, true)
	c.Check(items[0].MaxGas[0].Amount.Uint64(), Equals, uint64(56250))
}

func (s TxOutStoreVCURSuite) TestAddOutTxItem(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.BCHAsset, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	outboundFeeWithheldRune, err := w.keeper.GetOutboundFeeWithheldRune(w.ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	outboundFeeSpentRune, err := w.keeper.GetOutboundFeeSpentRune(w.ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	c.Check(outboundFeeWithheldRune.String(), Equals, "0")
	c.Check(outboundFeeSpentRune.String(), Equals, "0")

	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// Should get acc2. Acc3 hasn't signed and acc2 is the highest value
	item := TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(20*common.One)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Assert(msgs[0].VaultPubKey.String(), Equals, vault.PubKey.String())
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(1999925000)), Equals, true, Commentf("%d", msgs[0].Coin.Amount.Uint64()))

	// Gas withheld should be updated
	outboundFeeWithheldRune, err = w.keeper.GetOutboundFeeWithheldRune(w.ctx, common.DOGEAsset)
	c.Assert(err, IsNil)
	outboundFeeSpentRune, err = w.keeper.GetOutboundFeeSpentRune(w.ctx, common.DOGEAsset)
	c.Assert(err, IsNil)
	c.Check(outboundFeeWithheldRune.String(), Equals, "74999") // After slippage the 75000 fee is 74999 in RUNE
	c.Check(outboundFeeSpentRune.String(), Equals, "0")

	// Should get acc1. Acc3 hasn't signed and acc1 now has the highest amount
	// of coin.
	item = TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(20*common.One)),
	}
	txOutStore.ClearOutboundItems(w.ctx)
	success, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(success, Equals, true)
	c.Assert(err, IsNil)
	msgs, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Assert(msgs[0].VaultPubKey.String(), Equals, vault.PubKey.String())

	// Outbound fee withheld RUNE should be updated
	outboundFeeWithheldRune, err = w.keeper.GetOutboundFeeWithheldRune(w.ctx, common.DOGEAsset)
	c.Assert(err, IsNil)
	c.Assert(outboundFeeWithheldRune.String(), Equals, "149997")

	item = TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(1000*common.One)),
	}
	txOutStore.ClearOutboundItems(w.ctx)
	success, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(success, Equals, true)
	msgs, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Check(msgs[0].VaultPubKey.String(), Equals, vault.PubKey.String())

	// Outbound fee withheld RUNE should be updated
	outboundFeeWithheldRune, err = w.keeper.GetOutboundFeeWithheldRune(w.ctx, common.DOGEAsset)
	c.Assert(err, IsNil)
	c.Assert(outboundFeeWithheldRune.String(), Equals, "224994")

	networkFee := NewNetworkFee(common.BCHChain, 1, 10)
	c.Assert(w.keeper.SaveNetworkFee(w.ctx, common.BCHChain, networkFee), IsNil)

	item = TxOutItem{
		Chain:     common.BCHChain,
		ToAddress: "1EFJFJm7Y9mTVsCBXA9PKuRuzjgrdBe4rR",
		InHash:    inTxID,
		Coin:      common.NewCoin(common.BCHAsset, cosmos.NewUint(20*common.One)),
		MaxGas: common.Gas{
			common.NewCoin(common.BCHAsset, cosmos.NewUint(10000)),
		},
	}
	txOutStore.ClearOutboundItems(w.ctx)
	result, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(result, Equals, true)
	msgs, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	// this should be a mocknet address
	c.Assert(msgs[0].ToAddress.String(), Equals, "qzg5mkh7rkw3y8kw47l3rrnvhmenvctmd5yg6hxe64")

	// outbound originating from a pool should pay fee from asgard to reserve
	FundModule(c, w.ctx, w.keeper, AsgardName, 1000*common.One)
	testAndCheckModuleBalances(c, w.ctx, w.keeper,
		func() {
			item = TxOutItem{
				Chain:     common.SWITCHLYChain,
				ToAddress: GetRandomRUNEAddress(),
				InHash:    inTxID,
				Coin:      common.NewCoin(common.SwitchNative, cosmos.NewUint(1000*common.One)),
			}
			txOutStore.ClearOutboundItems(w.ctx)
			success, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
			c.Assert(err, IsNil)
			c.Assert(success, Equals, true)
			msgs, err = txOutStore.GetOutboundItems(w.ctx)
			c.Assert(err, IsNil)
			c.Assert(msgs, HasLen, 0)
		},
		ModuleBalances{
			Asgard:  -1000_00000000,
			Reserve: 2000000,
		},
	)

	// outbound originating from bond should pay fee from bond to reserve
	FundModule(c, w.ctx, w.keeper, BondName, 1000*common.One)
	testAndCheckModuleBalances(c, w.ctx, w.keeper,
		func() {
			item = TxOutItem{
				ModuleName: BondName,
				Chain:      common.SWITCHLYChain,
				ToAddress:  GetRandomRUNEAddress(),
				InHash:     inTxID,
				Coin:       common.NewCoin(common.SwitchNative, cosmos.NewUint(1000*common.One)),
			}
			txOutStore.ClearOutboundItems(w.ctx)
			success, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
			c.Assert(err, IsNil)
			c.Assert(success, Equals, true)
			msgs, err = txOutStore.GetOutboundItems(w.ctx)
			c.Assert(err, IsNil)
			c.Assert(msgs, HasLen, 0)
		},
		ModuleBalances{
			Bond:    -1000_00000000,
			Reserve: 2000000,
		},
	)

	// ensure that min out is respected
	success, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.NewUint(9999999999*common.One))
	c.Check(success, Equals, false)
	c.Check(err, NotNil)
}

func (s TxOutStoreVCURSuite) TestAddOutTxItem_OutboundHeightDoesNotGetOverride(c *C) {
	SetupConfigForTest()
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	w.keeper.SetMimir(w.ctx, constants.MinTxOutVolumeThreshold.String(), 100000000000)
	w.keeper.SetMimir(w.ctx, constants.TxOutDelayRate.String(), 2500000000)
	w.keeper.SetMimir(w.ctx, constants.MaxTxOutOffset.String(), 720)
	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// this should be sent via asgard
	item := TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(80*common.One)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 0)
	//  the outbound has been delayed
	newCtx := w.ctx.WithBlockHeight(4)
	msgs, err = txOutStore.GetOutboundItems(newCtx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Assert(msgs[0].VaultPubKey.String(), Equals, vault.PubKey.String())
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(7999925000)), Equals, true, Commentf("%d", msgs[0].Coin.Amount.Uint64()))

	// make sure outbound_height has been set correctly
	afterVoter, err := w.keeper.GetObservedTxInVoter(w.ctx, inTxID)
	c.Assert(err, IsNil)
	c.Assert(afterVoter.OutboundHeight, Equals, int64(4))

	item.Chain = common.SWITCHLYChain
	item.Coin = common.NewCoin(common.SwitchNative, cosmos.NewUint(100*common.One))
	item.ToAddress = GetRandomTHORAddress()
	ok, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// make sure outbound_height has not been overwritten
	afterVoter1, err := w.keeper.GetObservedTxInVoter(w.ctx, inTxID)
	c.Assert(err, IsNil)
	c.Assert(afterVoter1.OutboundHeight, Equals, int64(4))
}

func (s TxOutStoreVCURSuite) TestAddOutTxItemNotEnoughForFee(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	item := TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(300000)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, NotNil)
	c.Assert(err, Equals, ErrNotEnoughToPayFee)
	c.Assert(ok, Equals, false)
	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 0)
}

func (s TxOutStoreVCURSuite) TestAddOutTxItemWithoutBFT(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	inTxID := GetRandomTxHash()
	item := TxOutItem{
		Chain:     common.ETHChain,
		ToAddress: GetRandomETHAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.ETHAsset, cosmos.NewUint(20*common.One)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	success, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(success, Equals, true)
	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	// TODO: Eri: seems like before gas was subtracted, not sure why it would be
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(20*common.One)), Equals, true, Commentf("%d", msgs[0].Coin.Amount.Uint64()))
}

func (s TxOutStoreVCURSuite) TestCalcTxOutHeight(c *C) {
	ctx, keeper := setupKeeperForTest(c)

	pool := NewPool()
	pool.Asset = common.ETHAsset
	pool.BalanceRune = cosmos.NewUint(90527581399649)
	pool.BalanceAsset = cosmos.NewUint(1402011488988)
	c.Assert(keeper.SetPool(ctx, pool), IsNil)

	keeper.SetMimir(ctx, "MinTxOutVolumeThreshold", 2500000000)
	keeper.SetMimir(ctx, "TxOutDelayRate", 2500000000)
	keeper.SetMimir(ctx, "MaxTxOutOffset", 720)
	keeper.SetMimir(ctx, "TxOutDelayMax", 17280)
	// With the above values, a RUNE value of 18,000 would be delayed for the full MaxTxOutOffset.

	version := GetCurrentVersion()
	txout, err := GetTxOutStore(version, keeper, NewDummyEventMgr(), NewDummyGasManager())
	c.Assert(err, IsNil)

	toi := TxOutItem{
		Coin: common.NewCoin(common.ETHAsset, cosmos.NewUint(2*common.One)),
		Memo: "OUT:nomnomnom",
	}
	value := pool.AssetValueInRune(toi.Coin.Amount)
	c.Check(value.Uint64(), Equals, uint64(129_13957141), Commentf("%d", value.Uint64()))

	c.Check(ctx.BlockHeight(), Equals, int64(18), Commentf("%d", ctx.BlockHeight()))
	// Confirming that the current height is 18.

	targetBlock, _, err := txout.CalcTxOutHeight(ctx, keeper.GetVersion(), toi)
	c.Assert(err, IsNil)
	c.Check(targetBlock, Equals, int64(24))
	c.Assert(keeper.AppendTxOut(ctx, targetBlock, toi), IsNil)
	// (sumValue / minTxOutVolumeThreshold) * common.One = (129_13957141 / 25_00000000) * 1_00000000 = 5_00000000,
	// which reduces the 25_00000000 TxOutDelayRate to 20_00000000.
	// value / TxOutDelayRate is then 129 / 20 ~= 6, added to the starting height of 18 to get 24.

	targetBlock, _, err = txout.CalcTxOutHeight(ctx, keeper.GetVersion(), toi)
	c.Assert(err, IsNil)
	c.Check(targetBlock, Equals, int64(26))
	c.Assert(keeper.AppendTxOut(ctx, targetBlock, toi), IsNil)
	// (sumValue / minTxOutVolumeThreshold) * common.One = (258_27914282 / 25_00000000) * 1_00000000 = 10_00000000,
	// which reduces the 25_00000000 TxOutDelayRate to 15_00000000.
	// value / TxOutDelayRate is then 129 / 15 ~= 8, added to the starting height of 18 to get 26.

	normalGasTOI := toi
	normalGasTOI.MaxGas = common.Gas{toi.Coin}
	normalGasTOI.MaxGas[0].Amount = cosmos.NewUint(7500)
	normalGasValue := pool.AssetValueInRune(normalGasTOI.MaxGas[0].Amount)
	c.Check(normalGasValue.String(), Equals, "484273", Commentf("%s", normalGasValue.String()))
	targetBlock, _, err = txout.CalcTxOutHeight(ctx, keeper.GetVersion(), normalGasTOI)
	c.Assert(err, IsNil)
	c.Check(targetBlock, Equals, int64(31))

	millionGasTOI := normalGasTOI
	millionGasTOI.MaxGas = common.Gas{normalGasTOI.MaxGas[0]}
	millionGasTOI.MaxGas[0].Amount = millionGasTOI.MaxGas[0].Amount.MulUint64(1e6)
	millionGasValue := pool.AssetValueInRune(millionGasTOI.MaxGas[0].Amount)
	c.Check(millionGasValue.String(), Equals, "484273392786", Commentf("%s", millionGasValue.String()))
	targetBlock, _, err = txout.CalcTxOutHeight(ctx, keeper.GetVersion(), millionGasTOI)
	c.Assert(err, IsNil)
	c.Check(targetBlock, Equals, int64(738)) // This would have the maximum delay in contrast to with normal gas, due to its higher value.

	thousandSizeTOI := toi
	thousandSizeTOI.Coin.Amount = toi.Coin.Amount.MulUint64(1000)
	thousandSizeTOIValue := pool.AssetValueInRune(thousandSizeTOI.Coin.Amount)
	c.Check(thousandSizeTOIValue.Uint64(), Equals, uint64(129_139_57140964), Commentf("%d", thousandSizeTOIValue.Uint64()))

	targetBlock, _, err = txout.CalcTxOutHeight(ctx, keeper.GetVersion(), thousandSizeTOI)
	c.Assert(err, IsNil)
	c.Check(targetBlock, Equals, int64(738))
	c.Assert(keeper.AppendTxOut(ctx, targetBlock, thousandSizeTOI), IsNil)
	// (sumValue / minTxOutVolumeThreshold) * common.One = (129_397_85055246 / 25_00000000) * 1_00000000 = 5_175_00000000,
	// which reduces the 25_00000000 TxOutDelayRate to 1.
	// value / TxOutDelayRate is then 129_139_57140964 / 1, which is capped at MaxTxOutOffset (720).
	// 18 + 720 = 738

	// Now check the effect on TxOutDelayRate from the already-scheduled value.
	targetBlock, _, err = txout.CalcTxOutHeight(ctx, keeper.GetVersion(), toi)
	c.Assert(err, IsNil)
	c.Check(targetBlock, Equals, int64(739))
	// As above, sumValue reduces TxOutDelayRate to 1.
	// value / TxOutDelayRate is then 129_13957141 / 1, which is capped at MaxTxOutOffset (720).
	// 18 + 720 = 738, but since that block isn't empty (and the value sum would be greater than MinTxOutVolumeThreshold)
	// the outbound is scheduled for one block later, 739.
}

func (s TxOutStoreVCURSuite) TestAddOutTxItem_MultipleOutboundWillNotBeScheduledAtTheSameBlockHeight(c *C) {
	SetupConfigForTest()
	w := getHandlerTestWrapper(c, 1, true, true)
	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	w.keeper.SetMimir(w.ctx, constants.MinTxOutVolumeThreshold.String(), 100000000000)
	w.keeper.SetMimir(w.ctx, constants.TxOutDelayRate.String(), 2500000000)
	w.keeper.SetMimir(w.ctx, constants.MaxTxOutOffset.String(), 720)
	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// this should be sent via asgard
	item := TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(80*common.One)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 0)
	//  the outbound has been delayed

	item1 := TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(10*common.One)),
	}

	ok, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item1, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	msgs, err = txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	//  the smaller outbound hasn't been delayed
	c.Assert(msgs[0].VaultPubKey.String(), Equals, vault.PubKey.String())
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(9_99925000)), Equals, true, Commentf("%d", msgs[0].Coin.Amount.Uint64()))

	newCtx := w.ctx.WithBlockHeight(4)
	msgs, err = txOutStore.GetOutboundItems(newCtx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1) // the delayed outbound's height has been reached
	c.Assert(msgs[0].VaultPubKey.String(), Equals, vault.PubKey.String())
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(79_99925000)), Equals, true, Commentf("%d", msgs[0].Coin.Amount.Uint64()))

	// make sure outbound_height has been set correctly (to the furthest-future outbound height)
	afterVoter, err := w.keeper.GetObservedTxInVoter(w.ctx, inTxID)
	c.Assert(err, IsNil)
	c.Assert(afterVoter.OutboundHeight, Equals, int64(4))

	item.Chain = common.SWITCHLYChain
	item.Coin = common.NewCoin(common.SwitchNative, cosmos.NewUint(100*common.One))
	item.ToAddress = GetRandomTHORAddress()
	ok, err = txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// make sure outbound_height has not been lowered
	afterVoter1, err := w.keeper.GetObservedTxInVoter(w.ctx, inTxID)
	c.Assert(err, IsNil)
	c.Assert(afterVoter1.OutboundHeight, Equals, int64(4))
}

func (s TxOutStoreVCURSuite) TestAddOutTxItemInteractionWithPool(c *C) {
	w := getHandlerTestWrapper(c, 1, true, true)
	pool, err := w.keeper.GetPool(w.ctx, common.DOGEAsset)
	c.Assert(err, IsNil)
	// Set unequal values for the pool balances for this test.
	pool.BalanceAsset = cosmos.NewUint(50 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pool.Asset = common.DOGEAsset
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, IsNil)

	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(common.DOGEAsset, cosmos.NewUint(100*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, vault), IsNil)

	inTxID := GetRandomTxHash()
	item := TxOutItem{
		Chain:     common.DOGEChain,
		ToAddress: GetRandomDOGEAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.DOGEAsset, cosmos.NewUint(20*common.One)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	success, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(success, Equals, true)
	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
	c.Assert(msgs[0].Coin.Amount.Equal(cosmos.NewUint(1999925000)), Equals, true, Commentf("%d", msgs[0].Coin.Amount.Uint64()))
	pool, err = w.keeper.GetPool(w.ctx, common.DOGEAsset)
	c.Assert(err, IsNil)
	// Let:
	//   R_0 := the initial pool Rune balance
	//   A_0 := the initial pool Asset balance
	//   a   := the gas amount in Asset
	// Then the expected pool balances are:
	//   A_1 = A_0 + a = 50e8 + (20e8 - 1999925000) = 5000075000
	//   R_1 = R_0 - R_0 * a / (A_0 + a)  // slip formula
	//       = 100e8 - 100e8 * (20e8 - 1999925000) / (50e8 + (20e8 - 1999925000)) = 9999850002
	c.Assert(pool.BalanceAsset.Equal(cosmos.NewUint(5000075000)), Equals, true, Commentf("%d", pool.BalanceAsset.Uint64()))
	c.Assert(pool.BalanceRune.Equal(cosmos.NewUint(9999850002)), Equals, true, Commentf("%d", pool.BalanceRune.Uint64()))
}

func (s TxOutStoreVCURSuite) TestAddOutTxItemSendingFromRetiredVault(c *C) {
	SetupConfigForTest()
	w := getHandlerTestWrapper(c, 1, true, true)
	activeVault1 := GetRandomVault()
	activeVault1.Type = AsgardVault
	activeVault1.Status = ActiveVault
	activeVault1.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, activeVault1), IsNil)

	activeVault2 := GetRandomVault()
	activeVault2.Type = AsgardVault
	activeVault2.Status = ActiveVault
	activeVault2.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, activeVault2), IsNil)

	retiringVault1 := GetRandomVault()
	retiringVault1.Type = AsgardVault
	retiringVault1.Status = RetiringVault
	retiringVault1.Coins = common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(10000*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(1000*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, retiringVault1), IsNil)
	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	w.keeper.SetMimir(w.ctx, constants.MinTxOutVolumeThreshold.String(), 10000000000000)
	w.keeper.SetMimir(w.ctx, constants.TxOutDelayRate.String(), 250000000000)
	w.keeper.SetMimir(w.ctx, constants.MaxTxOutOffset.String(), 720)
	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// this should be sent via asgard
	item := TxOutItem{
		Chain:     common.ETHChain,
		ToAddress: GetRandomETHAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.ETHAsset, cosmos.NewUint(120*common.One)),
	}
	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)
	c.Assert(msgs, HasLen, 1)
}

func (s TxOutStoreVCURSuite) TestAddOutTxItem_SecurityVersusOutboundNumber(c *C) {
	// The historical context of this example:
	// TxIn hash:  179BF41ED245E74F2B0A4B9B970ED1F5D11335B192641AE7268F7AA3C1ADB724
	// finalised_height:  7243175
	// Network version:  1.95.0 (see _Example2 for a less extreme, more recent example)

	// For a less extreme, more recent example:
	// 268D0DF45CC6E99F56C3DF2EEF2737CD40B0127C06D2B11E5D256E7558387D5C
	// finalised_height:  7838089
	// Network version:  1.97

	// Within this example vault bonds are treated as zero, using only assets to represent security.

	SetupConfigForTest()
	w := getHandlerTestWrapper(c, 1, true, true)

	assetEthTwt, err := common.NewAsset("ETH.TWT-BC2")
	c.Assert(err, IsNil)

	// Prepare the relevant Asgard vault PubKeys.
	z2lfPubKey := GetRandomPubKey()
	qe5vPubKey := GetRandomPubKey()
	yxy5PubKey := GetRandomPubKey()

	// This vault represents vault of pubkey .z2lf .
	activeVault1 := GetRandomVault()
	activeVault1.PubKey = z2lfPubKey
	activeVault1.Type = AsgardVault
	activeVault1.Status = ActiveVault
	activeVault1.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(459265206245)),
		common.NewCoin(assetEthTwt, cosmos.NewUint(102469368)),
		// For .z2lf and .qe5v, record BTC amount to represent them being less secure than .yxy5 .
		common.NewCoin(common.BTCAsset, cosmos.NewUint(19169688813)),
		// For .z2lf only, record ETH amount to represent it being less secure than .qe5v .
		common.NewCoin(common.ATOMAsset, cosmos.NewUint(184220933893)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, activeVault1), IsNil)

	// This vault represents vault of pubkey .qe5v .
	activeVault2 := GetRandomVault()
	activeVault2.PubKey = qe5vPubKey
	activeVault2.Type = AsgardVault
	activeVault2.Status = ActiveVault
	activeVault2.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(547549226806)),
		// Note that Asgard .z2lf and .qe5v have had their ETH.TWT balance pushed down to the same number.
		common.NewCoin(assetEthTwt, cosmos.NewUint(102469368)),
		// For .z2lf and .qe5v, record BTC amount to represent them being less secure than .yxy5 .
		common.NewCoin(common.BTCAsset, cosmos.NewUint(26440155891)),
		// Leaving out ETH amount to represent .qe5v having higher security than .z2lf .
	}
	c.Assert(w.keeper.SetVault(w.ctx, activeVault2), IsNil)

	// This vault represents vault of pubkey .yxy5 .
	activeVault3 := GetRandomVault()
	activeVault3.PubKey = yxy5PubKey
	activeVault3.Type = AsgardVault
	activeVault3.Status = ActiveVault
	activeVault3.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(510688596460)),
		common.NewCoin(assetEthTwt, cosmos.NewUint(15859492234966)),
		// Leaving out BTC and ETH amount to represent .yxy5 having the highest security .
	}
	c.Assert(w.keeper.SetVault(w.ctx, activeVault3), IsNil)

	// Setting pools to be able to represent the Asset values.
	pool, err := w.keeper.GetPool(w.ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	pool.BalanceAsset = cosmos.NewUint(1653258402395)
	pool.BalanceRune = cosmos.NewUint(248680012786574)
	pool.Asset = common.ETHAsset
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, IsNil)
	///
	pool, err = w.keeper.GetPool(w.ctx, assetEthTwt)
	c.Assert(err, IsNil)
	pool.BalanceAsset = cosmos.NewUint(89359597473914)
	pool.BalanceRune = cosmos.NewUint(46962864904253)
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, NotNil)
	pool.Asset = assetEthTwt
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, IsNil)
	///
	pool, err = w.keeper.GetPool(w.ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	pool.BalanceAsset = cosmos.NewUint(80362018825)
	pool.BalanceRune = cosmos.NewUint(837898672769246)
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, NotNil)
	pool.Asset = common.BTCAsset
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, IsNil)
	///
	pool, err = w.keeper.GetPool(w.ctx, common.ATOMAsset)
	c.Assert(err, IsNil)
	pool.BalanceAsset = cosmos.NewUint(694112527552)
	pool.BalanceRune = cosmos.NewUint(612691971161372)
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, NotNil)
	pool.Asset = common.ATOMAsset
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, IsNil)

	var vaultsSecurityCheck Vaults
	vaultsSecurityCheck = append(vaultsSecurityCheck, activeVault1)
	vaultsSecurityCheck = append(vaultsSecurityCheck, activeVault2)
	vaultsSecurityCheck = append(vaultsSecurityCheck, activeVault3)
	vaultsSecurityCheck = w.keeper.SortBySecurity(w.ctx, vaultsSecurityCheck, 300)
	// Confirm that the vaults from least to most secure are .z2lf, .qe5v, .yxy5 .
	c.Assert(vaultsSecurityCheck[0].PubKey, Equals, z2lfPubKey)
	c.Assert(vaultsSecurityCheck[1].PubKey, Equals, qe5vPubKey)
	c.Assert(vaultsSecurityCheck[2].PubKey, Equals, yxy5PubKey)

	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	w.keeper.SetMimir(w.ctx, constants.MinTxOutVolumeThreshold.String(), 10000000000000)
	w.keeper.SetMimir(w.ctx, constants.TxOutDelayRate.String(), 250000000000)
	maxTxOutOffset := int64(720)
	w.keeper.SetMimir(w.ctx, constants.MaxTxOutOffset.String(), maxTxOutOffset)
	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// this should be sent via asgard
	item := TxOutItem{
		Chain:     common.ETHChain,
		ToAddress: GetRandomETHAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(assetEthTwt, cosmos.NewUint(39076+39076+94830689368)),
		// This Coin amount is an estimate, given slight changes to pool RUNE amount in a block.
	}

	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())
	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)

	// Only one outbound is created from the TxOutItem.
	c.Assert(msgs, HasLen, 1)

	// The outbound is from the single vault able to fulfill it in only one outbound.
	c.Assert(msgs[0].VaultPubKey, Equals, yxy5PubKey)

	scheduledOutbounds := make([]TxOutItem, 0)
	for height := w.ctx.BlockHeight() + 1; height <= w.ctx.BlockHeight()+17280; height++ {
		txOut, err := w.mgr.Keeper().GetTxOut(w.ctx, height)
		c.Assert(err, IsNil)
		if height > w.ctx.BlockHeight()+maxTxOutOffset && len(txOut.TxArray) == 0 {
			// we've hit our max offset, and an empty block, we can assume the
			// rest will be empty as well
			break
		}
		scheduledOutbounds = append(scheduledOutbounds, txOut.TxArray...)
	}
	// There are no scheduled outbounds.
	c.Assert(scheduledOutbounds, HasLen, 0)
}

func (s TxOutStoreVCURSuite) TestAddOutTxItem_VaultStatusVersusOutboundNumber(c *C) {
	// Within this example vault bonds are treated as zero, using only assets to represent security.

	SetupConfigForTest()
	w := getHandlerTestWrapper(c, 1, true, true)

	// Prepare the relevant Asgard vault PubKeys.
	activeVaultPubKey := GetRandomPubKey()
	retiringVault1PubKey := GetRandomPubKey()
	retiringVault2PubKey := GetRandomPubKey()

	activeVault := GetRandomVault()
	activeVault.PubKey = activeVaultPubKey
	activeVault.Type = AsgardVault
	activeVault.Status = ActiveVault
	activeVault.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(60*common.One)),
		// In this example only one Active vault has received the asset (e.g. only one migrate round),
		// and does not have enough to satisfy a 100 * common.One outbound.
	}
	c.Assert(w.keeper.SetVault(w.ctx, activeVault), IsNil)

	retiringVault1 := GetRandomVault()
	retiringVault1.PubKey = retiringVault1PubKey
	retiringVault1.Type = AsgardVault
	retiringVault1.Status = RetiringVault
	retiringVault1.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(80*common.One)),
		// Having the most assets, this vault is the least secure (most preferred for outbounds).
	}
	c.Assert(w.keeper.SetVault(w.ctx, retiringVault1), IsNil)

	retiringVault2 := GetRandomVault()
	retiringVault2.PubKey = retiringVault2PubKey
	retiringVault2.Type = AsgardVault
	retiringVault2.Status = RetiringVault
	retiringVault2.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(70*common.One)),
	}
	c.Assert(w.keeper.SetVault(w.ctx, retiringVault2), IsNil)

	// Setting a pool to be able to represent the Asset values.
	pool, err := w.keeper.GetPool(w.ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	pool.BalanceAsset = cosmos.NewUint(500 * common.One)
	pool.BalanceRune = cosmos.NewUint(75_000 * common.One)
	pool.Asset = common.ETHAsset
	err = w.keeper.SetPool(w.ctx, pool)
	c.Assert(err, IsNil)

	var vaultsSecurityCheck Vaults
	vaultsSecurityCheck = append(vaultsSecurityCheck, activeVault)
	vaultsSecurityCheck = append(vaultsSecurityCheck, retiringVault1)
	vaultsSecurityCheck = append(vaultsSecurityCheck, retiringVault2)
	vaultsSecurityCheck = w.keeper.SortBySecurity(w.ctx, vaultsSecurityCheck, 300)
	// Confirm that these vaults from least to most secure are retiringVault1, retiringVault2, activeVault .
	// Keep in mind that all else being equal, choosing outbounds from less secure vaults is preferred.
	c.Assert(vaultsSecurityCheck[0].PubKey, Equals, retiringVault1PubKey)
	c.Assert(vaultsSecurityCheck[1].PubKey, Equals, retiringVault2PubKey)
	c.Assert(vaultsSecurityCheck[2].PubKey, Equals, activeVaultPubKey)

	acc1 := GetRandomValidatorNode(NodeActive)
	acc2 := GetRandomValidatorNode(NodeActive)
	acc3 := GetRandomValidatorNode(NodeActive)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc1), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc2), IsNil)
	c.Assert(w.keeper.SetNodeAccount(w.ctx, acc3), IsNil)

	w.keeper.SetMimir(w.ctx, constants.MinTxOutVolumeThreshold.String(), 10000000000000)
	w.keeper.SetMimir(w.ctx, constants.TxOutDelayRate.String(), 250000000000)
	maxTxOutOffset := int64(720)
	w.keeper.SetMimir(w.ctx, constants.MaxTxOutOffset.String(), maxTxOutOffset)
	// Create voter
	inTxID := GetRandomTxHash()
	voter := NewObservedTxVoter(inTxID, common.ObservedTxs{
		common.ObservedTx{
			Tx:             GetRandomTx(),
			Status:         common.Status_incomplete,
			BlockHeight:    1,
			Signers:        []string{w.activeNodeAccount.NodeAddress.String(), acc1.NodeAddress.String(), acc2.NodeAddress.String()},
			KeysignMs:      0,
			FinaliseHeight: 1,
		},
	})
	w.keeper.SetObservedTxInVoter(w.ctx, voter)

	// this should be sent via asgard
	item := TxOutItem{
		Chain:     common.ETHChain,
		ToAddress: GetRandomETHAddress(),
		InHash:    inTxID,
		Coin:      common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
		// Cannot be fulfilled by any single vault
	}

	txOutStore := newTxOutStorageVCUR(w.keeper, w.mgr.GetConstants(), w.mgr.EventMgr(), w.mgr.GasMgr())

	ok, err := txOutStore.TryAddTxOutItem(w.ctx, w.mgr, item, cosmos.ZeroUint())
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	msgs, err := txOutStore.GetOutboundItems(w.ctx)
	c.Assert(err, IsNil)

	// The outbounds are not yet in the outbound queue, but should now be scheduled.
	c.Assert(msgs, HasLen, 0)

	scheduledOutbounds := make([]TxOutItem, 0)
	for height := w.ctx.BlockHeight() + 1; height <= w.ctx.BlockHeight()+17280; height++ {
		txOut, err := w.mgr.Keeper().GetTxOut(w.ctx, height)
		c.Assert(err, IsNil)
		if height > w.ctx.BlockHeight()+maxTxOutOffset && len(txOut.TxArray) == 0 {
			// we've hit our max offset, and an empty block, we can assume the
			// rest will be empty as well
			break
		}
		scheduledOutbounds = append(scheduledOutbounds, txOut.TxArray...)
	}
	// Two scheduled outbounds are created, because the prepareTxOutItem logic prefers 2 outbounds with zero remaining
	// to one outbound with non-zero remaining (and an "insufficient funds for outbound request" error).
	c.Assert(scheduledOutbounds, HasLen, 2)

	// As Active vaults are preferred to Retiring vaults (less migration keysign burden),
	// the two outbounds are from the Active vault and the less secure Retiring vault.
	c.Assert(scheduledOutbounds[0].VaultPubKey, Equals, activeVaultPubKey)
	c.Assert(scheduledOutbounds[1].VaultPubKey, Equals, retiringVault1PubKey)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutEqualDistribution(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}

	// Define test cases
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(30)
	clout2 := cosmos.NewUint(30)
	runeValue := cosmos.NewUint(60)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that the clouts were split equally and no rune value remains
	c.Check(amountFromClout1.Equal(cosmos.NewUint(30)), check.Equals, true)
	c.Check(amountFromClout2.Equal(cosmos.NewUint(30)), check.Equals, true)
	c.Check(remainingRune.IsZero(), check.Equals, true)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutOneSideExceeds(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}

	// Define test cases
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(70)
	clout2 := cosmos.NewUint(20)
	runeValue := cosmos.NewUint(60)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that clout2 took all it could, and clout1 took the remainder up to the limit
	c.Check(amountFromClout1.Equal(cosmos.NewUint(46)), check.Equals, true, Commentf("%d", amountFromClout1.Uint64()))
	c.Check(amountFromClout2.Equal(cosmos.NewUint(14)), check.Equals, true, Commentf("%d", amountFromClout2.Uint64()))
	c.Check(remainingRune.Equal(cosmos.NewUint(0)), check.Equals, true)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutUnevenDistribution(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}

	// Define test cases
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(60)
	clout2 := cosmos.NewUint(25)
	runeValue := cosmos.NewUint(50)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that the clouts were split according to their capacity
	c.Check(amountFromClout1.Equal(cosmos.NewUint(35)), check.Equals, true, Commentf("%d", amountFromClout1.Uint64()))
	c.Check(amountFromClout2.Equal(cosmos.NewUint(15)), check.Equals, true, Commentf("%d", amountFromClout2.Uint64()))
	c.Check(remainingRune.Equal(cosmos.NewUint(0)), check.Equals, true)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutOverLimit(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}

	// Define test cases
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(80)
	clout2 := cosmos.NewUint(80)
	runeValue := cosmos.NewUint(100)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that the clouts were split up to the limit
	c.Check(amountFromClout1.Equal(cosmos.NewUint(50)), check.Equals, true)
	c.Check(amountFromClout2.Equal(cosmos.NewUint(50)), check.Equals, true)
	c.Check(remainingRune.Equal(cosmos.NewUint(0)), check.Equals, true)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutWithZeroRuneValue(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}

	// Define test cases
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(40)
	clout2 := cosmos.NewUint(40)
	runeValue := cosmos.NewUint(0)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that no clout was taken from either and rune value remains zero
	c.Check(amountFromClout1.IsZero(), check.Equals, true)
	c.Check(amountFromClout2.IsZero(), check.Equals, true)
	c.Check(remainingRune.IsZero(), check.Equals, true)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutCloutsAtLimit(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(50)
	clout2 := cosmos.NewUint(50)
	runeValue := cosmos.NewUint(30)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that the clouts were split equally and no rune value remains
	c.Check(amountFromClout1.Equal(cosmos.NewUint(15)), check.Equals, true)
	c.Check(amountFromClout2.Equal(cosmos.NewUint(15)), check.Equals, true)
	c.Check(remainingRune.Equal(cosmos.NewUint(0)), check.Equals, true)
}

func (s *TxOutStoreVCURSuite) TestSplitCloutRuneValueExceedingClouts(c *check.C) {
	ctx, _ := setupManagerForTest(c)
	tos := TxOutStorageVCUR{}
	swapperCloutLimit := cosmos.NewUint(100)
	clout1 := cosmos.NewUint(40)
	clout2 := cosmos.NewUint(40)
	runeValue := cosmos.NewUint(120)

	// Call function under test
	amountFromClout1, amountFromClout2, remainingRune := tos.splitClout(ctx, swapperCloutLimit, clout1, clout2, runeValue)

	// Assert that the clouts took the maximum they could, and the remaining rune is correct
	c.Check(amountFromClout1.Equal(cosmos.NewUint(40)), check.Equals, true)
	c.Check(amountFromClout2.Equal(cosmos.NewUint(40)), check.Equals, true)
	c.Check(remainingRune.Equal(cosmos.NewUint(40)), check.Equals, true)
}
