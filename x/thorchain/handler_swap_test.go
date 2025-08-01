package thorchain

import (
	"errors"
	"fmt"
	"strings"

	se "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

type HandlerSwapSuite struct{}

var _ = Suite(&HandlerSwapSuite{})

func (s *HandlerSwapSuite) TestValidate(c *C) {
	ctx, _ := setupKeeperForTest(c)

	keeper := &TestSwapHandleKeeper{
		activeNodeAccount: GetRandomValidatorNode(NodeActive),
	}

	handler := NewSwapHandler(NewDummyMgrWithKeeper(keeper))

	txID := GetRandomTxHash()
	signerDOGEAddr := GetRandomDOGEAddress()
	observerAddr := keeper.activeNodeAccount.NodeAddress
	tx := common.NewTx(
		txID,
		signerDOGEAddr,
		signerDOGEAddr,
		common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.OneUint()),
		},
		common.Gas{
			common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
		},
		"",
	)
	msg := NewMsgSwap(tx, common.DOGEAsset, signerDOGEAddr, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, 0, observerAddr)
	err := handler.validate(ctx, *msg)
	c.Assert(err, IsNil)

	// bad aggregator reference
	msg.Aggregator = "zzzzzz"
	c.Assert(handler.validate(ctx, *msg), NotNil)

	// invalid msg
	msg = &MsgSwap{}
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)
}

type TestSwapHandleKeeper struct {
	keeper.KVStoreDummy
	pools             map[common.Asset]Pool
	activeNodeAccount NodeAccount
	synthSupply       cosmos.Uint
	haltChain         int64
	derivedAssets     bool
}

func (k *TestSwapHandleKeeper) GetConfigInt64(ctx cosmos.Context, key constants.ConstantName) int64 {
	val, _ := k.GetMimir(ctx, key.String())
	return val
}

func (k *TestSwapHandleKeeper) PoolExist(_ cosmos.Context, asset common.Asset) bool {
	asset = asset.GetLayer1Asset()
	_, ok := k.pools[asset]
	return ok
}

func (k *TestSwapHandleKeeper) GetPool(_ cosmos.Context, asset common.Asset) (Pool, error) {
	asset = asset.GetLayer1Asset()
	if pool, ok := k.pools[asset]; ok {
		return pool, nil
	}
	pool := NewPool()
	pool.Asset = asset
	return pool, nil
}

func (k *TestSwapHandleKeeper) GetPools(_ cosmos.Context) (pools Pools, err error) {
	for _, v := range k.pools {
		pools = append(pools, v)
	}
	return
}

func (k *TestSwapHandleKeeper) SetPool(_ cosmos.Context, pool Pool) error {
	asset := pool.Asset.GetLayer1Asset()
	k.pools[asset] = pool
	return nil
}

// IsActiveObserver see whether it is an active observer
func (k *TestSwapHandleKeeper) IsActiveObserver(_ cosmos.Context, addr cosmos.AccAddress) bool {
	return k.activeNodeAccount.NodeAddress.Equals(addr)
}

func (k *TestSwapHandleKeeper) GetNodeAccount(_ cosmos.Context, addr cosmos.AccAddress) (NodeAccount, error) {
	if k.activeNodeAccount.NodeAddress.Equals(addr) {
		return k.activeNodeAccount, nil
	}
	return NodeAccount{}, errors.New("not exist")
}

func (k *TestSwapHandleKeeper) AddToLiquidityFees(_ cosmos.Context, _ common.Asset, _ cosmos.Uint) error {
	return nil
}

func (k *TestSwapHandleKeeper) AddToSwapSlip(ctx cosmos.Context, asset common.Asset, fs cosmos.Int) error {
	return nil
}

func (k *TestSwapHandleKeeper) GetTotalSupply(_ cosmos.Context, _ common.Asset) cosmos.Uint {
	return k.synthSupply
}

func (k *TestSwapHandleKeeper) GetMimir(ctx cosmos.Context, key string) (int64, error) {
	if key == "MaxSynthPerPoolDepth" {
		return 5000, nil
	}
	if key == "EnableDerivedAssets" {
		if k.derivedAssets {
			return 1, nil
		}
		return 0, nil
	}
	return k.haltChain, nil
}

func (k *TestSwapHandleKeeper) GetMimirWithRef(ctx cosmos.Context, template string, ref ...any) (int64, error) {
	key := fmt.Sprintf(template, ref...)
	return k.GetMimir(ctx, key)
}

