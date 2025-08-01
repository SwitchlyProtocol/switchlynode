package thorchain

import (
	"errors"
	"fmt"

	. "gopkg.in/check.v1"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

type WithdrawSuite struct{}

var _ = Suite(&WithdrawSuite{})

var ethSingleTxFee = cosmos.NewUint(37500)

type WithdrawTestKeeper struct {
	keeper.KVStoreDummy
	store       map[string]interface{}
	networkFees map[common.Chain]NetworkFee
	keeper      keeper.Keeper
}

func NewWithdrawTestKeeper(keeper keeper.Keeper) *WithdrawTestKeeper {
	return &WithdrawTestKeeper{
		keeper:      keeper,
		store:       make(map[string]interface{}),
		networkFees: make(map[common.Chain]NetworkFee),
	}
}

// this one has an extra liquidity provider already set
func getWithdrawTestKeeper2(c *C, ctx cosmos.Context, k keeper.Keeper, runeAddress common.Address) keeper.Keeper {
	store := NewWithdrawTestKeeper(k)
	pool := Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.ETHAsset,
		LPUnits:      cosmos.NewUint(200 * common.One),
		SynthUnits:   cosmos.ZeroUint(),
		Status:       PoolAvailable,
	}
	c.Assert(store.SetPool(ctx, pool), IsNil)
	lp := LiquidityProvider{
		Asset:        pool.Asset,
		RuneAddress:  runeAddress,
		AssetAddress: runeAddress,
		Units:        cosmos.NewUint(100 * common.One),
		PendingRune:  cosmos.ZeroUint(),
	}
	store.SetLiquidityProvider(ctx, lp)
	return store
}

func (k *WithdrawTestKeeper) PoolExist(ctx cosmos.Context, asset common.Asset) bool {
	return !asset.Equals(common.Asset{Chain: common.ETHChain, Symbol: "NOTEXIST", Ticker: "NOTEXIST"})
}

func (k *WithdrawTestKeeper) GetPool(ctx cosmos.Context, asset common.Asset) (types.Pool, error) {
	if asset.Equals(common.Asset{Chain: common.ETHChain, Symbol: "NOTEXIST", Ticker: "NOTEXIST"}) {
		return types.Pool{}, nil
	}
	if val, ok := k.store[asset.String()]; ok {
		p, _ := val.(types.Pool)
		return p, nil
	}
	return types.Pool{
		BalanceRune:  cosmos.NewUint(100).MulUint64(common.One),
		BalanceAsset: cosmos.NewUint(100).MulUint64(common.One),
		LPUnits:      cosmos.NewUint(100).MulUint64(common.One),
		SynthUnits:   cosmos.ZeroUint(),
		Status:       PoolAvailable,
		Asset:        asset,
	}, nil
}

func (k *WithdrawTestKeeper) SetPool(ctx cosmos.Context, ps Pool) error {
	k.store[ps.Asset.String()] = ps
	return nil
}

func (k *WithdrawTestKeeper) GetGas(ctx cosmos.Context, asset common.Asset) ([]cosmos.Uint, error) {
	return []cosmos.Uint{cosmos.NewUint(37500), cosmos.NewUint(30000)}, nil
}

func (k *WithdrawTestKeeper) GetLiquidityProvider(ctx cosmos.Context, asset common.Asset, addr common.Address) (LiquidityProvider, error) {
	if asset.Equals(common.Asset{Chain: common.ETHChain, Symbol: "NOTEXISTSTICKER", Ticker: "NOTEXISTSTICKER"}) {
		return types.LiquidityProvider{}, errors.New("you asked for it")
	}
	if notExistLiquidityProviderAsset.Equals(asset) {
		return LiquidityProvider{}, errors.New("simulate error for test")
	}
	return k.keeper.GetLiquidityProvider(ctx, asset, addr)
}

func (k *WithdrawTestKeeper) GetNetworkFee(ctx cosmos.Context, chain common.Chain) (NetworkFee, error) {
	return k.networkFees[chain], nil
}

func (k *WithdrawTestKeeper) SaveNetworkFee(ctx cosmos.Context, chain common.Chain, networkFee NetworkFee) error {
	k.networkFees[chain] = networkFee
	return nil
}

func (k *WithdrawTestKeeper) SetLiquidityProvider(ctx cosmos.Context, lp LiquidityProvider) {
	k.keeper.SetLiquidityProvider(ctx, lp)
}

