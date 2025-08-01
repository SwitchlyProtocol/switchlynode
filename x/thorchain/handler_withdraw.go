package thorchain

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
	"github.com/hashicorp/go-multierror"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

// WithdrawLiquidityHandler to process withdraw requests
type WithdrawLiquidityHandler struct {
	mgr Manager
}

// NewWithdrawLiquidityHandler create a new instance of WithdrawLiquidityHandler to process withdraw request
func NewWithdrawLiquidityHandler(mgr Manager) WithdrawLiquidityHandler {
	return WithdrawLiquidityHandler{
		mgr: mgr,
	}
}

// Run is the main entry point of withdraw
func (h WithdrawLiquidityHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgWithdrawLiquidity)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgWithdrawLiquidity", "withdraw address", msg.WithdrawAddress, "withdraw basis points", msg.BasisPoints)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("MsgWithdrawLiquidity failed validation", "error", err)
		return nil, err
	}

	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg withdraw", "error", err)
		return nil, err
	}
	return result, err
}

func (h WithdrawLiquidityHandler) validate(ctx cosmos.Context, msg MsgWithdrawLiquidity) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h WithdrawLiquidityHandler) validateV3_0_0(ctx cosmos.Context, msg MsgWithdrawLiquidity) error {
	if err := msg.ValidateBasic(); err != nil {
		return errWithdrawFailValidation
	}

	if msg.Asset.IsSyntheticAsset() {
		if h.mgr.Keeper().GetConfigInt64(ctx, constants.BurnSynths) > 0 {
			// Burning synths is disabled (purposeful inconsistency of int64 vs. other mimirs)
			return fmt.Errorf("burning synths is disabled, unable to withdraw to savers")
		}
	}

	if msg.Asset.IsDerivedAsset() {
		return fmt.Errorf("cannot withdraw from a derived asset virtual pool")
	}

	pool, err := h.mgr.Keeper().GetPool(ctx, msg.Asset)
	if err != nil {
		errMsg := fmt.Sprintf("fail to get pool(%s)", msg.Asset)
		return ErrInternal(err, errMsg)
	}

	if err := pool.EnsureValidPoolStatus(&msg); err != nil {
		return multierror.Append(errInvalidPoolStatus, err)
	}

	// when ragnarok kicks off,  all pool will be set PoolStaged , the ragnarok tx's hash will be common.BlankTxID
	if pool.Status != PoolAvailable && !msg.WithdrawalAsset.IsEmpty() && !msg.Tx.ID.Equals(common.BlankTxID) {
		return fmt.Errorf("cannot specify a withdrawal asset while the pool is not available")
	}

	if h.mgr.Keeper().IsChainHalted(ctx, msg.Asset.Chain) || h.mgr.Keeper().IsLPPaused(ctx, msg.Asset.Chain) {
		return fmt.Errorf("unable to withdraw liquidity while chain is halted or paused LP actions")
	}

	return nil
}

func (h WithdrawLiquidityHandler) handle(ctx cosmos.Context, msg MsgWithdrawLiquidity) (*cosmos.Result, error) {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return nil, errBadVersion
	}
}