func (k *TestSwapHandleKeeper) MintToModule(_ cosmos.Context, _ string, _ common.Coin) error {
	return nil
}

func (k *TestSwapHandleKeeper) BurnFromModule(_ cosmos.Context, _ string, _ common.Coin) error {
	return nil
}

func (k *TestSwapHandleKeeper) SendFromModuleToModule(_ cosmos.Context, _, _ string, _ common.Coins) error {
	return nil
}

func (k *TestSwapHandleKeeper) GetModuleAddress(_ string) (common.Address, error) {
	return GetRandomTHORAddress(), nil
}

func (s *HandlerSwapSuite) TestValidation(c *C) {
	ctx, mgr := setupManagerForTest(c)
	pool := NewPool()
	pool.Asset = common.DOGEAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	pools := make(map[common.Asset]Pool)
	pools[pool.Asset] = pool
	keeper := &TestSwapHandleKeeper{
		pools:             pools,
		activeNodeAccount: GetRandomValidatorNode(NodeActive),
		synthSupply:       cosmos.ZeroUint(),
	}
	mgr.K = keeper
	mgr.txOutStore = NewTxStoreDummy()

	handler := NewSwapHandler(mgr)

	txID := GetRandomTxHash()
	signerDOGEAddr := GetRandomDOGEAddress()
	observerAddr := keeper.activeNodeAccount.NodeAddress
	tx := common.NewTx(
		txID,
		signerDOGEAddr,
		signerDOGEAddr,
		common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(common.One*100)),
		},
		common.Gas{
			common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
		},
		"",
	)
	msg := NewMsgSwap(tx, common.DOGEAsset.GetSyntheticAsset(), GetRandomTHORAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, 0, observerAddr)
	err := handler.validate(ctx, *msg)
	c.Assert(err, IsNil)

	// check that derived assets are no allowed
	msg.TargetAsset = common.BTCAsset.GetDerivedAsset()
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)
	// enable derived assets and try again
	keeper.derivedAssets = true
	err = handler.validate(ctx, *msg)
	c.Assert(err, IsNil)
	msg.TargetAsset = common.DOGEAsset.GetSyntheticAsset()

	// check that minting synths halts after hitting pool limit
	keeper.synthSupply = cosmos.NewUint(common.One * 200)
	mgr.K = keeper
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)
}

func (s *HandlerSwapSuite) TestValidationWithStreamingSwap(c *C) {
	ctx, mgr := setupManagerForTest(c)
	mgr.Keeper().SetMimir(ctx, "MaxStreamingSwapLength", 3850)
	mgr.Keeper().SetMimir(ctx, "MinBPStreamingSwap", 4)
	pool := NewPool()
	pool.Asset = common.DOGEAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	mgr.txOutStore = NewTxStoreDummy()

	na := GetRandomValidatorNode(NodeActive)
	c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)

	handler := NewSwapHandler(mgr)

	txID := GetRandomTxHash()
	signerDOGEAddr := GetRandomDOGEAddress()
	observerAddr := na.NodeAddress
	tx := common.NewTx(
		txID,
		signerDOGEAddr,
		signerDOGEAddr,
		common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(common.One*100)),
		},
		common.Gas{
			common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
		},
		"",
	)

	// happy path
	msg := NewMsgSwap(tx, common.DOGEAsset.GetSyntheticAsset(), GetRandomTHORAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 10, 20, observerAddr)
	err := handler.validate(ctx, *msg)
	c.Assert(err, IsNil)

	// test mimir shutdown
	mgr.Keeper().SetMimir(ctx, "StreamingSwapPause", 1)
	err = handler.validate(ctx, *msg)
	c.Assert(err, NotNil)
	mgr.Keeper().SetMimir(ctx, "StreamingSwapPause", 0)

	// check that validation fails due to synth cap
	mgr.Keeper().SetMimir(ctx, "MaxSynthPerPoolDepth", 1)
	// first swap should fail as it include the total value of the streaming swap
	c.Assert(handler.validate(ctx, *msg), NotNil)
	// second swap should NOT fail as the synth cap is ignored
	swp := msg.GetStreamingSwap()
	swp.Count = 3
	mgr.Keeper().SetStreamingSwap(ctx, swp)
	c.Assert(handler.validate(ctx, *msg), IsNil)
}