func (s *WithdrawSuite) SetUpSuite(c *C) {
	SetupConfigForTest()
}

// TestValidateWithdraw is to test validateWithdraw function
func (s WithdrawSuite) TestValidateWithdraw(c *C) {
	accountAddr := GetRandomValidatorNode(NodeWhiteListed).NodeAddress
	runeAddress, err := common.NewAddress("0x90f2b1ae50e6018230e90a33f98c7844a0ab635a")
	if err != nil {
		c.Error("fail to create new address")
	}
	inputs := []struct {
		name          string
		msg           MsgWithdrawLiquidity
		expectedError error
	}{
		{
			name: "empty-rune-address",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: "",
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			expectedError: errors.New("empty withdraw address"),
		},
		{
			name: "empty-withdraw-basis-points",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.ZeroUint(),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			expectedError: errors.New("withdraw basis points 0 is invalid"),
		},
		{
			name: "empty-request-txhash",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{},
				Signer:          accountAddr,
			},
			expectedError: errors.New("request tx hash is empty"),
		},
		{
			name: "empty-asset",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.Asset{},
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			expectedError: errors.New("empty asset"),
		},
		{
			name: "invalid-basis-point",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10001),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			expectedError: errors.New("withdraw basis points 10001 is invalid"),
		},
		{
			name: "invalid-pool-notexist",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.Asset{Chain: common.ETHChain, Ticker: "NOTEXIST", Symbol: "NOTEXIST"},
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			expectedError: errors.New("pool-ETH.NOTEXIST doesn't exist"),
		},
		{
			name: "all-good",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			expectedError: nil,
		},
	}

	for _, item := range inputs {
		ctx, _ := setupKeeperForTest(c)
		ps := &WithdrawTestKeeper{}
		c.Logf("name:%s", item.name)
		err := validateWithdraw(ctx, ps, item.msg)
		if item.expectedError != nil {
			c.Assert(err, NotNil)
			c.Assert(err.Error(), Equals, item.expectedError.Error())
			continue
		}
		c.Assert(err, IsNil)
	}
}

