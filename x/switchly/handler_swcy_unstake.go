package switchly

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// SWCYUnstakeHandler to process withdraw requests
type SWCYUnstakeHandler struct {
	mgr Manager
}

// NewSWCYUnstakeHandler create a new instance of SWCYUnstakeHandler to process withdraw request
func NewSWCYUnstakeHandler(mgr Manager) SWCYUnstakeHandler {
	return SWCYUnstakeHandler{
		mgr: mgr,
	}
}

// Run is the main entry point of withdraw
func (h SWCYUnstakeHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgSWCYUnstake)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgSWCYUnstake", "address", msg.Tx.FromAddress, "bps", msg.BasisPoints)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("MsgSWCYUnstake failed validation", "error", err)
		return nil, err
	}

	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg tcy unstake", "error", err)
		return nil, err
	}
	return result, err
}

func (h SWCYUnstakeHandler) validate(ctx cosmos.Context, msg MsgSWCYUnstake) error {
	if err := msg.ValidateBasic(); err != nil {
		return errSWCYUnstakeFailValidation
	}
	unstakingHalt := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWCYUnstakingHalt)
	if unstakingHalt > 0 {
		return fmt.Errorf("tcy unstaking is halt")
	}
	return nil
}

func (h SWCYUnstakeHandler) handle(ctx cosmos.Context, msg MsgSWCYUnstake) (*cosmos.Result, error) {
	staker, err := h.mgr.Keeper().GetSWCYStaker(ctx, msg.Tx.FromAddress)
	if err != nil {
		return &cosmos.Result{}, err
	}

	unstakeAmount := common.GetSafeShare(msg.BasisPoints, cosmos.NewUint(constants.MaxBasisPts), staker.Amount)
	if unstakeAmount.IsZero() {
		return &cosmos.Result{}, fmt.Errorf("staker: %s doesn't have enough tcy", staker.Address)
	}
	evt := types.NewEventSWCYUnstake(msg.Tx.FromAddress, unstakeAmount)

	stakerAddress, err := staker.Address.AccAddress()
	if err != nil {
		return &cosmos.Result{}, err
	}

	ctx.Logger().Info("unstaking tcy", "address", staker.Address.String(), "amount", unstakeAmount)
	coin := common.NewCoin(common.SWCY, unstakeAmount)
	err = h.mgr.Keeper().SendFromModuleToAccount(ctx, SWCYStakeName, stakerAddress, common.NewCoins(coin))
	if err != nil {
		return &cosmos.Result{}, fmt.Errorf("failed to send from staking module, address: %s, err: %w", msg.Tx.FromAddress.String(), err)
	}
	newStakingAmount := common.SafeSub(staker.Amount, unstakeAmount)
	if newStakingAmount.IsZero() {
		h.mgr.Keeper().DeleteSWCYStaker(ctx, msg.Tx.FromAddress)
		return &cosmos.Result{}, h.mgr.EventMgr().EmitEvent(ctx, evt)
	}

	err = h.mgr.Keeper().SetSWCYStaker(ctx, types.NewSWCYStaker(msg.Tx.FromAddress, newStakingAmount))
	if err != nil {
		return &cosmos.Result{}, err
	}

	return &cosmos.Result{}, h.mgr.EventMgr().EmitEvent(ctx, evt)
}
