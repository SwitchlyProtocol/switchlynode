package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/v3/common"
	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/constants"
	"gitlab.com/thorchain/thornode/v3/x/thorchain/types"
)

// TCYUnstakeHandler to process withdraw requests
type TCYUnstakeHandler struct {
	mgr Manager
}

// NewTCYUnstakeHandler create a new instance of TCYUnstakeHandler to process withdraw request
func NewTCYUnstakeHandler(mgr Manager) TCYUnstakeHandler {
	return TCYUnstakeHandler{
		mgr: mgr,
	}
}

// Run is the main entry point of withdraw
func (h TCYUnstakeHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgTCYUnstake)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgTCYUnstake", "address", msg.Tx.FromAddress, "bps", msg.BasisPoints)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("MsgTCYUnstake failed validation", "error", err)
		return nil, err
	}

	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg tcy unstake", "error", err)
		return nil, err
	}
	return result, err
}

func (h TCYUnstakeHandler) validate(ctx cosmos.Context, msg MsgTCYUnstake) error {
	if err := msg.ValidateBasic(); err != nil {
		return errTCYUnstakeFailValidation
	}
	unstakingHalt := h.mgr.Keeper().GetConfigInt64(ctx, constants.TCYUnstakingHalt)
	if unstakingHalt > 0 {
		return fmt.Errorf("tcy unstaking is halt")
	}
	return nil
}

func (h TCYUnstakeHandler) handle(ctx cosmos.Context, msg MsgTCYUnstake) (*cosmos.Result, error) {
	staker, err := h.mgr.Keeper().GetTCYStaker(ctx, msg.Tx.FromAddress)
	if err != nil {
		return &cosmos.Result{}, err
	}

	unstakeAmount := common.GetSafeShare(msg.BasisPoints, cosmos.NewUint(constants.MaxBasisPts), staker.Amount)
	if unstakeAmount.IsZero() {
		return &cosmos.Result{}, fmt.Errorf("staker: %s doesn't have enough tcy", staker.Address)
	}
	evt := types.NewEventTCYUnstake(msg.Tx.FromAddress, unstakeAmount)

	stakerAddress, err := staker.Address.AccAddress()
	if err != nil {
		return &cosmos.Result{}, err
	}

	ctx.Logger().Info("unstaking tcy", "address", staker.Address.String(), "amount", unstakeAmount)
	coin := common.NewCoin(common.TCY, unstakeAmount)
	err = h.mgr.Keeper().SendFromModuleToAccount(ctx, TCYStakeName, stakerAddress, common.NewCoins(coin))
	if err != nil {
		return &cosmos.Result{}, fmt.Errorf("failed to send from staking module, address: %s, err: %w", msg.Tx.FromAddress.String(), err)
	}
	newStakingAmount := common.SafeSub(staker.Amount, unstakeAmount)
	if newStakingAmount.IsZero() {
		h.mgr.Keeper().DeleteTCYStaker(ctx, msg.Tx.FromAddress)
		return &cosmos.Result{}, h.mgr.EventMgr().EmitEvent(ctx, evt)
	}

	err = h.mgr.Keeper().SetTCYStaker(ctx, types.NewTCYStaker(msg.Tx.FromAddress, newStakingAmount))
	if err != nil {
		return &cosmos.Result{}, err
	}

	return &cosmos.Result{}, h.mgr.EventMgr().EmitEvent(ctx, evt)
}