func (s WithdrawSuite) TestCalculateUnsake(c *C) {
	inputs := []struct {
		name                  string
		poolUnit              cosmos.Uint
		poolRune              cosmos.Uint
		poolAsset             cosmos.Uint
		lpUnit                cosmos.Uint
		percentage            cosmos.Uint
		expectedWithdrawRune  cosmos.Uint
		expectedWithdrawAsset cosmos.Uint
		expectedUnitLeft      cosmos.Uint
		expectedErr           error
	}{
		{
			name:                  "zero-poolunit",
			poolUnit:              cosmos.ZeroUint(),
			poolRune:              cosmos.ZeroUint(),
			poolAsset:             cosmos.ZeroUint(),
			lpUnit:                cosmos.ZeroUint(),
			percentage:            cosmos.ZeroUint(),
			expectedWithdrawRune:  cosmos.ZeroUint(),
			expectedWithdrawAsset: cosmos.ZeroUint(),
			expectedUnitLeft:      cosmos.ZeroUint(),
			expectedErr:           errors.New("poolUnits can't be zero"),
		},

		{
			name:                  "zero-poolrune",
			poolUnit:              cosmos.NewUint(500 * common.One),
			poolRune:              cosmos.ZeroUint(),
			poolAsset:             cosmos.ZeroUint(),
			lpUnit:                cosmos.ZeroUint(),
			percentage:            cosmos.ZeroUint(),
			expectedWithdrawRune:  cosmos.ZeroUint(),
			expectedWithdrawAsset: cosmos.ZeroUint(),
			expectedUnitLeft:      cosmos.ZeroUint(),
			expectedErr:           errors.New("pool rune balance can't be zero"),
		},

		{
			name:                  "zero-poolasset",
			poolUnit:              cosmos.NewUint(500 * common.One),
			poolRune:              cosmos.NewUint(500 * common.One),
			poolAsset:             cosmos.ZeroUint(),
			lpUnit:                cosmos.ZeroUint(),
			percentage:            cosmos.ZeroUint(),
			expectedWithdrawRune:  cosmos.ZeroUint(),
			expectedWithdrawAsset: cosmos.ZeroUint(),
			expectedUnitLeft:      cosmos.ZeroUint(),
			expectedErr:           errors.New("pool asset balance can't be zero"),
		},
		{
			name:                  "negative-liquidity-provider-unit",
			poolUnit:              cosmos.NewUint(500 * common.One),
			poolRune:              cosmos.NewUint(500 * common.One),
			poolAsset:             cosmos.NewUint(5100 * common.One),
			lpUnit:                cosmos.ZeroUint(),
			percentage:            cosmos.ZeroUint(),
			expectedWithdrawRune:  cosmos.ZeroUint(),
			expectedWithdrawAsset: cosmos.ZeroUint(),
			expectedUnitLeft:      cosmos.ZeroUint(),
			expectedErr:           errors.New("liquidity provider unit can't be zero"),
		},

		{
			name:                  "percentage-larger-than-100",
			poolUnit:              cosmos.NewUint(500 * common.One),
			poolRune:              cosmos.NewUint(500 * common.One),
			poolAsset:             cosmos.NewUint(500 * common.One),
			lpUnit:                cosmos.NewUint(100 * common.One),
			percentage:            cosmos.NewUint(12000),
			expectedWithdrawRune:  cosmos.ZeroUint(),
			expectedWithdrawAsset: cosmos.ZeroUint(),
			expectedUnitLeft:      cosmos.ZeroUint(),
			expectedErr:           fmt.Errorf("withdraw basis point %s is not valid", cosmos.NewUint(12000)),
		},
		{
			name:                  "withdraw-1",
			poolUnit:              cosmos.NewUint(700 * common.One),
			poolRune:              cosmos.NewUint(700 * common.One),
			poolAsset:             cosmos.NewUint(700 * common.One),
			lpUnit:                cosmos.NewUint(200 * common.One),
			percentage:            cosmos.NewUint(10000),
			expectedUnitLeft:      cosmos.ZeroUint(),
			expectedWithdrawAsset: cosmos.NewUint(200 * common.One),
			expectedWithdrawRune:  cosmos.NewUint(200 * common.One),
			expectedErr:           nil,
		},
		{
			name:                  "withdraw-2",
			poolUnit:              cosmos.NewUint(100),
			poolRune:              cosmos.NewUint(15 * common.One),
			poolAsset:             cosmos.NewUint(155 * common.One),
			lpUnit:                cosmos.NewUint(100),
			percentage:            cosmos.NewUint(1000),
			expectedUnitLeft:      cosmos.NewUint(90),
			expectedWithdrawAsset: cosmos.NewUint(1550000000),
			expectedWithdrawRune:  cosmos.NewUint(150000000),
			expectedErr:           nil,
		},
	}

	ctx, keeper := setupKeeperForTest(c)
	for _, item := range inputs {
		c.Logf("name:%s", item.name)
		withDrawRune, withDrawAsset, unitAfter, err := calculateWithdraw(ctx, keeper, common.EmptyAsset, item.poolUnit, item.poolRune, item.poolAsset, item.lpUnit, item.percentage, common.EmptyAsset, common.NoAddress)
		// Empty context, manager, poolAsset, withdrawAddress due to only being needed for (non-POL) asymmetric withdraws (and this being symmetric).
		if item.expectedErr == nil {
			c.Assert(err, IsNil)
		} else {
			c.Assert(err.Error(), Equals, item.expectedErr.Error())
		}
		c.Logf("expected rune:%s,rune:%s", item.expectedWithdrawRune, withDrawRune)
		c.Check(item.expectedWithdrawRune.Uint64(), Equals, withDrawRune.Uint64(), Commentf("Expected %d, got %d", item.expectedWithdrawRune.Uint64(), withDrawRune.Uint64()))
		c.Check(item.expectedWithdrawAsset.Uint64(), Equals, withDrawAsset.Uint64(), Commentf("Expected %d, got %d", item.expectedWithdrawAsset.Uint64(), withDrawAsset.Uint64()))
		c.Check(item.expectedUnitLeft.Uint64(), Equals, unitAfter.Uint64())
	}
}