func (s *HandlerSwapSuite) TestHandle(c *C) {
	ctx, mgr := setupManagerForTest(c)
	keeper := &TestSwapHandleKeeper{
		pools:             make(map[common.Asset]Pool),
		activeNodeAccount: GetRandomValidatorNode(NodeActive),
		synthSupply:       cosmos.ZeroUint(),
	}
	mgr.txOutStore = NewTxStoreDummy()
	handler := NewSwapHandler(mgr)

	result, err := handler.Run(ctx, NewMsgMimir("what", 1, GetRandomBech32Addr()))
	c.Check(err, NotNil)
	c.Check(result, IsNil)
	c.Check(errors.Is(err, errInvalidMessage), Equals, true)

	txID := GetRandomTxHash()
	signerDOGEAddr := GetRandomDOGEAddress()
	observerAddr := keeper.activeNodeAccount.NodeAddress
	// no pool
	tx := common.NewTx(
		txID,
		signerDOGEAddr,
		signerDOGEAddr,
		common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.OneUint()),
		},
		common.Gas{
			common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
		},
		"",
	)
	msg := NewMsgSwap(tx, common.DOGEAsset, signerDOGEAddr, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, 0, observerAddr)

	pool := NewPool()
	pool.Asset = common.DOGEAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)

	// swap of only 0.00000001 would emit 0, thus rejected.
	_, err = handler.handle(ctx, *msg)
	c.Assert(err.Error(), Equals, "zero emit asset")

	tx = common.NewTx(
		txID,
		signerDOGEAddr,
		signerDOGEAddr,
		common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(2*common.One)),
		},
		common.Gas{
			common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
		},
		"",
	)
	msgSwapPriceProtection := NewMsgSwap(tx, common.DOGEAsset, signerDOGEAddr, cosmos.NewUint(2*common.One), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, 0, observerAddr)
	result, err = handler.handle(ctx, *msgSwapPriceProtection)
	c.Assert(err.Error(), Equals, errors.New("emit asset 192233756 less than price limit 200000000").Error())
	c.Assert(result, IsNil)

	poolTCAN := NewPool()
	tCanAsset, err := common.NewAsset("DOGE.TCAN-014")
	c.Assert(err, IsNil)
	poolTCAN.Asset = tCanAsset
	poolTCAN.BalanceAsset = cosmos.NewUint(334850000)
	poolTCAN.BalanceRune = cosmos.NewUint(2349500000)
	c.Assert(mgr.Keeper().SetPool(ctx, poolTCAN), IsNil)
	dogeAddr := GetRandomDOGEAddress()
	m, err := ParseMemo(mgr.GetVersion(), "swap:DOGE.DOGE:"+dogeAddr.String()+":121893238")
	c.Assert(err, IsNil)
	txIn := NewObservedTx(
		common.NewTx(GetRandomTxHash(), signerDOGEAddr, GetRandomDOGEAddress(),
			common.Coins{
				common.NewCoin(tCanAsset, cosmos.NewUint(20000000)),
			},
			common.Gas{
				common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
			},
			"swap:DOGE.DOGE:"+signerDOGEAddr.String()+":121893238",
		),
		1,
		GetRandomPubKey(), 1,
	)
	msgSwapFromTxIn, err := getMsgSwapFromMemo(m.(SwapMemo), txIn, observerAddr)
	c.Assert(err, IsNil)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 0)

	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil) // reset the pool
	_, err = handler.Run(ctx, msgSwapFromTxIn.(*MsgSwap))
	c.Assert(err, IsNil)
	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)

	result, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)
	msgSwap := NewMsgSwap(GetRandomTx(), common.EmptyAsset, GetRandomDOGEAddress(), cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, 0, GetRandomBech32Addr())
	result, err = handler.Run(ctx, msgSwap)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)

	// When chain is halted , swap should respect it
	keeper.haltChain = 1
	result, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, NotNil)
	c.Assert(result, IsNil)
	keeper.haltChain = 0
}

