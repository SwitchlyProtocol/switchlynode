package thorchain

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

type SwapperVCUR struct{}

func newSwapperVCUR() *SwapperVCUR {
	return &SwapperVCUR{}
}

// validateMessage is trying to validate the legitimacy of the incoming message and decide whether THORNode can handle it
func (s *SwapperVCUR) validateMessage(tx common.Tx, target common.Asset, destination common.Address) error {
	if err := tx.Valid(); err != nil {
		return err
	}
	if target.IsEmpty() {
		return errors.New("target is empty")
	}
	if destination.IsEmpty() {
		return errors.New("destination is empty")
	}
	if tx.Coins[0].Asset.IsTradeAsset() && !target.IsTradeAsset() && !target.IsSwitch() {
		return errors.New("swaps from trade asset to L1 incur slip, use trade-")
	}
	if target.IsTradeAsset() && !tx.Coins[0].Asset.IsTradeAsset() && !tx.Coins[0].IsSwitch() {
		return errors.New("swaps from L1 to trade asset incur slip, use trade+")
	}

	return nil
}

func (s *SwapperVCUR) Swap(ctx cosmos.Context,
	keeper keeper.Keeper,
	tx common.Tx,
	target common.Asset,
	destination common.Address,
	swapTarget cosmos.Uint,
	dexAgg string,
	dexAggTargetAsset string,
	dexAggLimit *cosmos.Uint,
	swp StreamingSwap,
	synthVirtualDepthMult int64, mgr Manager,
) (cosmos.Uint, []*EventSwap, error) {
	var swapEvents []*EventSwap

	if err := s.validateMessage(tx, target, destination); err != nil {
		return cosmos.ZeroUint(), swapEvents, err
	}
	source := tx.Coins[0].Asset

	if source.IsSyntheticAsset() {
		burnHeight := mgr.Keeper().GetConfigInt64(ctx, constants.BurnSynths)
		if burnHeight > 0 && ctx.BlockHeight() > burnHeight {
			return cosmos.ZeroUint(), swapEvents, fmt.Errorf("burning synthetics has been disabled")
		}
	}
	if target.IsSyntheticAsset() {
		mintHeight := mgr.Keeper().GetConfigInt64(ctx, constants.MintSynths)
		if mintHeight > 0 && ctx.BlockHeight() > mintHeight {
			return cosmos.ZeroUint(), swapEvents, fmt.Errorf("minting synthetics has been disabled")
		}
	}

	if !destination.IsNoop() && !destination.IsChain(target.GetChain()) {
		return cosmos.ZeroUint(), swapEvents, fmt.Errorf("destination address is not a valid %s address", target.GetChain())
	}
	if source.Equals(target) {
		return cosmos.ZeroUint(), swapEvents, fmt.Errorf("cannot swap from %s --> %s, assets match", source, target)
	}

	isDoubleSwap := !source.IsSwitch() && !target.IsSwitch()
	if isDoubleSwap {
		var swapErr error
		var swapEvt *EventSwap
		var amt cosmos.Uint
		// Here we use a swapTarget of 0 because the target is for the next swap asset in a double swap
		amt, swapEvt, swapErr = s.swapOne(ctx, mgr, tx, common.SwitchNative, destination, cosmos.ZeroUint(), synthVirtualDepthMult)
		if swapErr != nil {
			return cosmos.ZeroUint(), swapEvents, swapErr
		}
		tx.Coins = common.Coins{common.NewCoin(common.SwitchNative, amt)}
		tx.Gas = nil
		swapEvents = append(swapEvents, swapEvt)
	}
	assetAmount, swapEvt, swapErr := s.swapOne(ctx, mgr, tx, target, destination, swapTarget, synthVirtualDepthMult)
	if swapErr != nil {
		return cosmos.ZeroUint(), swapEvents, swapErr
	}
	swapEvents = append(swapEvents, swapEvt)
	if !swapTarget.IsZero() && assetAmount.LT(swapTarget) {
		// **NOTE** this error string is utilized by the adv swap queue manager to
		// catch the error. DO NOT change this error string without updating
		// the adv swap queue manager as well
		return cosmos.ZeroUint(), swapEvents, fmt.Errorf("emit asset %s less than price limit %s", assetAmount, swapTarget)
	}
	// emit asset is zero
	if assetAmount.IsZero() {
		return cosmos.ZeroUint(), swapEvents, errors.New("zero emit asset")
	}

	// Thanks to CacheContext, the swap event can be emitted before handling outbounds,
	// since if there's a later error the event emission will not take place.
	for _, evt := range swapEvents {
		if swp.Quantity > evt.StreamingSwapQuantity {
			evt.StreamingSwapQuantity = swp.Quantity
			evt.StreamingSwapCount = swp.Count + 1 // first swap count is "zero"
		} else {
			evt.StreamingSwapQuantity = 1
			evt.StreamingSwapCount = 1
		}
		if err := mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
			ctx.Logger().Error("fail to emit swap event", "error", err)
		}
		telemetry.IncrCounterWithLabels(
			[]string{"thornode", "swap", "count"},
			float32(1),
			[]metrics.Label{telemetry.NewLabel("pool", evt.Pool.String())},
		)
		telemetry.IncrCounterWithLabels(
			[]string{"thornode", "swap", "slip"},
			telem(evt.SwapSlip),
			[]metrics.Label{telemetry.NewLabel("pool", evt.Pool.String())},
		)
		telemetry.IncrCounterWithLabels(
			[]string{"thornode", "swap", "liquidity_fee"},
			telem(evt.LiquidityFeeInRune),
			[]metrics.Label{telemetry.NewLabel("pool", evt.Pool.String())},
		)
	}

	if !destination.IsNoop() {
		toi := TxOutItem{
			Chain:                 target.GetChain(),
			InHash:                tx.ID,
			ToAddress:             destination,
			Coin:                  common.NewCoin(target, assetAmount),
			Aggregator:            dexAgg,
			AggregatorTargetAsset: dexAggTargetAsset,
			AggregatorTargetLimit: dexAggLimit,
		}

		// streaming swap outbounds are handled in the swap queue manager
		if swp.Valid() != nil {
			ok, err := mgr.TxOutStore().TryAddTxOutItem(ctx, mgr, toi, swapTarget)
			if err != nil {
				return assetAmount, swapEvents, ErrInternal(err, "fail to add outbound tx")
			}
			if !ok {
				return assetAmount, swapEvents, errFailAddOutboundTx
			}
		}
	}

	return assetAmount, swapEvents, nil
}