func (WithdrawSuite) TestWithdraw(c *C) {
	ctx, mgr := setupManagerForTest(c)
	accountAddr := GetRandomValidatorNode(NodeWhiteListed).NodeAddress
	runeAddress := GetRandomRUNEAddress()
	ps := NewWithdrawTestKeeper(mgr.Keeper())
	ps2 := getWithdrawTestKeeper(c, ctx, mgr.Keeper(), runeAddress)

	remainGas := uint64(56250)
	testCases := []struct {
		name          string
		msg           MsgWithdrawLiquidity
		ps            keeper.Keeper
		runeAmount    cosmos.Uint
		assetAmount   cosmos.Uint
		expectedError error
	}{
		{
			name: "empty-rune-address",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: "",
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("empty withdraw address"),
		},
		{
			name: "empty-request-txhash",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("request tx hash is empty"),
		},
		{
			name: "empty-asset",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.Asset{},
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("empty asset"),
		},
		{
			name: "invalid-basis-point",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10001),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("withdraw basis points 10001 is invalid"),
		},
		{
			name: "invalid-pool-notexist",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.Asset{Chain: common.ETHChain, Ticker: "NOTEXIST", Symbol: "NOTEXIST"},
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("pool-ETH.NOTEXIST doesn't exist"),
		},
		{
			name: "invalid-pool-liquidity-provider-notexist",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.Asset{Chain: common.ETHChain, Ticker: "NOTEXISTSTICKER", Symbol: "NOTEXISTSTICKER"},
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("you asked for it"),
		},
		{
			name: "nothing-to-withdraw",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.ZeroUint(),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps,
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: errors.New("withdraw basis points 0 is invalid"),
		},
		{
			name: "all-good-half",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(5000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps2,
			runeAmount:    cosmos.NewUint(50 * common.One),
			assetAmount:   cosmos.NewUint(50 * common.One),
			expectedError: nil,
		},
		{
			name: "all-good",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				Signer:          accountAddr,
			},
			ps:            ps2,
			runeAmount:    cosmos.NewUint(50 * common.One),
			assetAmount:   cosmos.NewUint(50 * common.One).Sub(cosmos.NewUint(remainGas)),
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		c.Logf("name:%s", tc.name)
		mgr.K = tc.ps
		c.Assert(tc.ps.SaveNetworkFee(ctx, common.ETHChain, NetworkFee{
			Chain:              common.ETHChain,
			TransactionSize:    1,
			TransactionFeeRate: ethSingleTxFee.Uint64(),
		}), IsNil)
		r, asset, _, _, err := withdraw(ctx, tc.msg, mgr)
		if tc.expectedError != nil {
			c.Assert(err, NotNil)
			c.Check(err.Error(), Equals, tc.expectedError.Error())
			c.Check(r.Uint64(), Equals, tc.runeAmount.Uint64())
			c.Check(asset.Uint64(), Equals, tc.assetAmount.Uint64())
			continue
		}
		c.Assert(err, IsNil)
		c.Assert(r.Uint64(), Equals, tc.runeAmount.Uint64(), Commentf("%d != %d", r.Uint64(), tc.runeAmount.Uint64()))
		c.Assert(asset.Equal(tc.assetAmount), Equals, true, Commentf("expect:%s, however got:%s", tc.assetAmount.String(), asset.String()))
	}
}

func (WithdrawSuite) TestWithdrawAsym(c *C) {
	accountAddr := GetRandomValidatorNode(NodeWhiteListed).NodeAddress
	runeAddress := GetRandomRUNEAddress()

	testCases := []struct {
		name          string
		msg           MsgWithdrawLiquidity
		runeAmount    cosmos.Uint
		assetAmount   cosmos.Uint
		expectedError error
	}{
		{
			name: "all-good-asymmetric-rune",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				WithdrawalAsset: common.SwitchNative,
				Signer:          accountAddr,
			},
			runeAmount:    cosmos.NewUint(6250000000),
			assetAmount:   cosmos.ZeroUint(),
			expectedError: nil,
		},
		{
			name: "all-good-asymmetric-asset",
			msg: MsgWithdrawLiquidity{
				WithdrawAddress: runeAddress,
				BasisPoints:     cosmos.NewUint(10000),
				Asset:           common.ETHAsset,
				Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
				WithdrawalAsset: common.ETHAsset,
				Signer:          accountAddr,
			},
			runeAmount:    cosmos.ZeroUint(),
			assetAmount:   cosmos.NewUint(6250000000),
			expectedError: nil,
		},
	}
	for _, tc := range testCases {
		c.Logf("name:%s", tc.name)
		ctx, mgr := setupManagerForTest(c)
		ps := getWithdrawTestKeeper2(c, ctx, mgr.Keeper(), runeAddress)
		mgr.K = ps
		c.Assert(ps.SaveNetworkFee(ctx, common.ETHChain, NetworkFee{
			Chain:              common.ETHChain,
			TransactionSize:    1,
			TransactionFeeRate: ethSingleTxFee.Uint64(),
		}), IsNil)
		r, asset, _, _, err := withdraw(ctx, tc.msg, mgr)
		if tc.expectedError != nil {
			c.Assert(err, NotNil)
			c.Check(err.Error(), Equals, tc.expectedError.Error())
			c.Check(r.Uint64(), Equals, tc.runeAmount.Uint64())
			c.Check(asset.Uint64(), Equals, tc.assetAmount.Uint64())
			continue
		}
		c.Assert(err, IsNil)
		c.Assert(r.Uint64(), Equals, tc.runeAmount.Uint64(), Commentf("%d != %d", r.Uint64(), tc.runeAmount.Uint64()))
		c.Assert(asset.Equal(tc.assetAmount), Equals, true, Commentf("expect:%s, however got:%s", tc.assetAmount.String(), asset.String()))
	}
}

