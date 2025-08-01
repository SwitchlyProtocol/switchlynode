package thorchain

import (
	"sort"
	"testing"

	"github.com/cosmos/cosmos-sdk/types"
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

func TestPackage(t *testing.T) { TestingT(t) }

type ThorchainSuite struct{}

var _ = Suite(&ThorchainSuite{})

func (s *ThorchainSuite) TestLiquidityProvision(c *C) {
	var err error
	ctx, keeper := setupKeeperForTest(c)
	user1rune := GetRandomRUNEAddress()
	user1asset := GetRandomETHAddress()
	user2rune := GetRandomRUNEAddress()
	user2asset := GetRandomETHAddress()
	txID := GetRandomTxHash()
	constAccessor := constants.GetConstantValues(GetCurrentVersion())
	c.Assert(err, IsNil)

	// create eth pool
	pool := NewPool()
	pool.Asset = common.ETHAsset
	c.Assert(keeper.SetPool(ctx, pool), IsNil)
	addHandler := NewAddLiquidityHandler(NewDummyMgrWithKeeper(keeper))
	// liquidity provider for user1
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1rune, user1asset, txID, false, constAccessor)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1rune, user1asset, txID, false, constAccessor)
	c.Assert(err, IsNil)
	lp1, err := keeper.GetLiquidityProvider(ctx, common.ETHAsset, user1rune)
	c.Assert(err, IsNil)
	c.Check(lp1.Units.IsZero(), Equals, false)

	// liquidity provider for user2
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(75*common.One), cosmos.NewUint(75*common.One), user2rune, user2asset, txID, false, constAccessor)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(75*common.One), cosmos.NewUint(75*common.One), user2rune, user2asset, txID, false, constAccessor)
	c.Assert(err, IsNil)
	lp2, err := keeper.GetLiquidityProvider(ctx, common.ETHAsset, user2rune)
	c.Assert(err, IsNil)
	c.Check(lp2.Units.IsZero(), Equals, false)

	// withdraw for user1
	msg := NewMsgWithdrawLiquidity(GetRandomTx(), user1rune, cosmos.NewUint(10000), common.ETHAsset, common.EmptyAsset, GetRandomBech32Addr())
	_, _, _, _, err = withdraw(ctx, *msg, NewDummyMgrWithKeeper(keeper))
	c.Assert(err, IsNil)
	lp1, err = keeper.GetLiquidityProvider(ctx, common.ETHAsset, user1rune)
	c.Assert(err, IsNil)
	c.Check(lp1.Units.IsZero(), Equals, true)

	// withdraw for user2
	msg = NewMsgWithdrawLiquidity(GetRandomTx(), user2rune, cosmos.NewUint(10000), common.ETHAsset, common.EmptyAsset, GetRandomBech32Addr())
	_, _, _, _, err = withdraw(ctx, *msg, NewDummyMgrWithKeeper(keeper))
	c.Assert(err, IsNil)
	lp2, err = keeper.GetLiquidityProvider(ctx, common.ETHAsset, user2rune)
	c.Assert(err, IsNil)
	c.Check(lp2.Units.IsZero(), Equals, true)

	// check pool is now empty
	pool, err = keeper.GetPool(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	c.Check(pool.BalanceRune.IsZero(), Equals, true)
	remainGas := uint64(37500)
	c.Check(pool.BalanceAsset.Uint64(), Equals, remainGas) // leave a little behind for gas
	c.Check(pool.LPUnits.IsZero(), Equals, true)

	// liquidity provider for user1, again
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1rune, user1asset, txID, false, constAccessor)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), user1rune, user1asset, txID, false, constAccessor)
	c.Assert(err, IsNil)
	lp1, err = keeper.GetLiquidityProvider(ctx, common.ETHAsset, user1rune)
	c.Assert(err, IsNil)
	c.Check(lp1.Units.IsZero(), Equals, false)

	// check pool is NOT empty
	pool, err = keeper.GetPool(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	c.Check(pool.BalanceRune.Equal(cosmos.NewUint(200*common.One)), Equals, true)
	c.Check(pool.BalanceAsset.Equal(cosmos.NewUint(20000000000+remainGas)), Equals, true, Commentf("%d", pool.BalanceAsset.Uint64()))
	c.Check(pool.LPUnits.IsZero(), Equals, false)
}