func (s *SwapperVCUR) swapOne(ctx cosmos.Context,
	mgr Manager, tx common.Tx,
	target common.Asset,
	destination common.Address,
	swapTarget cosmos.Uint,
	synthVirtualDepthMult int64,
) (amt cosmos.Uint, evt *EventSwap, swapErr error) {
	tradeAccountsEnabled := mgr.Keeper().GetConfigInt64(ctx, constants.TradeAccountsEnabled)
	tradeAccountsDepositEnabled := mgr.Keeper().GetConfigInt64(ctx, constants.TradeAccountsDepositEnabled)

	source := tx.Coins[0].Asset
	amount := tx.Coins[0].Amount

	ctx.Logger().Info("swapping", "from", tx.FromAddress, "coins", tx.Coins[0], "target", target, "to", destination)

	// Set asset to our pool asset
	var minSlipAsset common.Asset
	if source.IsSwitch() {
		minSlipAsset = target
	} else {
		minSlipAsset = source
	}
	poolAsset := minSlipAsset.GetLayer1Asset()

	if source.IsTradeAsset() {
		if tradeAccountsEnabled <= 0 {
			return cosmos.ZeroUint(), evt, fmt.Errorf("trade accounts are disabled")
		}
		fromAcc, err := cosmos.AccAddressFromBech32(tx.FromAddress.String())
		if err != nil {
			return cosmos.ZeroUint(), evt, ErrInternal(err, "fail to parse from address")
		}
		amount, err = mgr.TradeAccountManager().Withdrawal(ctx, source, amount, fromAcc, common.NoAddress, tx.ID)
		if err != nil {
			return cosmos.ZeroUint(), evt, ErrInternal(err, "fail to withdraw from trade")
		}
	}

	if source.IsSecuredAsset() {
		fromAcc, err := cosmos.AccAddressFromBech32(tx.FromAddress.String())
		if err != nil {
			return cosmos.ZeroUint(), evt, ErrInternal(err, "fail to parse from address")
		}
		withdrawAmount, err := mgr.SecuredAssetManager().Withdraw(ctx, source, amount, fromAcc, common.NoAddress, tx.ID)
		if err != nil {
			return cosmos.ZeroUint(), evt, ErrInternal(err, "fail to withdraw from secured asset")
		}
		amount = withdrawAmount.Amount
	}

	if target.IsTradeAsset() && tradeAccountsDepositEnabled <= 0 {
		return cosmos.ZeroUint(), evt, fmt.Errorf("trade accounts deposits are disabled")
	}

	swapEvt := NewEventSwap(
		poolAsset,
		swapTarget,
		cosmos.ZeroUint(),
		cosmos.ZeroUint(),
		cosmos.ZeroUint(),
		tx,
		common.NoCoin,
		cosmos.ZeroUint(),
	)

	// Update swap event input with source and amount details,
	// notably for if the Trade/Secured amount withdrawn is less than the transaction-specified amount.
	// For streaming swaps, InTx already only represents the sub-swap amount, not the original inbound.
	swapEvt.InTx.Coins = common.NewCoins(common.NewCoin(source, amount))

	if poolAsset.IsDerivedAsset() {
		// regenerate derived virtual pool
		mgr.NetworkMgr().SpawnDerivedAsset(ctx, poolAsset, mgr)
	}

	// Check if pool exists
	keeper := mgr.Keeper()
	if !keeper.PoolExist(ctx, poolAsset) {
		err := fmt.Errorf("pool %s doesn't exist", poolAsset)
		return cosmos.ZeroUint(), evt, err
	}

	pool, err := keeper.GetPool(ctx, poolAsset)
	if err != nil {
		return cosmos.ZeroUint(), evt, ErrInternal(err, fmt.Sprintf("fail to get pool(%s)", poolAsset))
	}
	// sanity check: ensure we're never swapping with the vault
	// (technically is actually the yield bearing synth vault)
	if pool.Asset.IsSyntheticAsset() {
		return cosmos.ZeroUint(), evt, ErrInternal(err, fmt.Sprintf("dev error: swapping with a vault(%s) is not allowed", pool.Asset))
	}
	synthSupply := keeper.GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
	pool.CalcUnits(synthSupply)

	// pool must be available unless source is synthetic
	// synths may be redeemed regardless of pool status
	if !source.IsSyntheticAsset() && !pool.IsAvailable() {
		return cosmos.ZeroUint(), evt, fmt.Errorf("pool(%s) is not available", pool.Asset)
	}

	// Get our X, x, Y values
	var X, Y cosmos.Uint
	if source.IsSwitch() {
		X = pool.BalanceRune
		Y = pool.BalanceAsset
	} else {
		Y = pool.BalanceRune
		X = pool.BalanceAsset
	}
	x := amount

	// give virtual pool depth if we're swapping with a synthetic asset
	if source.IsSyntheticAsset() || target.IsSyntheticAsset() {
		X = common.GetUncappedShare(cosmos.NewUint(uint64(synthVirtualDepthMult)), cosmos.NewUint(10_000), X)
		Y = common.GetUncappedShare(cosmos.NewUint(uint64(synthVirtualDepthMult)), cosmos.NewUint(10_000), Y)
	}

	// check our X,x,Y values are valid
	if x.IsZero() {
		return cosmos.ZeroUint(), evt, errSwapFailInvalidAmount
	}
	if X.IsZero() || Y.IsZero() {
		return cosmos.ZeroUint(), evt, errSwapFailInvalidBalance
	}

	swapSlipBps := s.CalcSwapSlip(X, x)
	swapEvt.PoolSlip = swapSlipBps

	// the non-RUNE asset is used to determine which min slip to use
	minSlipBps := getMinSlipBps(ctx, mgr.Keeper(), minSlipAsset)

	var (
		emitAssets   cosmos.Uint
		liquidityFee cosmos.Uint
	)
	emitAssets, liquidityFee, swapEvt.SwapSlip = s.GetSwapCalc(X, x, Y, swapSlipBps, minSlipBps)
	emitAssets = cosmos.RoundToDecimal(emitAssets, pool.Decimals)
	swapEvt.EmitAsset = common.NewCoin(target, emitAssets)
	swapEvt.LiquidityFee = liquidityFee

	// do THORNode have enough balance to swap?
	if emitAssets.GTE(Y) {
		return cosmos.ZeroUint(), evt, errSwapFailNotEnoughBalance
	}

	ctx.Logger().Info("pre swap", "pool", pool.Asset, "rune", pool.BalanceRune, "asset", pool.BalanceAsset, "lp units", pool.LPUnits, "synth units", pool.SynthUnits)

	// Burning of input synth or derived pool input (Asset or RUNE).
	if source.IsSyntheticAsset() || pool.Asset.IsDerivedAsset() {
		burnCoin := tx.Coins[0]
		if err := mgr.Keeper().SendFromModuleToModule(ctx, AsgardName, ModuleName, common.NewCoins(burnCoin)); err != nil {
			ctx.Logger().Error("fail to move coins during swap", "error", err)
			return cosmos.ZeroUint(), evt, err
		} else if err := mgr.Keeper().BurnFromModule(ctx, ModuleName, burnCoin); err != nil {
			ctx.Logger().Error("fail to burn coins during swap", "error", err)
		} else {
			burnEvt := NewEventMintBurn(BurnSupplyType, burnCoin.Asset.Native(), burnCoin.Amount, "swap")
			if err := mgr.EventMgr().EmitEvent(ctx, burnEvt); err != nil {
				ctx.Logger().Error("fail to emit burn event", "error", err)
			}
		}
	}

	// Minting of output synth or derived pool output (Asset or RUNE).
	if (target.IsSyntheticAsset() || pool.Asset.IsDerivedAsset()) &&
		!emitAssets.IsZero() {
		// If the source isn't RUNE, the target should be RUNE.
		mintCoin := common.NewCoin(target, emitAssets)
		if err := mgr.Keeper().MintToModule(ctx, ModuleName, mintCoin); err != nil {
			ctx.Logger().Error("fail to mint coins during swap", "error", err)
			return cosmos.ZeroUint(), evt, err
		} else {
			mintEvt := NewEventMintBurn(MintSupplyType, mintCoin.Asset.Native(), mintCoin.Amount, "swap")
			if err := mgr.EventMgr().EmitEvent(ctx, mintEvt); err != nil {
				ctx.Logger().Error("fail to emit mint event", "error", err)
			}

			if err := mgr.Keeper().SendFromModuleToModule(ctx, ModuleName, AsgardName, common.NewCoins(mintCoin)); err != nil {
				ctx.Logger().Error("fail to move coins during swap", "error", err)
				return cosmos.ZeroUint(), evt, err
			}
		}
	}

	// Use pool fields here rather than X and Y as synthVirtualDepthMult could affect X and Y.
	// Only alter BalanceAsset when the non-RUNE asset isn't a synth.
	if source.IsSwitch() {
		pool.BalanceRune = pool.BalanceRune.Add(x)
		if !target.IsSyntheticAsset() {
			pool.BalanceAsset = common.SafeSub(pool.BalanceAsset, emitAssets)
		}
	} else {
		// The target should be RUNE.
		pool.BalanceRune = common.SafeSub(pool.BalanceRune, emitAssets)
		if !source.IsSyntheticAsset() {
			pool.BalanceAsset = pool.BalanceAsset.Add(x)
		}
	}
	if source.IsSyntheticAsset() || target.IsSyntheticAsset() {
		synthSupply = keeper.GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
		pool.CalcUnits(synthSupply)
	}

	// Now that pool depths have been adjusted to post-swap, determine LiquidityFeeInRune.
	if target.IsSwitch() {
		// Because the output asset is RUNE, liquidity Fee is already in RUNE.
		swapEvt.LiquidityFeeInRune = swapEvt.LiquidityFee
	} else {
		// Momentarily deduct the liquidity fee for RuneDisbursementForAssetAdd.
		pool.BalanceAsset = common.SafeSub(pool.BalanceAsset, swapEvt.LiquidityFee)
		swapEvt.LiquidityFeeInRune = pool.RuneDisbursementForAssetAdd(swapEvt.LiquidityFee)
		// Restore the BalanceAsset which is constant-depths-product swapped for the LiquidityFeeInRune.
		pool.BalanceAsset = pool.BalanceAsset.Add(swapEvt.LiquidityFee)
	}

	if !pool.Asset.IsDerivedAsset() {
		// Deduct LiquidityFeeInRune from the pool's RUNE depth and send it to the Reserve Module to be system income.
		// (So that the liquidity fee isn't used for later swaps in the same block.)
		if !swapEvt.LiquidityFeeInRune.IsZero() {
			pool.BalanceRune = common.SafeSub(pool.BalanceRune, swapEvt.LiquidityFeeInRune)
			liqFeeCoin := common.NewCoin(common.SwitchNative, swapEvt.LiquidityFeeInRune)

			targetModule := ReserveName
			if pool.Asset.IsTCY() {
				targetModule = TCYClaimingName
			}

			if err := mgr.Keeper().SendFromModuleToModule(ctx, AsgardName, targetModule, common.NewCoins(liqFeeCoin)); err != nil {
				ctx.Logger().Error("fail to move liquidity fee RUNE during swap", "module", targetModule, "error", err)
				return cosmos.ZeroUint(), evt, err
			}

			// Only add to liquidity fees if not TCY
			if !pool.Asset.IsTCY() {
				if err := keeper.AddToLiquidityFees(ctx, pool.Asset, swapEvt.LiquidityFeeInRune); err != nil {
					return cosmos.ZeroUint(), evt, fmt.Errorf("fail to add to liquidity fees: %w", err)
				}
			}
		}

		// use calculated floor
		if err := keeper.AddToSwapSlip(ctx, pool.Asset, cosmos.NewInt(int64(swapEvt.PoolSlip.Uint64()))); err != nil {
			return cosmos.ZeroUint(), evt, fmt.Errorf("fail to add to swap slip: %w", err)
		}
	}

	ctx.Logger().Info("post swap", "pool", pool.Asset, "rune", pool.BalanceRune, "asset", pool.BalanceAsset, "lp units", pool.LPUnits, "synth units", pool.SynthUnits, "emit asset", emitAssets)

	// Even for a Derived Asset pool, set the pool so the txout manager's GetFee for toi.Coin.Asset uses updated balances.
	if err := keeper.SetPool(ctx, pool); err != nil {
		return cosmos.ZeroUint(), evt, fmt.Errorf("fail to set pool")
	}

	// if target is trade account, check whether swaps to trade assets are enabled
	if target.IsTradeAsset() {
		if tradeAccountsEnabled <= 0 {
			return cosmos.ZeroUint(), evt, fmt.Errorf("trade accounts are disabled")
		}
	}

	// if target is secured asset, check whether swaps to secured assets are enabled
	if target.IsSecuredAsset() {
		if err := mgr.SecuredAssetManager().CheckHalt(ctx); err != nil {
			return cosmos.ZeroUint(), evt, err
		}
	}

	// apply swapper clout
	availableClout := swapEvt.LiquidityFeeInRune
	for i, addr := range []common.Address{tx.FromAddress, destination} {
		if addr.IsEmpty() {
			ctx.Logger().Error("dev error: address is empty for clout calculation")
			continue
		}
		clout, err := keeper.GetSwapperClout(ctx, addr)
		if err != nil {
			ctx.Logger().Error("fail to get swapper clout destination address", "error", err)
			continue
		}
		if i == 0 {
			clout.Score = clout.Score.Add(availableClout.QuoUint64(2))
			availableClout = common.SafeSub(availableClout, availableClout.QuoUint64(2))
		} else {
			clout.Score = clout.Score.Add(availableClout)
		}
		if err := keeper.SetSwapperClout(ctx, clout); err != nil {
			ctx.Logger().Error("fail to save swapper clout", "error", err)
		}
	}

	return emitAssets, swapEvt, nil
}