func (h WithdrawLiquidityHandler) handleV3_0_0(ctx cosmos.Context, msg MsgWithdrawLiquidity) (*cosmos.Result, error) {
	lp, err := h.mgr.Keeper().GetLiquidityProvider(ctx, msg.Asset, msg.WithdrawAddress)
	if err != nil {
		return nil, multierror.Append(errFailGetLiquidityProvider, err)
	}
	runeAmt, assetAmt, units, gasAsset, err := withdraw(ctx, msg, h.mgr)
	if err != nil {
		return nil, ErrInternal(err, "fail to process withdraw request")
	}

	memo := ""
	if msg.Tx.ID.Equals(common.BlankTxID) {
		// tx id is blank, must be triggered by the ragnarok protocol
		memo = NewRagnarokMemo(ctx.BlockHeight()).String()
	}

	// Thanks to CacheContext, the withdraw event can be emitted before handling outbounds,
	// since if there's a later error the event emission will not take place.
	if units.IsZero() {
		// withdraw pending liquidity event
		runeHash := common.TxID("")
		assetHash := common.TxID("")
		if msg.Tx.Chain.Equals(common.SWITCHLYChain) {
			runeHash = msg.Tx.ID
		} else {
			assetHash = msg.Tx.ID
		}
		evt := NewEventPendingLiquidity(
			msg.Asset,
			WithdrawPendingLiquidity,
			lp.RuneAddress,
			runeAmt,
			lp.AssetAddress,
			assetAmt,
			runeHash,
			assetHash,
		)
		if err := h.mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
			return nil, multierror.Append(errFailSaveEvent, err)
		}
	} else {
		withdrawEvt := NewEventWithdraw(
			msg.Asset,
			units,
			int64(msg.BasisPoints.Uint64()),
			cosmos.ZeroDec(),
			msg.Tx,
			assetAmt,
			runeAmt,
		)
		if err := h.mgr.EventMgr().EmitEvent(ctx, withdrawEvt); err != nil {
			return nil, multierror.Append(errFailSaveEvent, err)
		}
	}

	transfer := func(coin common.Coin, addr common.Address) error {
		toi := TxOutItem{
			Chain:     coin.Asset.GetChain(),
			InHash:    msg.Tx.ID,
			ToAddress: addr,
			Coin:      coin,
			Memo:      memo,
		}
		if !gasAsset.IsZero() {
			// TODO: chain specific logic should be in a single location
			if msg.Asset.GetChain().GetGasAsset().Equals(msg.Asset) {
				toi.MaxGas = common.Gas{
					common.NewCoin(msg.Asset.GetChain().GetGasAsset(), gasAsset),
				}
			}
			toi.GasRate = int64(h.mgr.GasMgr().GetGasRate(ctx, msg.Asset.GetChain()).Uint64())
		}

		ok, err := h.mgr.TxOutStore().TryAddTxOutItem(ctx, h.mgr, toi, cosmos.ZeroUint())
		if err != nil {
			return multierror.Append(errFailAddOutboundTx, err)
		}
		if !ok {
			return errFailAddOutboundTx
		}

		return nil
	}

	if !assetAmt.IsZero() {
		coin := common.NewCoin(msg.Asset, assetAmt)
		// TODO: this might be an issue for single sided/AVAX->ETH, ETH -> AVAX
		if !msg.Asset.IsSwitch() && !lp.AssetAddress.IsChain(msg.Asset.GetChain()) {
			if err := h.swap(ctx, msg, coin, lp.AssetAddress); err != nil {
				return nil, err
			}
		} else {
			if err := transfer(coin, lp.AssetAddress); err != nil {
				return nil, err
			}
		}
	}

	if !runeAmt.IsZero() {
		coin := common.NewCoin(common.SwitchNative, runeAmt)
		if err := transfer(coin, lp.RuneAddress); err != nil {
			return nil, err
		}

		// if its the POL withdrawing, track rune withdrawn
		polAddress, err := h.mgr.Keeper().GetModuleAddress(ReserveName)
		if err != nil {
			return nil, err
		}

		if polAddress.Equals(lp.RuneAddress) {
			pol, err := h.mgr.Keeper().GetPOL(ctx)
			if err != nil {
				return nil, err
			}
			pol.RuneWithdrawn = pol.RuneWithdrawn.Add(runeAmt)

			if err := h.mgr.Keeper().SetPOL(ctx, pol); err != nil {
				return nil, err
			}

			ctx.Logger().Info("POL withdrawn", "pool", msg.Asset, "rune", runeAmt)
			telemetry.IncrCounterWithLabels(
				[]string{"thornode", "pol", "pool", "rune_withdrawn"},
				telem(runeAmt),
				[]metrics.Label{telemetry.NewLabel("pool", msg.Asset.String())},
			)
		}
	}

	// any extra rune in the transaction will be donated to reserve
	reserveCoin := msg.Tx.Coins.GetCoin(common.SwitchNative)
	if !reserveCoin.IsEmpty() {
		if err := h.mgr.Keeper().AddPoolFeeToReserve(ctx, reserveCoin.Amount); err != nil {
			ctx.Logger().Error("fail to add fee to reserve", "error", err)
			return nil, err
		}
	}

	// any extra non-rune in the transaction will be donated to its pool, if existing
	for _, withdrawalCoin := range msg.Tx.Coins {
		if withdrawalCoin.IsEmpty() || withdrawalCoin.Asset == common.SwitchNative {
			continue
		}

		withdrawalCoinPool, err := h.mgr.Keeper().GetPool(ctx, withdrawalCoin.Asset)
		if err != nil {
			return nil, ErrInternal(err, "fail to get pool")
		}

		if withdrawalCoinPool.IsEmpty() {
			continue
		}

		withdrawalCoinPool.BalanceAsset = withdrawalCoinPool.BalanceAsset.Add(withdrawalCoin.Amount)
		if err := h.mgr.Keeper().SetPool(ctx, withdrawalCoinPool); err != nil {
			return nil, ErrInternal(err, "fail to save pool to key value store")
		}
	}

	return &cosmos.Result{}, nil
}

func (h WithdrawLiquidityHandler) swap(ctx cosmos.Context, msg MsgWithdrawLiquidity, coin common.Coin, addr common.Address) error {
	// ensure TxID does NOT have a collision with another swap, this could
	// happen if the user submits two identical loan requests in the same
	// block
	if ok := h.mgr.Keeper().HasSwapQueueItem(ctx, msg.Tx.ID, 0); ok {
		return fmt.Errorf("txn hash conflict")
	}

	// Use layer 1 asset in case msg.Asset is synthetic (i.e. savers withdraw)
	targetAsset := msg.Asset.GetLayer1Asset()

	// Get streaming swaps interval to use for synth -> native swap
	ssInterval := h.mgr.Keeper().GetConfigInt64(ctx, constants.SaversStreamingSwapsInterval)
	if ssInterval <= 0 {
		ssInterval = 0
	}

	// if the asset is in ragnarok, disable streaming withdraw
	key := "RAGNAROK-" + targetAsset.MimirString()
	ragnarok, err := h.mgr.Keeper().GetMimir(ctx, key)
	if err == nil && ragnarok > 0 {
		ssInterval = 0
	}

	memo := fmt.Sprintf("=:%s:%s", targetAsset, addr)
	msg.Tx.Memo = memo
	msg.Tx.Coins = common.NewCoins(coin)
	swapMsg := NewMsgSwap(msg.Tx, targetAsset, addr, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, uint64(ssInterval), msg.Signer)

	// sanity check swap msg
	handler := NewSwapHandler(h.mgr)
	if err := handler.validate(ctx, *swapMsg); err != nil {
		return err
	}

	if err := h.mgr.Keeper().SetSwapQueueItem(ctx, *swapMsg, 0); err != nil {
		ctx.Logger().Error("fail to add swap to queue", "error", err)
		return err
	}

	return nil
}