func (s *ThorchainSuite) TestChurn(c *C) {
	ctx, mgr := setupManagerForTest(c)
	ver := GetCurrentVersion()
	consts := constants.GetConstantValues(ver)
	// create starting point, vault and four node active node accounts
	vault := GetRandomVault()
	vault.AddFunds(common.Coins{
		common.NewCoin(common.SwitchNative, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(79*common.One)),
	})
	c.Assert(mgr.Keeper().SaveNetworkFee(ctx, common.ETHChain, NetworkFee{
		Chain:              common.ETHChain,
		TransactionSize:    1,
		TransactionFeeRate: 25000,
	}), IsNil)
	c.Assert(mgr.Keeper().SetPool(ctx, Pool{
		BalanceRune:  cosmos.NewUint(common.One),
		BalanceAsset: cosmos.NewUint(common.One),
		Asset:        common.ETHAsset,
		LPUnits:      cosmos.NewUint(common.One),
		Status:       PoolAvailable,
	}), IsNil)
	addresses := make([]cosmos.AccAddress, 4)
	var existingValidators []string
	for i := 0; i <= 3; i++ {
		na := GetRandomValidatorNode(NodeActive)
		addresses[i] = na.NodeAddress
		na.SignerMembership = common.PubKeys{vault.PubKey}.Strings()
		if i == 0 { // give the first node account slash points
			na.RequestedToLeave = true
		}
		pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, na.ValidatorConsPubKey)
		if err != nil {
			ctx.Logger().Error("fail to parse consensus public key", "key", na.ValidatorConsPubKey, "error", err)
			continue
		}
		caddr := types.ValAddress(pk.Address()).String()
		existingValidators = append(existingValidators, caddr)
		vault.Membership = append(vault.Membership, na.PubKeySet.Secp256k1.String())
		c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)
	}
	c.Assert(mgr.Keeper().SetVault(ctx, vault), IsNil)

	// create new node account to rotate in
	na := GetRandomValidatorNode(NodeReady)
	c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)

	// trigger marking bad actors as well as a keygen
	rotateHeight := consts.GetInt64Value(constants.ChurnInterval) + vault.BlockHeight
	ctx = ctx.WithBlockHeight(rotateHeight)
	valMgr := newValidatorMgrVCUR(mgr.Keeper(), mgr.NetworkMgr(), mgr.TxOutStore(), mgr.EventMgr())
	c.Assert(valMgr.BeginBlock(ctx, mgr, existingValidators), IsNil)

	// check we've created a keygen, with the correct members
	keygenBlock, err := mgr.Keeper().GetKeygenBlock(ctx, ctx.BlockHeight())
	c.Assert(err, IsNil)
	c.Assert(keygenBlock.IsEmpty(), Equals, false)
	expected := append(vault.Membership[1:], na.PubKeySet.Secp256k1.String()) // nolint
	c.Assert(keygenBlock.Keygens, HasLen, 1)
	keygen := keygenBlock.Keygens[0]
	// sort our slices so they are in the same order
	sort.Slice(expected, func(i, j int) bool { return expected[i] < expected[j] })
	sort.Slice(keygen.Members, func(i, j int) bool { return keygen.Members[i] < keygen.Members[j] })
	c.Assert(expected, HasLen, len(keygen.Members))
	for i := range expected {
		c.Assert(expected[i], Equals, keygen.Members[i], Commentf("%d: %s <==> %s", i, expected[i], keygen.Members[i]))
	}

	// generate a tss keygen handler event
	newVaultPk := GetRandomPubKey()
	signer, err := common.PubKey(keygen.Members[0]).GetThorAddress()
	c.Assert(err, IsNil)
	keygenTime := int64(1024)
	msg, err := NewMsgTssPool(keygen.Members, newVaultPk, []byte("fakeSignature"), nil, AsgardKeygen, ctx.BlockHeight(), Blame{}, common.Chains{common.SwitchNative.Chain}.Strings(), signer, keygenTime)
	c.Assert(err, IsNil)
	tssHandler := NewTssHandler(mgr)

	voter := NewTssVoter(msg.ID, msg.PubKeys, msg.PoolPubKey)
	signers := make([]string, len(msg.PubKeys)-1)
	signatures := make([]string, len(msg.PubKeys)-1)
	for i, pk := range msg.PubKeys {
		if i == 0 {
			continue
		}
		var err error
		sig, err := common.PubKey(pk).GetThorAddress()
		c.Assert(err, IsNil)
		signers[i-1] = sig.String()
		signatures[i-1] = "fakeSignature"
	}
	voter.Signers = signers // ensure we have consensus, so handler is properly executed
	voter.Secp256K1Signatures = signatures
	mgr.Keeper().SetTssVoter(ctx, voter)

	_, err = tssHandler.Run(ctx, msg)
	c.Assert(err, IsNil)

	// check that we've rotated our vaults
	vault1, err := mgr.Keeper().GetVault(ctx, vault.PubKey)
	c.Assert(err, IsNil)
	c.Assert(vault1.Status, Equals, RetiringVault) // first vault should now be retiring
	vault2, err := mgr.Keeper().GetVault(ctx, newVaultPk)
	c.Assert(err, IsNil)
	c.Assert(vault2.Status, Equals, ActiveVault) // new vault should now be active
	c.Assert(vault2.Membership, HasLen, 4)

	// check our validators get rotated appropriately
	validators := valMgr.EndBlock(ctx, mgr)
	nas, err := mgr.Keeper().ListActiveValidators(ctx)
	c.Assert(err, IsNil)
	c.Assert(nas, HasLen, 4)
	c.Assert(validators, HasLen, 2)
	// ensure that the first one is rotated out and the new one is rotated in
	standby, err := mgr.Keeper().GetNodeAccount(ctx, addresses[0])
	c.Assert(err, IsNil)
	c.Check(standby.Status == NodeStandby, Equals, true)
	na, err = mgr.Keeper().GetNodeAccount(ctx, na.NodeAddress)
	c.Assert(err, IsNil)
	c.Check(na.Status == NodeActive, Equals, true)

	// check that the funds can be migrated from the retiring vault to the new
	// vault
	ctx = ctx.WithBlockHeight(vault1.StatusSince)
	err = mgr.NetworkMgr().EndBlock(ctx, mgr) // should attempt to send 20% of the coin values
	c.Assert(err, IsNil)
	vault, err = mgr.Keeper().GetVault(ctx, vault1.PubKey)
	c.Assert(err, IsNil)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
	item := items[0]
	c.Check(item.Coin.Amount.Uint64(), Equals, uint64(1579962500), Commentf("%d", item.Coin.Amount.Uint64()))

	// check we empty the rest at the last migration event
	// Ensure that the height is past the SigningTransactionPeriod,
	// so not deducting the previous migrate amount as a pending outbound.
	migrateInterval := consts.GetInt64Value(constants.FundMigrationInterval)
	signingTranactionPeriod := consts.GetInt64Value(constants.SigningTransactionPeriod)
	ctx = ctx.WithBlockHeight(vault.StatusSince + (migrateInterval * signingTranactionPeriod))
	vault, err = mgr.Keeper().GetVault(ctx, vault.PubKey)
	c.Assert(err, IsNil)
	c.Assert(vault.LenPendingTxBlockHeights(ctx.BlockHeight(), signingTranactionPeriod), Equals, 0)
	c.Check(mgr.NetworkMgr().EndBlock(ctx, mgr), IsNil) // should attempt to send 100% of the coin values
	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1, Commentf("%d", len(items)))
	item = items[0]
	c.Check(item.Coin.Amount.Uint64(), Equals, uint64(7899962500), Commentf("%d", item.Coin.Amount.Uint64()))
}