// calculate the number of assets sent to the address (includes liquidity fee)
// nolint
func (s *SwapperVCUR) CalcAssetEmission(X, x, Y cosmos.Uint) cosmos.Uint {
	// ( x * X * Y ) / ( x + X )^2
	numerator := x.Mul(X).Mul(Y)
	denominator := x.Add(X).Mul(x.Add(X))
	if denominator.IsZero() {
		return cosmos.ZeroUint()
	}
	return numerator.Quo(denominator)
}

// calculate the asset amount to be sent to address using a predefined fee (fee calculated using artificial floor)
// nolint
func (s *SwapperVCUR) CalcMaxAssetEmission(X, x, Y, fee cosmos.Uint) cosmos.Uint {
	// (( x * Y ) / ( x + X )) - fee
	numerator := x.Mul(Y)
	denominator := x.Add(X)
	if denominator.IsZero() {
		return cosmos.ZeroUint()
	}
	return common.SafeSub(numerator.Quo(denominator), fee)
}

// CalculateLiquidityFee the fee of the swap
// nolint
func (s *SwapperVCUR) CalcLiquidityFee(X, x, Y cosmos.Uint) cosmos.Uint {
	// ( x^2 *  Y ) / ( x + X )^2
	numerator := x.Mul(x).Mul(Y)
	denominator := x.Add(X).Mul(x.Add(X))
	if denominator.IsZero() {
		return cosmos.ZeroUint()
	}
	return numerator.Quo(denominator)
}