func (WithdrawSuite) TestWithdrawPendingRuneOrAsset(c *C) {
	accountAddr := GetRandomValidatorNode(NodeActive).NodeAddress
	ctx, mgr := setupManagerForTest(c)
	pool := Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.ETHAsset,
		LPUnits:      cosmos.NewUint(200 * common.One),
		Status:       PoolAvailable,
	}
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	lp := LiquidityProvider{
		Asset:              common.ETHAsset,
		RuneAddress:        GetRandomRUNEAddress(),
		AssetAddress:       GetRandomETHAddress(),
		LastAddHeight:      1024,
		LastWithdrawHeight: 0,
		Units:              cosmos.ZeroUint(),
		PendingRune:        cosmos.NewUint(1024),
		PendingAsset:       cosmos.ZeroUint(),
		PendingTxID:        GetRandomTxHash(),
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp)
	msg := MsgWithdrawLiquidity{
		WithdrawAddress: lp.RuneAddress,
		BasisPoints:     cosmos.NewUint(10000),
		Asset:           common.ETHAsset,
		Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
		WithdrawalAsset: common.ETHAsset,
		Signer:          accountAddr,
	}
	runeAmt, assetAmt, unitsLeft, gas, err := withdraw(ctx, msg, mgr)
	c.Assert(err, IsNil)
	c.Assert(runeAmt.Equal(cosmos.NewUint(1024)), Equals, true)
	c.Assert(assetAmt.IsZero(), Equals, true)
	c.Assert(unitsLeft.IsZero(), Equals, true)
	c.Assert(gas.IsZero(), Equals, true)

	lp1 := LiquidityProvider{
		Asset:              common.ETHAsset,
		RuneAddress:        GetRandomRUNEAddress(),
		AssetAddress:       GetRandomETHAddress(),
		LastAddHeight:      1024,
		LastWithdrawHeight: 0,
		Units:              cosmos.ZeroUint(),
		PendingRune:        cosmos.ZeroUint(),
		PendingAsset:       cosmos.NewUint(1024),
		PendingTxID:        GetRandomTxHash(),
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp1)
	msg1 := MsgWithdrawLiquidity{
		WithdrawAddress: lp1.RuneAddress,
		BasisPoints:     cosmos.NewUint(10000),
		Asset:           common.ETHAsset,
		Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
		WithdrawalAsset: common.ETHAsset,
		Signer:          accountAddr,
	}
	runeAmt, assetAmt, unitsLeft, gas, err = withdraw(ctx, msg1, mgr)
	c.Assert(err, IsNil)
	c.Assert(assetAmt.Equal(cosmos.NewUint(1024)), Equals, true)
	c.Assert(runeAmt.IsZero(), Equals, true)
	c.Assert(unitsLeft.IsZero(), Equals, true)
	c.Assert(gas.IsZero(), Equals, true)
}