func (s *ThorchainSuite) TestRagnarok(c *C) {
	SetupConfigForTest()
	var err error
	ctx, mgr := setupManagerForTest(c)
	ctx = ctx.WithBlockHeight(10)
	ver := GetCurrentVersion()
	consts := constants.GetConstantValues(ver)
	c.Assert(mgr.Keeper().SaveNetworkFee(ctx, common.ETHChain, NetworkFee{
		Chain:              common.ETHChain,
		TransactionSize:    1,
		TransactionFeeRate: 10000,
	}), IsNil)

	// create active asgard vault
	asgard := GetRandomVault()
	c.Assert(mgr.Keeper().SetVault(ctx, asgard), IsNil)

	// create pools
	pool := NewPool()
	pool.Asset = common.ETHAsset
	pool.Status = PoolAvailable
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	boltAsset, err := common.NewAsset("ETH.BOLT-123")
	c.Assert(err, IsNil)
	pool.Asset = boltAsset
	pool.Status = PoolAvailable
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	pool = NewPool()
	pool.Asset = common.BTCAsset
	pool.Status = PoolAvailable
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	addHandler := NewAddLiquidityHandler(mgr)
	// add liquidity providers
	lp1 := GetRandomRUNEAddress() // LiquidityProvider1
	lp1asset := GetRandomETHAddress()
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(10*common.One), lp1, lp1asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, boltAsset, cosmos.NewUint(50*common.One), cosmos.NewUint(11*common.One), lp1, lp1asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	lp2 := GetRandomRUNEAddress() // liquidity provider 2
	lp2asset := GetRandomETHAddress()
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), lp2, lp2asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, boltAsset, cosmos.NewUint(20*common.One), cosmos.NewUint(4*common.One), lp2, lp2asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	lp3 := GetRandomRUNEAddress() // liquidity provider 3
	lp3asset := GetRandomETHAddress()
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), lp3, lp3asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)

	lp4 := GetRandomTHORAddress() // liquidity provider 4 , BTC
	lp4Asset := GetRandomBTCAddress()
	err = addHandler.addLiquidity(ctx, common.BTCAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(100*common.One), lp4, lp4Asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)

	lp5 := GetRandomTHORAddress() // Rune only
	err = addHandler.addLiquidity(ctx, common.BTCAsset, cosmos.NewUint(100*common.One), cosmos.ZeroUint(), lp5, common.NoAddress, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	lp6Asset := GetRandomBTCAddress() // BTC only

	err = addHandler.addLiquidity(ctx, common.BTCAsset, cosmos.ZeroUint(), cosmos.NewUint(100*common.One), common.NoAddress, lp6Asset, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)

	asgard.AddFunds(common.Coins{
		common.NewCoin(common.BTCAsset, cosmos.NewUint(101*common.One)),
	})

	lpsAssets := []common.Address{
		lp1asset, lp2asset, lp3asset,
	}

	// Add bonders/validators
	bonderCount := 3
	bonders := make(NodeAccounts, bonderCount)
	for i := 1; i <= bonderCount; i++ {
		na := GetRandomValidatorNode(NodeActive)
		na.Bond = cosmos.NewUint(1_000_000 * uint64(i) * common.One)
		FundModule(c, ctx, mgr.Keeper(), BondName, na.Bond.Uint64())
		c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)
		bonders[i-1] = na

		// Add bond to asgard
		asgard.AddFunds(common.Coins{
			common.NewCoin(common.SwitchNative, na.Bond),
		})
		c.Assert(mgr.Keeper().SetVault(ctx, asgard), IsNil)
	}

	// ////////////////////////////////////////////////////////
	// ////////////// Start Ragnarok Protocol /////////////////
	// ////////////////////////////////////////////////////////
	network := Network{
		BondRewardRune: cosmos.NewUint(1000_000 * common.One),
		TotalBondUnits: cosmos.NewUint(3 * 1014), // block height * node count
	}
	FundModule(c, ctx, mgr.Keeper(), ReserveName, 400_100_000*common.One)
	c.Assert(mgr.Keeper().SetNetwork(ctx, network), IsNil)
	ctx = ctx.WithBlockHeight(1024)

	for i := 1; i <= 11; i++ { // simulate each round of ragnarok (max of ten)
		c.Assert(mgr.ValidatorMgr().processRagnarok(ctx, mgr), IsNil)
		_, err := mgr.TxOutStore().GetOutboundItems(ctx)
		c.Assert(err, IsNil)
		// validate liquidity providers get their returns
		for _, lp := range lpsAssets {
			items := mgr.TxOutStore().GetOutboundItemByToAddress(ctx, lp)
			c.Assert(items, HasLen, 0)
		}
		mgr.TxOutStore().ClearOutboundItems(ctx) // clear out txs
		mgr.Keeper().SetRagnarokPending(ctx, 0)
		items, err := mgr.TxOutStore().GetOutboundItems(ctx)
		c.Assert(items, HasLen, 0)
		c.Assert(err, IsNil)
	}
}

