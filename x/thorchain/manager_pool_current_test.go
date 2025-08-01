package thorchain

import (
	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

type PoolMgrVCURSuite struct{}

var _ = Suite(&PoolMgrVCURSuite{})

func (s *PoolMgrVCURSuite) TestEnableNextPool(c *C) {
	var err error
	ctx, k := setupKeeperForTest(c)
	mgr := NewDummyMgrWithKeeper(k)
	c.Assert(err, IsNil)
	pool := NewPool()
	pool.Asset = common.ETHAsset
	pool.Status = PoolAvailable
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	pool = NewPool()
	pool.Asset = common.BTCAsset // gas pool should be enabled by default
	pool.Status = PoolAvailable
	pool.BalanceRune = cosmos.NewUint(50 * common.One)
	pool.BalanceAsset = cosmos.NewUint(50 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	usdcAsset, err := common.NewAsset("ETH.USDC-0X9999999999999999999999999999999999999999")
	c.Assert(err, IsNil)
	pool = NewPool()
	pool.Asset = usdcAsset
	pool.Status = PoolStaged
	pool.BalanceRune = cosmos.NewUint(40 * common.One)
	pool.BalanceAsset = cosmos.NewUint(40 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	xmrAsset, err := common.NewAsset("XMR.XMR")
	c.Assert(err, IsNil)
	pool = NewPool()
	pool.Asset = xmrAsset
	pool.Status = PoolStaged
	pool.BalanceRune = cosmos.NewUint(40 * common.One)
	pool.BalanceAsset = cosmos.NewUint(0 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	// usdAsset
	usdAsset, err := common.NewAsset("ETH.TUSDB")
	c.Assert(err, IsNil)
	pool = NewPool()
	pool.Asset = usdAsset
	pool.Status = PoolStaged
	pool.BalanceRune = cosmos.NewUint(140 * common.One)
	pool.BalanceAsset = cosmos.NewUint(0 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	poolMgr := newPoolMgrVCUR()

	// should enable BTC
	c.Assert(poolMgr.cyclePools(ctx, 100, 1, 0, mgr), IsNil)
	pool, err = k.GetPool(ctx, common.BTCAsset)
	c.Assert(err, IsNil)
	c.Check(pool.Status, Equals, PoolAvailable)

	// should enable ETH
	c.Assert(poolMgr.cyclePools(ctx, 100, 1, 0, mgr), IsNil)
	pool, err = k.GetPool(ctx, usdcAsset)
	c.Assert(err, IsNil)
	c.Check(pool.Status, Equals, PoolAvailable)

	// should NOT enable XMR, since it has no assets
	c.Assert(poolMgr.cyclePools(ctx, 100, 1, 10*common.One, mgr), IsNil)
	pool, err = k.GetPool(ctx, xmrAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.IsEmpty(), Equals, false)
	c.Check(pool.Status, Equals, PoolStaged)
	c.Check(pool.BalanceRune.Uint64(), Equals, uint64(30*common.One))
}

func (s *PoolMgrVCURSuite) TestAbandonPool(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewDummyMgrWithKeeper(k)
	usdAsset, err := common.NewAsset("ETH.TUSDB")
	c.Assert(err, IsNil)
	pool := NewPool()
	pool.Asset = usdAsset
	pool.Status = PoolStaged
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(usdAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	}
	c.Assert(k.SetVault(ctx, vault), IsNil)

	runeAddr := GetRandomRUNEAddress()
	ethAddr := GetRandomETHAddress()
	lp := LiquidityProvider{
		Asset:        usdAsset,
		RuneAddress:  runeAddr,
		AssetAddress: ethAddr,
		Units:        cosmos.ZeroUint(),
		PendingRune:  cosmos.ZeroUint(),
		PendingAsset: cosmos.ZeroUint(),
	}
	k.SetLiquidityProvider(ctx, lp)

	poolMgr := newPoolMgrVCUR()

	// add event manager to context to intecept withdraw event
	em := cosmos.NewEventManager()
	ctx = ctx.WithEventManager(em)
	eventManager, err := GetEventManager(GetCurrentVersion())
	c.Assert(err, IsNil)
	mgr.eventMgr = eventManager

	// cycle pools
	c.Assert(poolMgr.cyclePools(ctx, 100, 1, 100*common.One, mgr), IsNil)

	// check withdraw event (keys must exist with empty values for midgard)
	expected := map[string]string{
		"pool":                     "ETH.TUSDB",
		"liquidity_provider_units": "0",
		"basis_points":             "10000",
		"asymmetry":                "0.000000000000000000",
		"emit_asset":               "0",
		"emit_rune":                "0",
		"id":                       "0000000000000000000000000000000000000000000000000000000000000000",
		"chain":                    "SWITCHLY",
		"from":                     runeAddr.String(),
		"to":                       "",
		"coin":                     "0 SWITCHLY.SWITCH",
		"memo":                     "",
	}
	for _, e := range em.Events() {
		if e.Type == "withdraw" {
			actual := make(map[string]string)
			for _, attr := range e.Attributes {
				actual[attr.Key] = attr.Value
			}
			c.Assert(actual, DeepEquals, expected)
		}
	}

	// check pool was deleted
	pool, err = k.GetPool(ctx, usdAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceRune.IsZero(), Equals, true)
	c.Assert(pool.BalanceAsset.IsZero(), Equals, true)

	// check vault remove pool asset
	vault, err = k.GetVault(ctx, vault.PubKey)
	c.Assert(err, IsNil)
	c.Assert(vault.HasAsset(usdAsset), Equals, false)
	c.Assert(vault.CoinLength(), Equals, 1)

	// check that liquidity provider got removed
	count := 0
	iterator := k.GetLiquidityProviderIterator(ctx, usdAsset)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	c.Assert(count, Equals, 0)
}

func (s *PoolMgrVCURSuite) TestDemotePoolWithLowLiquidityFees(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewDummyMgrWithKeeper(k)
	usdAsset, err := common.NewAsset("ETH.TUSDB")
	c.Assert(err, IsNil)
	pool := NewPool()
	pool.Asset = usdAsset
	pool.Status = PoolStaged
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	c.Assert(k.SetPool(ctx, pool), IsNil)

	poolETH := NewPool()
	poolETH.Asset = common.ETHAsset
	poolETH.Status = PoolAvailable
	poolETH.BalanceRune = cosmos.NewUint(100000 * common.One)
	poolETH.BalanceAsset = cosmos.NewUint(100000 * common.One)
	c.Assert(k.SetPool(ctx, poolETH), IsNil)

	ethUsdc, err := common.NewAsset("ETH.USDC-0X9999999999999999999999999999999999999999")
	c.Assert(err, IsNil)
	poolUsdc := NewPool()
	poolUsdc.Asset = ethUsdc
	poolUsdc.Status = PoolAvailable
	poolUsdc.BalanceRune = cosmos.NewUint(100000 * common.One)
	poolUsdc.BalanceAsset = cosmos.NewUint(100000 * common.One)
	c.Assert(k.SetPool(ctx, poolUsdc), IsNil)

	vault := GetRandomVault()
	vault.Coins = common.Coins{
		common.NewCoin(usdAsset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(100*common.One)),
	}
	c.Assert(k.SetVault(ctx, vault), IsNil)

	runeAddr := GetRandomRUNEAddress()
	ethAddr := GetRandomETHAddress()
	lp := LiquidityProvider{
		Asset:        usdAsset,
		RuneAddress:  runeAddr,
		AssetAddress: ethAddr,
		Units:        cosmos.ZeroUint(),
		PendingRune:  cosmos.ZeroUint(),
		PendingAsset: cosmos.ZeroUint(),
	}
	k.SetLiquidityProvider(ctx, lp)
	k.SetMimir(ctx, constants.MinimumPoolLiquidityFee.String(), 100000000)

	poolMgr := newPoolMgrVCUR()

	// cycle pools
	c.Assert(poolMgr.cyclePools(ctx, 100, 1, 100*common.One, mgr), IsNil)

	// check pool was deleted
	pool, err = k.GetPool(ctx, usdAsset)
	c.Assert(err, IsNil)
	c.Assert(pool.BalanceRune.IsZero(), Equals, true)
	c.Assert(pool.BalanceAsset.IsZero(), Equals, true)

	// check vault remove pool asset
	vault, err = k.GetVault(ctx, vault.PubKey)
	c.Assert(err, IsNil)
	c.Assert(vault.HasAsset(usdAsset), Equals, false)
	c.Assert(vault.CoinLength(), Equals, 1)

	// check that liquidity provider got removed
	count := 0
	iterator := k.GetLiquidityProviderIterator(ctx, usdAsset)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		count++
	}
	c.Assert(count, Equals, 0)
	afterETHPool, err := k.GetPool(ctx, common.ETHAsset)
	c.Assert(err, IsNil)
	c.Assert(afterETHPool.Status == PoolAvailable, Equals, true)
	afterETHEth, err := k.GetPool(ctx, ethUsdc)
	c.Assert(err, IsNil)
	c.Assert(afterETHEth.Status == PoolStaged, Equals, true)
}

func (s *PoolMgrVCURSuite) TestPoolMeetTradingVolumeCriteria(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewDummyMgrWithKeeper(k)
	pm := newPoolMgrVCUR()

	asset := common.BTCAsset

	pool := Pool{
		Asset:        asset,
		BalanceAsset: cosmos.NewUint(1000 * common.One),
		BalanceRune:  cosmos.NewUint(1000 * common.One),
	}

	minFee := cosmos.ZeroUint()
	meets := pm.poolMeetTradingVolumeCriteria(ctx, mgr, pool, minFee)
	c.Assert(meets, Equals, true,
		Commentf("all pools should meet criteria if min fees is zero"))

	minFee = cosmos.NewUint(10 * common.One)
	meets = pm.poolMeetTradingVolumeCriteria(ctx, mgr, pool, minFee)
	c.Assert(meets, Equals, false,
		Commentf("pool with no fees collected should not meet criteria"))

	err := k.AddToLiquidityFees(ctx, pool.Asset, minFee.Add(cosmos.NewUint(1)))
	c.Assert(err, IsNil)
	cur, err := k.GetRollingPoolLiquidityFee(ctx, pool.Asset)
	c.Assert(err, IsNil)
	c.Assert(cur, Equals, minFee.Add(cosmos.NewUint(1)).Uint64())

	meets = pm.poolMeetTradingVolumeCriteria(ctx, mgr, pool, minFee)
	c.Assert(meets, Equals, true,
		Commentf("pool should meet min fee criteria"))
}

func (s *PoolMgrVCURSuite) TestRemoveAssetFromVault(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewDummyMgrWithKeeper(k)
	pm := newPoolMgrVCUR()

	asset := common.BTCAsset

	v0 := GetRandomVault()
	v0.Coins = common.Coins{
		common.NewCoin(common.ETHAsset, cosmos.NewUint(1*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(10*common.One)),
	}
	c.Assert(k.SetVault(ctx, v0), IsNil)
	c.Assert(v0.HasAsset(asset), Equals, false,
		Commentf("vault0 should not have asset balance"))

	v1 := GetRandomVault()
	v1.Coins = common.Coins{
		common.NewCoin(asset, cosmos.NewUint(100*common.One)),
		common.NewCoin(common.ETHAsset, cosmos.NewUint(1000*common.One)),
	}
	c.Assert(k.SetVault(ctx, v1), IsNil)
	c.Assert(v1.HasAsset(asset), Equals, true,
		Commentf("vault1 should have asset balance"))

	pm.removeAssetFromVault(ctx, asset, mgr)

	v0, err := k.GetVault(ctx, v0.PubKey)
	c.Assert(err, IsNil)
	c.Assert(v0.HasAsset(asset), Equals, false,
		Commentf("vault0 should still not have asset balance"))

	v1, err = k.GetVault(ctx, v1.PubKey)
	c.Assert(err, IsNil)
	c.Assert(v1.HasAsset(asset), Equals, false,
		Commentf("vault1 should no longer have asset"))
}

func (s *PoolMgrVCURSuite) TestRemoveLiquidityProviders(c *C) {
	ctx, k := setupKeeperForTest(c)
	mgr := NewDummyMgrWithKeeper(k)
	pm := newPoolMgrVCUR()

	countLiquidityProviders := func(ctx cosmos.Context, k keeper.Keeper, asset common.Asset) int {
		count := 0
		iterator := k.GetLiquidityProviderIterator(ctx, asset)
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			count++
		}
		return count
	}

	asset := common.BTCAsset

	c.Assert(countLiquidityProviders(ctx, k, asset), Equals, 0,
		Commentf("should not have lps before adding"))

	k.SetLiquidityProvider(ctx, LiquidityProvider{
		Asset:       asset,
		RuneAddress: GetRandomRUNEAddress(),
	})
	k.SetLiquidityProvider(ctx, LiquidityProvider{
		Asset:       asset,
		RuneAddress: GetRandomRUNEAddress(),
	})
	c.Assert(countLiquidityProviders(ctx, k, asset), Equals, 2,
		Commentf("should have 2 lps after adding"))

	pm.removeLiquidityProviders(ctx, asset, mgr)

	c.Assert(countLiquidityProviders(ctx, k, asset), Equals, 0,
		Commentf("should have 0 lps after removing"))
}

func (s *PoolMgrVCURSuite) TestCommitPendingLiquidity(c *C) {
	var err error
	ctx, mgr := setupManagerForTest(c)

	pendingLiquidityAgeLimit := mgr.Keeper().GetConfigInt64(ctx, constants.PendingLiquidityAgeLimit)
	ctx = ctx.WithBlockHeight(pendingLiquidityAgeLimit + 10)

	pm := newPoolMgrVCUR()

	na := GetRandomValidatorNode(NodeActive)
	c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)

	asset := common.BTCAsset

	pool := NewPool()
	pool.Asset = asset
	pool.BalanceRune = cosmos.NewUint(100_000)
	pool.BalanceAsset = cosmos.NewUint(100_000)
	pool.PendingInboundRune = cosmos.OneUint()
	pool.PendingInboundAsset = cosmos.OneUint()
	pool.LPUnits = cosmos.NewUint(1000)
	c.Check(mgr.Keeper().SetPool(ctx, pool), IsNil)

	lp1 := LiquidityProvider{
		Asset:         pool.Asset,
		RuneAddress:   GetRandomRUNEAddress(),
		AssetAddress:  GetRandomBTCAddress(),
		PendingRune:   cosmos.NewUint(501),
		Units:         cosmos.NewUint(300),
		LastAddHeight: 1,
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp1)

	lp2 := LiquidityProvider{
		Asset:         pool.Asset,
		RuneAddress:   GetRandomRUNEAddress(),
		AssetAddress:  GetRandomBTCAddress(),
		PendingAsset:  cosmos.NewUint(600),
		Units:         cosmos.NewUint(500),
		LastAddHeight: 1,
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp2)

	lp3 := LiquidityProvider{
		Asset:         pool.Asset,
		RuneAddress:   GetRandomRUNEAddress(),
		AssetAddress:  GetRandomBTCAddress(),
		PendingAsset:  cosmos.NewUint(811),
		Units:         cosmos.NewUint(200),
		LastAddHeight: ctx.BlockHeight(),
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp3)

	err = pm.commitPendingLiquidity(ctx, pool, mgr)
	c.Assert(err, IsNil)

	pool, err = mgr.Keeper().GetPool(ctx, pool.Asset)
	c.Assert(err, IsNil)
	c.Check(pool.BalanceRune.Uint64(), Equals, uint64(100_501), Commentf("%d", pool.BalanceRune.Uint64()))
	c.Check(pool.BalanceAsset.Uint64(), Equals, uint64(100_600), Commentf("%d", pool.BalanceAsset.Uint64()))

	lp, err := mgr.Keeper().GetLiquidityProvider(ctx, pool.Asset, lp1.GetAddress())
	c.Assert(err, IsNil)
	c.Check(lp.Units.Uint64(), Equals, uint64(302), Commentf("%d", lp.Units.Uint64()))
	c.Check(lp.PendingRune.Uint64(), Equals, uint64(0), Commentf("%d", lp.PendingRune.Uint64()))
	c.Check(lp.PendingAsset.Uint64(), Equals, uint64(0), Commentf("%d", lp.PendingAsset.Uint64()))

	lp, err = mgr.Keeper().GetLiquidityProvider(ctx, pool.Asset, lp2.GetAddress())
	c.Assert(err, IsNil)
	c.Check(lp.Units.Uint64(), Equals, uint64(502), Commentf("%d", lp.Units.Uint64()))
	c.Check(lp.PendingRune.Uint64(), Equals, uint64(0), Commentf("%d", lp.PendingRune.Uint64()))
	c.Check(lp.PendingAsset.Uint64(), Equals, uint64(0), Commentf("%d", lp.PendingAsset.Uint64()))

	lp, err = mgr.Keeper().GetLiquidityProvider(ctx, pool.Asset, lp3.GetAddress())
	c.Assert(err, IsNil)
	c.Check(lp.Units.Uint64(), Equals, uint64(200), Commentf("%d", lp.Units.Uint64()))
	c.Check(lp.PendingRune.Uint64(), Equals, uint64(0), Commentf("%d", lp.PendingRune.Uint64()))
	c.Check(lp.PendingAsset.Uint64(), Equals, uint64(811), Commentf("%d", lp.PendingAsset.Uint64()))
}