func (s *WithdrawSuite) TestWithdrawPendingLiquidityShouldRoundToPoolDecimals(c *C) {
	accountAddr := GetRandomValidatorNode(NodeActive).NodeAddress
	ctx, mgr := setupManagerForTest(c)
	pool := Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.ETHAsset,
		LPUnits:      cosmos.NewUint(200 * common.One),
		Status:       PoolAvailable,
		Decimals:     int64(6),
	}
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	v := GetCurrentVersion()
	constantAccessor := constants.GetConstantValues(v)
	addHandler := NewAddLiquidityHandler(mgr)
	// create a LP record that has pending asset
	lpAddr := GetRandomTHORAddress()
	c.Assert(addHandler.addLiquidity(ctx,
		common.ETHAsset,
		cosmos.ZeroUint(),
		cosmos.NewUint(339448125567),
		lpAddr,
		GetRandomBTCAddress(),
		GetRandomTxHash(),
		true,
		constantAccessor), IsNil)

	newctx := ctx.WithBlockHeight(ctx.BlockHeight() + 17280*2)
	msg2 := MsgWithdrawLiquidity{
		WithdrawAddress: lpAddr,
		BasisPoints:     cosmos.NewUint(10000),
		Asset:           common.ETHAsset,
		Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
		WithdrawalAsset: common.ETHAsset,
		Signer:          accountAddr,
	}
	runeAmt, assetAmt, unitsClaimed, _, err := withdraw(newctx, msg2, mgr)
	c.Assert(err, IsNil)
	c.Assert(assetAmt.Equal(cosmos.NewUint(339448125500)), Equals, true, Commentf("%d", assetAmt.Uint64()))
	c.Assert(runeAmt.IsZero(), Equals, true)
	c.Assert(unitsClaimed.IsZero(), Equals, true)
}

func getWithdrawTestKeeper(c *C, ctx cosmos.Context, k keeper.Keeper, runeAddress common.Address) keeper.Keeper {
	store := NewWithdrawTestKeeper(k)
	pool := Pool{
		BalanceRune:  cosmos.NewUint(100 * common.One),
		BalanceAsset: cosmos.NewUint(100 * common.One),
		Asset:        common.ETHAsset,
		LPUnits:      cosmos.NewUint(100 * common.One),
		SynthUnits:   cosmos.ZeroUint(),
		Status:       PoolAvailable,
	}
	c.Assert(store.SetPool(ctx, pool), IsNil)
	lp := LiquidityProvider{
		Asset:              pool.Asset,
		RuneAddress:        runeAddress,
		AssetAddress:       runeAddress,
		LastAddHeight:      0,
		LastWithdrawHeight: 0,
		Units:              cosmos.NewUint(100 * common.One),
		PendingRune:        cosmos.ZeroUint(),
		PendingAsset:       cosmos.ZeroUint(),
		PendingTxID:        "",
		RuneDepositValue:   cosmos.NewUint(100 * common.One),
		AssetDepositValue:  cosmos.NewUint(100 * common.One),
	}
	store.SetLiquidityProvider(ctx, lp)
	return store
}

func (WithdrawSuite) TestWithdrawSynth(c *C) {
	accountAddr := GetRandomValidatorNode(NodeActive).NodeAddress
	ctx, mgr := setupManagerForTest(c)
	asset := common.BTCAsset.GetSyntheticAsset()

	coin := common.NewCoin(asset, cosmos.NewUint(100*common.One))
	c.Assert(mgr.Keeper().MintToModule(ctx, ModuleName, coin), IsNil)
	c.Assert(mgr.Keeper().SendFromModuleToModule(ctx, ModuleName, AsgardName, common.NewCoins(coin)), IsNil)

	pool := Pool{
		BalanceRune:  cosmos.ZeroUint(),
		BalanceAsset: coin.Amount,
		Asset:        asset,
		LPUnits:      cosmos.NewUint(200 * common.One),
		Status:       PoolAvailable,
	}
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	lp := LiquidityProvider{
		Asset:              asset,
		RuneAddress:        common.NoAddress,
		AssetAddress:       GetRandomRUNEAddress(),
		LastAddHeight:      0,
		LastWithdrawHeight: 0,
		Units:              cosmos.NewUint(100 * common.One),
		PendingRune:        cosmos.ZeroUint(),
		PendingAsset:       cosmos.ZeroUint(),
		PendingTxID:        GetRandomTxHash(),
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp)
	msg := MsgWithdrawLiquidity{
		WithdrawAddress: lp.AssetAddress,
		BasisPoints:     cosmos.NewUint(MaxWithdrawBasisPoints / 2),
		Asset:           asset,
		Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
		WithdrawalAsset: common.EmptyAsset,
		Signer:          accountAddr,
	}
	runeAmt, assetAmt, unitsLeft, gas, err := withdraw(ctx, msg, mgr)
	c.Assert(err, IsNil)
	c.Check(assetAmt.Uint64(), Equals, uint64(25*common.One), Commentf("%d", assetAmt.Uint64()))
	c.Check(runeAmt.IsZero(), Equals, true)
	c.Check(unitsLeft.Uint64(), Equals, uint64(50*common.One), Commentf("%d", unitsLeft.Uint64()))
	c.Check(gas.IsZero(), Equals, true)

	pool, err = mgr.Keeper().GetPool(ctx, asset)
	c.Assert(err, IsNil)
	c.Check(pool.BalanceRune.Uint64(), Equals, uint64(0), Commentf("%d", pool.BalanceRune.Uint64()))
	c.Check(pool.BalanceAsset.Uint64(), Equals, uint64(75*common.One), Commentf("%d", pool.BalanceAsset.Uint64()))
	c.Check(pool.LPUnits.Uint64(), Equals, uint64(150*common.One), Commentf("%d", pool.LPUnits.Uint64())) // LP units did decreased
}