func (s *ThorchainSuite) TestRagnarokNoOneLeave(c *C) {
	var err error
	ctx, mgr := setupManagerForTest(c)
	ctx = ctx.WithBlockHeight(10)
	ver := GetCurrentVersion()
	consts := constants.GetConstantValues(ver)

	// create active asgard vault
	asgard := GetRandomVault()
	c.Assert(mgr.Keeper().SetVault(ctx, asgard), IsNil)

	// create pools
	pool := NewPool()
	pool.Asset = common.ETHAsset
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	boltAsset, err := common.NewAsset("ETH.BOLT-123")
	c.Assert(err, IsNil)
	pool.Asset = boltAsset
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	addHandler := NewAddLiquidityHandler(NewDummyMgrWithKeeper(mgr.Keeper()))
	// add liquidity providers
	lp1 := GetRandomRUNEAddress() // LiquidityProvider1
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(100*common.One), cosmos.NewUint(10*common.One), lp1, lp1, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, boltAsset, cosmos.NewUint(50*common.One), cosmos.NewUint(11*common.One), lp1, lp1, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	lp2 := GetRandomRUNEAddress() // liquidity provider 2
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), lp2, lp2, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	err = addHandler.addLiquidity(ctx, boltAsset, cosmos.NewUint(20*common.One), cosmos.NewUint(4*common.One), lp2, lp2, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	lp3 := GetRandomRUNEAddress() // liquidity provider 3
	err = addHandler.addLiquidity(ctx, common.ETHAsset, cosmos.NewUint(155*common.One), cosmos.NewUint(15*common.One), lp3, lp3, GetRandomTxHash(), false, consts)
	c.Assert(err, IsNil)
	lps := []common.Address{
		lp1, lp2, lp3,
	}
	_ = lps

	// Add bonders/validators
	bonderCount := 4
	bonders := make(NodeAccounts, bonderCount)
	for i := 1; i <= bonderCount; i++ {
		na := GetRandomValidatorNode(NodeActive)
		na.ActiveBlockHeight = 5
		na.Bond = cosmos.NewUint(1_000_000 * uint64(i) * common.One)
		c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)
		bonders[i-1] = na

		// Add bond to asgard
		asgard.AddFunds(common.Coins{
			common.NewCoin(common.SwitchNative, na.Bond),
		})
		asgard.Membership = append(asgard.Membership, na.PubKeySet.Secp256k1.String())
		c.Assert(mgr.Keeper().SetVault(ctx, asgard), IsNil)
	}

	// Add reserve contributors
	contrib1 := GetRandomETHAddress()
	contrib2 := GetRandomETHAddress()
	reserves := ReserveContributors{
		NewReserveContributor(contrib1, cosmos.NewUint(400_000_000*common.One)),
		NewReserveContributor(contrib2, cosmos.NewUint(100_000*common.One)),
	}
	resHandler := NewReserveContributorHandler(mgr)
	for _, res := range reserves {
		asgard.AddFunds(common.Coins{
			common.NewCoin(common.SwitchNative, res.Amount),
		})
		msg := NewMsgReserveContributor(GetRandomTx(), res, bonders[0].NodeAddress)
		err := resHandler.handle(ctx, *msg)
		_ = err
		// c.Assert(err, IsNil)
	}
	c.Assert(mgr.Keeper().SetVault(ctx, asgard), IsNil)
	asgard.Membership = asgard.Membership[:len(asgard.Membership)-1]
	c.Assert(mgr.Keeper().SetVault(ctx, asgard), IsNil)
	// no validator should leave, because it trigger ragnarok
	updates := mgr.ValidatorMgr().EndBlock(ctx, mgr)
	c.Assert(updates, IsNil)
	ragnarokHeight, err := mgr.Keeper().GetRagnarokBlockHeight(ctx)
	c.Assert(err, IsNil)
	c.Assert(ragnarokHeight, Equals, ctx.BlockHeight())
	currentHeight := ctx.BlockHeight()
	migrateInterval := consts.GetInt64Value(constants.FundMigrationInterval)
	ctx = ctx.WithBlockHeight(currentHeight + migrateInterval)
	c.Assert(mgr.ValidatorMgr().BeginBlock(ctx, mgr, nil), IsNil)
	mgr.TxOutStore().ClearOutboundItems(ctx)
	c.Assert(mgr.ValidatorMgr().EndBlock(ctx, mgr), IsNil)
}
