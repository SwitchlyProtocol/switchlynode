package switchly

import (
	"errors"
	"fmt"

	"github.com/blang/semver"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/keeper"
)

func withdraw(ctx cosmos.Context, msg MsgWithdrawLiquidity, mgr Manager) (cosmos.Uint, cosmos.Uint, cosmos.Uint, cosmos.Uint, error) {
	version := mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return withdrawV3_0_0(ctx, msg, mgr)
	default:
		zero := cosmos.ZeroUint()
		return zero, zero, zero, zero, errInvalidVersion
	}
}

// Performs the withdraw for the provided MsgWithdrawLiquidity message.
// Returns: switchAmt, assetAmount, units, lastWithdraw, err
func withdrawV3_0_0(ctx cosmos.Context, msg MsgWithdrawLiquidity, mgr Manager) (cosmos.Uint, cosmos.Uint, cosmos.Uint, cosmos.Uint, error) {
	if err := validateWithdraw(ctx, mgr.Keeper(), msg); err != nil {
		ctx.Logger().Error("msg withdraw failed validation", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err
	}

	pool, err := mgr.Keeper().GetPool(ctx, msg.Asset)
	if err != nil {
		ctx.Logger().Error("failed to get pool", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err
	}
	synthSupply := mgr.Keeper().GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
	pool.CalcUnits(synthSupply)

	lp, err := mgr.Keeper().GetLiquidityProvider(ctx, msg.Asset, msg.WithdrawAddress)
	if err != nil {
		ctx.Logger().Error("failed to find liquidity provider", "error", err)
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err

	}

	poolSWITCH := pool.BalanceSwitch
	poolAsset := pool.BalanceAsset
	originalLiquidityProviderUnits := lp.Units
	fLiquidityProviderUnit := lp.Units
	if lp.Units.IsZero() {
		if !lp.PendingSwitch.IsZero() || !lp.PendingAsset.IsZero() {
			mgr.Keeper().RemoveLiquidityProvider(ctx, lp)
			pool.PendingInboundSwitch = common.SafeSub(pool.PendingInboundSwitch, lp.PendingSwitch)
			pool.PendingInboundAsset = common.SafeSub(pool.PendingInboundAsset, lp.PendingAsset)
			if err := mgr.Keeper().SetPool(ctx, pool); err != nil {
				ctx.Logger().Error("failed to save pool pending inbound funds", "error", err)
			}
			// remove lp

			return lp.PendingSwitch, cosmos.RoundToDecimal(lp.PendingAsset, pool.Decimals), lp.Units, cosmos.ZeroUint(), nil
		}
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errNoLiquidityUnitLeft
	}

	// fail if the last add height less than the lockup period in the past
	height := ctx.BlockHeight()
	lockupBlocks := mgr.Keeper().GetConfigInt64(ctx, constants.LiquidityLockUpBlocks)
	if height < (lp.LastAddHeight + lockupBlocks) {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errWithdrawLockup
	}

	ctx.Logger().Info("pool before withdraw", "pool units", pool.GetPoolUnits(), "balance SWITCH", poolSWITCH, "balance asset", poolAsset)
	ctx.Logger().Info("liquidity provider before withdraw", "liquidity provider unit", fLiquidityProviderUnit)

	pauseAsym, _ := mgr.Keeper().GetMimir(ctx, fmt.Sprintf("PauseAsymWithdrawal-%s", pool.Asset.GetChain()))
	assetToWithdraw := assetToWithdraw(msg, lp, pauseAsym)

	if pool.Status == PoolAvailable && lp.SwitchDepositValue.IsZero() && lp.AssetDepositValue.IsZero() {
		lp.SwitchDepositValue = lp.SwitchDepositValue.Add(common.GetSafeShare(lp.Units, pool.GetPoolUnits(), pool.BalanceSwitch))
		lp.AssetDepositValue = lp.AssetDepositValue.Add(common.GetSafeShare(lp.Units, pool.GetPoolUnits(), pool.BalanceAsset))
	}

	var withdrawSWITCH, withDrawAsset, unitAfter cosmos.Uint
	if pool.Asset.IsSyntheticAsset() {
		withdrawSWITCH, withDrawAsset, unitAfter = calculateVaultWithdraw(pool.GetPoolUnits(), poolAsset, originalLiquidityProviderUnits, msg.BasisPoints)
	} else {
		// Note, have to use msg.WithdrawAddress rather than msg.Signer,
		// because the POL removePOLLiquidity signer is nodeAccounts[0].NodeAddress, not the ReserveName address.
		withdrawSWITCH, withDrawAsset, unitAfter, err = calculateWithdraw(ctx, mgr.Keeper(), pool.Asset, pool.GetPoolUnits(), poolSWITCH, poolAsset, originalLiquidityProviderUnits, msg.BasisPoints, assetToWithdraw, msg.WithdrawAddress)
		if err != nil {
			ctx.Logger().Error("fail to withdraw", "error", err)
			return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errWithdrawFail
		}
	}
	if !pool.Asset.IsSyntheticAsset() {
		if (withdrawSWITCH.Equal(poolSWITCH) && !withDrawAsset.Equal(poolAsset)) || (!withdrawSWITCH.Equal(poolSWITCH) && withDrawAsset.Equal(poolAsset)) {
			ctx.Logger().Error("fail to withdraw: cannot withdraw 100% of only one side of the pool")
			return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errWithdrawFail
		}
	}
	withDrawAsset = cosmos.RoundToDecimal(withDrawAsset, pool.Decimals)
	gasAsset := cosmos.ZeroUint()
	// If the pool is empty, and there is a gas asset, subtract required gas
	if common.SafeSub(pool.GetPoolUnits(), fLiquidityProviderUnit).Add(unitAfter).IsZero() {
		maxGas, err := mgr.GasMgr().GetMaxGas(ctx, pool.Asset.GetChain())
		if err != nil {
			ctx.Logger().Error("fail to get gas for asset", "asset", pool.Asset, "error", err)
			return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errWithdrawFail
		}
		// minus gas costs for our transactions
		// TODO: chain specific logic should be in a single location
		if pool.Asset.GetChain().GetGasAsset().Equals(pool.Asset) {
			gasAsset = maxGas.Amount
			if gasAsset.GT(withDrawAsset) {
				gasAsset = withDrawAsset
			}
			withDrawAsset = common.SafeSub(withDrawAsset, gasAsset)
		}
	}

	ctx.Logger().Info("client withdraw", "SWITCH", withdrawSWITCH, "asset", withDrawAsset, "units left", unitAfter)
	// update pool
	pool.LPUnits = common.SafeSub(pool.LPUnits, common.SafeSub(fLiquidityProviderUnit, unitAfter))
	pool.BalanceSwitch = common.SafeSub(poolSWITCH, withdrawSWITCH)
	pool.BalanceAsset = common.SafeSub(poolAsset, withDrawAsset)

	ctx.Logger().Info("pool after withdraw", "pool unit", pool.GetPoolUnits(), "balance SWITCH", pool.BalanceSwitch, "balance asset", pool.BalanceAsset)

	lp.LastWithdrawHeight = ctx.BlockHeight()
	maxPts := cosmos.NewUint(uint64(MaxWithdrawBasisPoints))
	lp.SwitchDepositValue = common.SafeSub(lp.SwitchDepositValue, common.GetSafeShare(msg.BasisPoints, maxPts, lp.SwitchDepositValue))
	lp.AssetDepositValue = common.SafeSub(lp.AssetDepositValue, common.GetSafeShare(msg.BasisPoints, maxPts, lp.AssetDepositValue))
	lp.Units = unitAfter

	// sanity check, we don't increase LP units
	if unitAfter.GTE(originalLiquidityProviderUnits) {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), ErrInternal(err, fmt.Sprintf("sanity check: LP units cannot increase during a withdrawal: %d --> %d", originalLiquidityProviderUnits.Uint64(), unitAfter.Uint64()))
	}

	// Create a pool event if SWITCHLYNode have no switch or assets
	if (pool.BalanceAsset.IsZero() || pool.BalanceSwitch.IsZero()) && !pool.Asset.IsSyntheticAsset() {
		poolEvt := NewEventPool(pool.Asset, PoolStaged)
		if err := mgr.EventMgr().EmitEvent(ctx, poolEvt); nil != err {
			ctx.Logger().Error("fail to emit pool event", "error", err)
		}
		pool.Status = PoolStaged
	}

	if err := mgr.Keeper().SetPool(ctx, pool); err != nil {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), ErrInternal(err, "fail to save pool")
	}
	if mgr.Keeper().RagnarokInProgress(ctx) {
		mgr.Keeper().SetLiquidityProvider(ctx, lp)
	} else {
		if !lp.Units.Add(lp.PendingAsset).Add(lp.PendingSwitch).IsZero() {
			mgr.Keeper().SetLiquidityProvider(ctx, lp)
		} else {
			mgr.Keeper().RemoveLiquidityProvider(ctx, lp)
		}
	}

	return withdrawSWITCH, withDrawAsset, common.SafeSub(originalLiquidityProviderUnits, unitAfter), gasAsset, nil
}