func (WithdrawSuite) TestWithdrawSynthSingleLP(c *C) {
	accountAddr := GetRandomValidatorNode(NodeActive).NodeAddress
	ctx, mgr := setupManagerForTest(c)
	asset := common.BTCAsset.GetSyntheticAsset()

	coin := common.NewCoin(asset, cosmos.NewUint(30*common.One))
	c.Assert(mgr.Keeper().MintToModule(ctx, ModuleName, coin), IsNil)
	c.Assert(mgr.Keeper().SendFromModuleToModule(ctx, ModuleName, AsgardName, common.NewCoins(coin)), IsNil)

	pool := Pool{
		BalanceRune:  cosmos.ZeroUint(),
		BalanceAsset: coin.Amount,
		Asset:        asset,
		LPUnits:      cosmos.NewUint(200 * common.One),
		Status:       PoolAvailable,
	}
	c.Assert(mgr.Keeper().SetPool(ctx, pool), IsNil)
	lp := LiquidityProvider{
		Asset:              asset,
		RuneAddress:        common.NoAddress,
		AssetAddress:       GetRandomRUNEAddress(),
		LastAddHeight:      0,
		LastWithdrawHeight: 0,
		Units:              cosmos.NewUint(200 * common.One),
		PendingRune:        cosmos.ZeroUint(),
		PendingAsset:       cosmos.ZeroUint(),
		PendingTxID:        GetRandomTxHash(),
	}
	mgr.Keeper().SetLiquidityProvider(ctx, lp)
	msg := MsgWithdrawLiquidity{
		WithdrawAddress: lp.AssetAddress,
		BasisPoints:     cosmos.NewUint(MaxWithdrawBasisPoints),
		Asset:           asset,
		Tx:              common.Tx{ID: "28B40BF105A112389A339A64BD1A042E6140DC9082C679586C6CF493A9FDE3FE"},
		WithdrawalAsset: common.EmptyAsset,
		Signer:          accountAddr,
	}
	runeAmt, assetAmt, unitsLeft, gas, err := withdraw(ctx, msg, mgr)
	c.Assert(err, IsNil)
	c.Check(assetAmt.Uint64(), Equals, coin.Amount.Uint64(), Commentf("%d", assetAmt.Uint64()))
	c.Check(runeAmt.IsZero(), Equals, true)
	c.Check(unitsLeft.Uint64(), Equals, uint64(200*common.One), Commentf("%d", unitsLeft.Uint64()))
	c.Check(gas.IsZero(), Equals, true)

	pool, err = mgr.Keeper().GetPool(ctx, asset)
	c.Check(err, IsNil)
	c.Check(pool.BalanceRune.Uint64(), Equals, uint64(0), Commentf("%d", pool.BalanceRune.Uint64()))
	c.Check(pool.BalanceAsset.Uint64(), Equals, uint64(0), Commentf("%d", pool.BalanceAsset.Uint64()))
	c.Check(pool.LPUnits.Uint64(), Equals, uint64(0), Commentf("%d", pool.LPUnits.Uint64())) // LP units did decreased
}