// CalcMinLiquidityFee calculates the fee of the swap using min artificial slip floor
// nolint
func (s *SwapperVCUR) CalcMinLiquidityFee(X, x, Y, minSlipBps cosmos.Uint) cosmos.Uint {
	// minSlip * ( x  *  Y ) / ( x + X )
	numerator := common.GetSafeShare(minSlipBps, cosmos.NewUint(constants.MaxBasisPts), x.Mul(Y))
	denominator := x.Add(X)
	if denominator.IsZero() {
		return cosmos.ZeroUint()
	}
	return numerator.Quo(denominator)
}

// CalcSwapSlip - calculate the swap slip, expressed in basis points (10000)
// nolint
func (s *SwapperVCUR) CalcSwapSlip(Xi, xi cosmos.Uint) cosmos.Uint {
	// Cast to DECs
	xD := cosmos.NewDecFromBigInt(xi.BigInt())
	XD := cosmos.NewDecFromBigInt(Xi.BigInt())
	dec10k := cosmos.NewDec(10000)
	// x / (x + X)
	denD := xD.Add(XD)
	if denD.IsZero() {
		return cosmos.ZeroUint()
	}
	swapSlipD := xD.Quo(denD)                                     // Division with DECs
	swapSlip := swapSlipD.Mul(dec10k)                             // Adds 5 0's
	swapSlipUint := cosmos.NewUint(uint64(swapSlip.RoundInt64())) // Casts back to Uint as Basis Points
	return swapSlipUint
}

// GetSwapCalc returns emission, liquidity fee and slip for a swap
// nolint
func (s *SwapperVCUR) GetSwapCalc(X, x, Y, slipBps, minSlipBps cosmos.Uint) (emitAssets, liquidityFee, slip cosmos.Uint) {
	if minSlipBps.GT(slipBps) {
		// adjust calc emission based on artificial floor
		liquidityFee = s.CalcMinLiquidityFee(X, x, Y, minSlipBps)
		emitAssets = s.CalcMaxAssetEmission(X, x, Y, liquidityFee)
		slip = minSlipBps
	} else {
		liquidityFee = s.CalcLiquidityFee(X, x, Y)
		emitAssets = s.CalcAssetEmission(X, x, Y)
		slip = slipBps
	}
	return
}