func (s *HandlerSwapSuite) TestHandleStreamingSwap(c *C) {
	var err error
	ctx, mgr := setupManagerForTest(c)

	pool := NewPool()
	pool.Asset = common.DOGEAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	mgr.txOutStore = NewTxStoreDummy()

	na := GetRandomValidatorNode(NodeActive)
	c.Assert(mgr.Keeper().SetNodeAccount(ctx, na), IsNil)

	handler := NewSwapHandler(mgr)

	txID := GetRandomTxHash()
	signerDOGEAddr := GetRandomDOGEAddress()
	// no pool
	tx := common.NewTx(
		txID,
		signerDOGEAddr,
		signerDOGEAddr,
		common.Coins{
			common.NewCoin(common.SwitchNative, cosmos.NewUint(2_123400000)),
		},
		common.Gas{
			common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
		},
		fmt.Sprintf("=:DOGE.DOGE:%s", signerDOGEAddr),
	)
	msg := NewMsgSwap(tx, common.DOGEAsset, signerDOGEAddr, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 3, 5, na.NodeAddress)
	swp := msg.GetStreamingSwap()
	swp.Deposit = tx.Coins[0].Amount
	mgr.Keeper().SetStreamingSwap(ctx, swp)
	mgr.Keeper().SetMimir(ctx, constants.L1SlipMinBps.String(), 1)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)

	// ensure we don't add items into txout for streaming swaps. That is
	// handled in the swap queue manager
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 0)

	swp, err = mgr.Keeper().GetStreamingSwap(ctx, txID)
	c.Assert(err, IsNil)
	c.Check(swp.In.String(), Equals, "707800000")
	c.Check(swp.Out.String(), Equals, "617319586")

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	swp, err = mgr.Keeper().GetStreamingSwap(ctx, txID)
	c.Assert(err, IsNil)
	c.Check(swp.In.String(), Equals, "1415600000")
	c.Check(swp.Out.String(), Equals, "1165237487")

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, IsNil)
	swp, err = mgr.Keeper().GetStreamingSwap(ctx, txID)
	c.Assert(err, IsNil)
	c.Check(swp.In.String(), Equals, "2123400000")
	c.Check(swp.Out.String(), Equals, "1654583341")

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	_, err = handler.handle(ctx, *msg)
	c.Assert(err, NotNil)
	swp, err = mgr.Keeper().GetStreamingSwap(ctx, txID)
	c.Assert(err, IsNil)
	c.Check(swp.In.String(), Equals, "2123400000")
	c.Check(swp.Out.String(), Equals, "1654583341")
}

