package switchly

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// SWCYStakeHandler to process withdraw requests
type SWCYStakeHandler struct {
	mgr Manager
}

// NewSWCYStakeHandler create a new instance of SWCYStakeHandler to process withdraw request
func NewSWCYStakeHandler(mgr Manager) SWCYStakeHandler {
	return SWCYStakeHandler{
		mgr: mgr,
	}
}

// Run is the main entry point of withdraw
func (h SWCYStakeHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgSWCYStake)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgSWCYStake", "address", msg.Tx.FromAddress, "amount", msg.Tx.Coins[0].Amount)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("MsgSWCYStake failed validation", "error", err)
		return nil, err
	}

	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg tcy stake", "error", err)
		return nil, err
	}
	return result, err
}

func (h SWCYStakeHandler) validate(ctx cosmos.Context, msg MsgSWCYStake) error {
	if err := msg.ValidateBasic(); err != nil {
		return errSWCYStakeFailValidation
	}
	stakingHalt := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWCYStakingHalt)
	if stakingHalt > 0 {
		return fmt.Errorf("tcy staking is halted")
	}
	return nil
}

func (h SWCYStakeHandler) handle(ctx cosmos.Context, msg MsgSWCYStake) (*cosmos.Result, error) {
	ctx.Logger().Info("staking tcy claim", "address", msg.Tx.FromAddress.String(), "amount", msg.Tx.Coins[0].Amount.String())

	err := h.mgr.Keeper().UpdateSWCYStaker(ctx, msg.Tx.FromAddress, msg.Tx.Coins[0].Amount)
	if err != nil {
		ctx.Logger().Error("failed to update tcy staker", "err", err)
	}

	evt := types.NewEventSWCYStake(msg.Tx.FromAddress, msg.Tx.Coins[0].Amount)
	return &cosmos.Result{}, h.mgr.EventMgr().EmitEvent(ctx, evt)
}