func assetToWithdraw(msg MsgWithdrawLiquidity, lp LiquidityProvider, pauseAsym int64) common.Asset {
	if lp.SwitchAddress.IsEmpty() {
		return msg.Asset
	}
	if lp.AssetAddress.IsEmpty() {
		return common.SwitchNative
	}
	if pauseAsym > 0 {
		return common.EmptyAsset
	}

	return msg.WithdrawalAsset
}

func calculateWithdraw(ctx cosmos.Context, keeper keeper.Keeper, poolAsset common.Asset, poolUnits, poolSWITCHDepth, poolAssetDepth, lpUnits, withdrawBasisPoints cosmos.Uint, withdrawalAsset common.Asset, withdrawAddress common.Address) (cosmos.Uint, cosmos.Uint, cosmos.Uint, error) {
	version := keeper.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return calculateWithdrawV3_0_0(ctx, keeper, poolAsset, poolUnits, poolSWITCHDepth, poolAssetDepth, lpUnits, withdrawBasisPoints, withdrawalAsset, withdrawAddress)
	default:
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errBadVersion
	}
}

func calculateWithdrawV3_0_0(ctx cosmos.Context, keeper keeper.Keeper, poolAsset common.Asset, poolUnits, poolSWITCHDepth, poolAssetDepth, lpUnits, withdrawBasisPoints cosmos.Uint, withdrawalAsset common.Asset, withdrawAddress common.Address) (cosmos.Uint, cosmos.Uint, cosmos.Uint, error) {
	if poolUnits.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("poolUnits can't be zero")
	}
	if poolSWITCHDepth.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("pool switch balance can't be zero")
	}
	if poolAssetDepth.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("pool asset balance can't be zero")
	}
	if lpUnits.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("liquidity provider unit can't be zero")
	}
	if withdrawBasisPoints.GT(cosmos.NewUint(MaxWithdrawBasisPoints)) {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), fmt.Errorf("withdraw basis point %s is not valid", withdrawBasisPoints.String())
	}

	unitsToClaim := common.GetSafeShare(withdrawBasisPoints, cosmos.NewUint(10000), lpUnits)
	unitAfter := common.SafeSub(lpUnits, unitsToClaim)

	withdrawSWITCH := common.GetSafeShare(unitsToClaim, poolUnits, poolSWITCHDepth)
	withdrawAsset := common.GetSafeShare(unitsToClaim, poolUnits, poolAssetDepth)
	if withdrawalAsset.IsEmpty() {
		return withdrawSWITCH, withdrawAsset, unitAfter, nil
	}

	// Past this point is asymmetric withdrawal only,
	// the withdrawn half from one side being swapped to the other side.

	swapper, err := GetSwapper(keeper.GetVersion())
	if err != nil {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err
	}

	remainingAsset := common.SafeSub(poolAssetDepth, withdrawAsset)
	remainingSWITCH := common.SafeSub(poolSWITCHDepth, withdrawSWITCH)
	var x, X, Y, yAdd cosmos.Uint
	var isPOL bool
	if withdrawalAsset.IsSwitch() || withdrawalAsset.IsSwitch() {
		// POL withdraws are SWITCH-only, so only needing to check for SWITCH asymmetric withdraws.
		polAddress, err := keeper.GetModuleAddress(ReserveName)
		if err != nil {
			ctx.Logger().Error("failed to get reserve module address", "error", err)
			return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint(), err
		}
		isPOL = withdrawAddress.Equals(polAddress)

		x = withdrawAsset
		X = remainingAsset
		Y = remainingSWITCH
		yAdd = withdrawSWITCH
	} else {
		x = withdrawSWITCH
		X = remainingSWITCH
		Y = remainingAsset
		yAdd = withdrawAsset
	}

	swapSlipBps := swapper.CalcSwapSlip(X, x)
	minSlipBps := getMinSlipBps(ctx, keeper, poolAsset)
	// Being for a non-Savers asymmetric liquidity action, MinSlipBps should always be for an L1 swap.
	y, liqFee, _ := swapper.GetSwapCalc(X, x, Y, swapSlipBps, minSlipBps)
	outputAmount := y.Add(yAdd)
	// Waive any implicit slip fee for POL withdrawals, effectively an XYK constant-depths half-swap.
	if isPOL {
		outputAmount = outputAmount.Add(liqFee)
	}

	if withdrawalAsset.IsSwitch() || withdrawalAsset.IsSwitch() {
		return outputAmount, cosmos.ZeroUint(), unitAfter, nil
	}
	return cosmos.ZeroUint(), outputAmount, unitAfter, nil
}