func (s *HandlerSwapSuite) TestSwapSynthERC20(c *C) {
	ctx, mgr := setupManagerForTest(c)
	mgr.txOutStore = NewTxStoreDummy()
	handler := NewSwapHandler(mgr)

	pool := NewPool()
	asset, err := common.NewAsset("ETH.AAVE-0X7FC66500C84A76AD7E9C93437BFC5AC33E2DDAE9")
	c.Assert(err, IsNil)
	pool.Asset = asset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(mgr.K.SetPool(ctx, pool), IsNil)

	m, err := ParseMemo(mgr.GetVersion(), "=:ETH/AAVE-0X7FC66:thor1x0jkvqdh2hlpeztd5zyyk70n3efx6mhudkmnn2::thor1a427q3v96psuj4fnughdw8glt5r7j38lj7rkp8:100")
	c.Assert(err, IsNil)
	swapM, ok := m.(SwapMemo)
	c.Assert(ok, Equals, true)
	swapM.Asset = fuzzyAssetMatch(ctx, mgr.K, swapM.Asset)
	txIn := NewObservedTx(
		common.NewTx("832B575FC2E92057BE1E1D69277B5AF690ADDF3E98E76FFC67232F846D87CB45", "0xd58610f89265a2fb637ac40edf59141ff873b266", "0x90f2b1ae50e6018230e90a33f98c7844a0ab635a",
			common.Coins{
				common.NewCoin(common.ETHAsset, cosmos.NewUint(20000000)),
			},
			common.Gas{
				common.NewCoin(common.ETHAsset, cosmos.NewUint(10000)),
			},
			"=:ETH/AAVE-0X7FC66:thor1x0jkvqdh2hlpeztd5zyyk70n3efx6mhudkmnn2::thor1a427q3v96psuj4fnughdw8glt5r7j38lj7rkp8:100",
		),
		1,
		GetRandomPubKey(), 1,
	)
	observerAddr, err := GetRandomTHORAddress().AccAddress()
	c.Assert(err, IsNil)
	msgSwapFromTxIn, err := getMsgSwapFromMemo(m.(SwapMemo), txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err := handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(res, IsNil)
	c.Assert(err, NotNil)
}

func (s *HandlerSwapSuite) TestDoubleSwap(c *C) {
	ctx, mgr := setupManagerForTest(c)
	keeper := &TestSwapHandleKeeper{
		pools:             make(map[common.Asset]Pool),
		activeNodeAccount: GetRandomValidatorNode(NodeActive),
		synthSupply:       cosmos.ZeroUint(),
	}
	mgr.txOutStore = NewTxStoreDummy()
	handler := NewSwapHandler(mgr)

	pool := NewPool()
	pool.Asset = common.DOGEAsset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)

	poolTCAN := NewPool()
	tCanAsset, err := common.NewAsset("DOGE.TCAN-014")
	c.Assert(err, IsNil)
	poolTCAN.Asset = tCanAsset
	poolTCAN.BalanceAsset = cosmos.NewUint(334850000)
	poolTCAN.BalanceRune = cosmos.NewUint(2349500000)
	c.Assert(mgr.Keeper().SetPool(ctx, poolTCAN), IsNil)

	signerDOGEAddr := GetRandomDOGEAddress()
	observerAddr := keeper.activeNodeAccount.NodeAddress

	// double swap - happy path
	m, err := ParseMemo(mgr.GetVersion(), "swap:DOGE.DOGE:"+signerDOGEAddr.String())
	c.Assert(err, IsNil)
	txIn := NewObservedTx(
		common.NewTx(GetRandomTxHash(), signerDOGEAddr, GetRandomDOGEAddress(),
			common.Coins{
				common.NewCoin(tCanAsset, cosmos.NewUint(20000000)),
			},
			common.Gas{
				common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
			},
			"swap:DOGE.DOGE:"+signerDOGEAddr.String(),
		),
		1,
		GetRandomPubKey(), 1,
	)
	msgSwapFromTxIn, err := getMsgSwapFromMemo(m.(SwapMemo), txIn, observerAddr)
	c.Assert(err, IsNil)

	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 0)

	_, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, IsNil)

	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
	// double swap , RUNE not enough to pay for transaction fee
	testnetDOGEAddr := GetRandomDOGEAddress()
	m1, err := ParseMemo(mgr.GetVersion(), "swap:DOGE.DOGE:"+testnetDOGEAddr.String())
	c.Assert(err, IsNil)
	txIn1 := NewObservedTx(
		common.NewTx(GetRandomTxHash(), signerDOGEAddr, GetRandomDOGEAddress(),
			common.Coins{
				common.NewCoin(tCanAsset, cosmos.NewUint(10000)),
			},
			common.Gas{
				common.NewCoin(common.DOGEAsset, cosmos.NewUint(10000)),
			},
			"swap:DOGE.DOGE:"+testnetDOGEAddr.String(),
		),
		1,
		GetRandomPubKey(), 1,
	)
	msgSwapFromTxIn1, err := getMsgSwapFromMemo(m1.(SwapMemo), txIn1, observerAddr)
	c.Assert(err, IsNil)
	mgr.TxOutStore().ClearOutboundItems(ctx)
	_, err = handler.Run(ctx, msgSwapFromTxIn1)
	c.Assert(err, IsNil)
	// This would actually error with ErrNotEnoughToPayFee from the txout manager,
	// but here mgr.txOutStore is a TxOutStoreDummy and without vaults set the happy path would also fail;
	// this test only checks the behaviour of the swap handler, not what happens after a TxOutItem is sent to the txout manager.

	items, err = mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
}