func calculateVaultWithdraw(vaultUnits, assetAmt, lpUnits, withdrawBasisPoints cosmos.Uint) (cosmos.Uint, cosmos.Uint, cosmos.Uint) {
	if vaultUnits.IsZero() || lpUnits.IsZero() || assetAmt.IsZero() || withdrawBasisPoints.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), cosmos.ZeroUint()
	}

	unitsToClaim := common.GetSafeShare(withdrawBasisPoints, cosmos.NewUint(MaxWithdrawBasisPoints), lpUnits)
	unitAfter := common.SafeSub(lpUnits, unitsToClaim)
	withdrawAsset := common.GetSafeShare(unitsToClaim, vaultUnits, assetAmt)
	return cosmos.ZeroUint(), withdrawAsset, unitAfter
}

func validateWithdraw(ctx cosmos.Context, keeper keeper.Keeper, msg MsgWithdrawLiquidity) error {
	if msg.WithdrawAddress.IsEmpty() {
		return errors.New("empty withdraw address")
	}
	if msg.Tx.ID.IsEmpty() {
		return errors.New("request tx hash is empty")
	}
	if msg.Asset.IsEmpty() {
		return errors.New("empty asset")
	}
	withdrawBasisPoints := msg.BasisPoints
	// when BasisPoints is zero, it will be override in parse memo, so if a message can get here
	// the witdrawBasisPoints must between 1~MaxWithdrawBasisPoints
	if !withdrawBasisPoints.GT(cosmos.ZeroUint()) || withdrawBasisPoints.GT(cosmos.NewUint(MaxWithdrawBasisPoints)) {
		return fmt.Errorf("withdraw basis points %s is invalid", msg.BasisPoints)
	}
	if !keeper.PoolExist(ctx, msg.Asset) {
		// pool doesn't exist
		return fmt.Errorf("pool-%s doesn't exist", msg.Asset)
	}
	return nil
}