func (s *HandlerSwapSuite) TestSwapOutDexIntegration(c *C) {
	ctx, mgr := setupManagerForTest(c)
	mgr.txOutStore = NewTxStoreDummy()
	handler := NewSwapHandler(mgr)

	pool := NewPool()
	asset, err := common.NewAsset("ETH.ETH")
	c.Assert(err, IsNil)
	pool.Asset = asset
	pool.BalanceAsset = cosmos.NewUint(100 * common.One)
	pool.BalanceRune = cosmos.NewUint(100 * common.One)
	c.Assert(mgr.K.SetPool(ctx, pool), IsNil)

	c.Assert(mgr.K.SaveNetworkFee(ctx, common.ETHChain, NetworkFee{
		Chain:              common.ETHChain,
		TransactionSize:    10,
		TransactionFeeRate: 50_000,
	}), IsNil)

	swapMemo := "swap:ETH.ETH:" + types.GetRandomETHAddress().String() + "::::2f2386f3848:" + types.GetRandomETHAddress().String()
	m, err := ParseMemoWithTHORNames(ctx, mgr.Keeper(), swapMemo)
	c.Assert(err, IsNil)

	txIn := NewObservedTx(
		common.NewTx(GetRandomTxHash(), GetRandomTHORAddress(), GetRandomTHORAddress(),
			common.Coins{
				common.NewCoin(common.SwitchNative, cosmos.NewUint(2000000000)),
			},
			common.Gas{
				common.NewCoin(common.SwitchNative, cosmos.NewUint(20000000)),
			},
			swapMemo,
		),
		1,
		GetRandomPubKey(), 1,
	)

	observerAddr, err := GetRandomTHORAddress().AccAddress()
	c.Assert(err, IsNil)
	msgSwapFromTxIn, err := getMsgSwapFromMemo(m.(SwapMemo), txIn, observerAddr)
	c.Assert(err, IsNil)
	// when SwapOut Dex integration has been disabled by mimir , it should return an error cause refund
	mgr.Keeper().SetMimir(ctx, constants.SwapOutDexAggregationDisabled.String(), 1)
	res, err := handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(res, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "swap out dex integration disabled")

	mgr.Keeper().SetMimir(ctx, constants.SwapOutDexAggregationDisabled.String(), 0)

	// when target asset address is empty , swap should fail
	swapM, ok := m.(SwapMemo)
	c.Assert(ok, Equals, true)
	swapM.DexTargetAddress = ""
	msgSwapFromTxIn, err = getMsgSwapFromMemo(swapM, txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, NotNil)
	c.Assert(res, IsNil)
	c.Assert(errors.Is(err, se.ErrUnknownRequest), Equals, true)
	c.Assert(strings.HasPrefix(err.Error(), "aggregator target asset address is empty"), Equals, true)

	// When the target asset is not ETH.ETH, it should fail
	swapM, ok = m.(SwapMemo)
	c.Assert(ok, Equals, true)
	AAVEAsset, err := common.NewAsset("ETH.AAVE-0X7FC66500C84A76AD7E9C93437BFC5AC33E2DDAE9")
	c.Assert(err, IsNil)
	swapM.Asset = AAVEAsset
	msgSwapFromTxIn, err = getMsgSwapFromMemo(swapM, txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, NotNil)
	c.Assert(res, IsNil)
	c.Assert(err.Error(), Equals, "target asset (ETH.AAVE-0X7FC66500C84A76AD7E9C93437BFC5AC33E2DDAE9) is not gas asset , can't use dex feature")

	// when specified aggregator is not white list , swap should fail
	swapM, ok = m.(SwapMemo)
	c.Assert(ok, Equals, true)
	swapM.DexAggregator = "whatever"
	msgSwapFromTxIn, err = getMsgSwapFromMemo(swapM, txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, NotNil)
	c.Assert(res, IsNil)
	c.Assert(err.Error(), Equals, "whatever aggregator not found")

	// when aggregator target address is not valid , but we don't care
	swapM, ok = m.(SwapMemo)
	c.Assert(ok, Equals, true)
	swapM.DexTargetAddress = "whatever"
	msgSwapFromTxIn, err = getMsgSwapFromMemo(swapM, txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, IsNil)
	c.Assert(res, NotNil)

	// when aggregator target address and target chain doesn't match , don't care
	swapM, ok = m.(SwapMemo)
	c.Assert(ok, Equals, true)
	swapM.DexTargetAddress = GetRandomDOGEAddress().String()
	msgSwapFromTxIn, err = getMsgSwapFromMemo(swapM, txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, IsNil)
	c.Assert(res, NotNil)

	mgr.TxOutStore().ClearOutboundItems(ctx)
	// normal swap with DEX
	swapM, ok = m.(SwapMemo)
	c.Assert(ok, Equals, true)
	msgSwapFromTxIn, err = getMsgSwapFromMemo(swapM, txIn, observerAddr)
	c.Assert(err, IsNil)
	res, err = handler.Run(ctx, msgSwapFromTxIn)
	c.Assert(err, IsNil)
	c.Assert(res, NotNil)
	items, err := mgr.TxOutStore().GetOutboundItems(ctx)
	c.Assert(err, IsNil)
	c.Assert(items, HasLen, 1)
	c.Assert(items[0].Aggregator, Equals, "0x69800327b38A4CeF30367Dec3f64c2f2386f3848")
	c.Assert(items[0].AggregatorTargetAsset, Equals, swapM.DexTargetAddress)
	c.Assert(items[0].AggregatorTargetLimit, IsNil)
}
