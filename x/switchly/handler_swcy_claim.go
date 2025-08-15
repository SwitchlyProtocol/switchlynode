package switchly

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// SWCYClaimHandler to process withdraw requests
type SWCYClaimHandler struct {
	mgr Manager
}

// NewSWCYClaimHandler create a new instance of SWCYClaimHandler to process withdraw request
func NewSWCYClaimHandler(mgr Manager) SWCYClaimHandler {
	return SWCYClaimHandler{
		mgr: mgr,
	}
}

// Run is the main entry point of withdraw
func (h SWCYClaimHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgSWCYClaim)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgSWCYClaim", "rune_address", msg.SwitchAddress, "l1_address", msg.L1Address)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("MsgSWCYClaim failed validation", "error", err)
		return nil, err
	}

	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg tcy claim", "error", err)
		return nil, err
	}
	return result, err
}

func (h SWCYClaimHandler) validate(ctx cosmos.Context, msg MsgSWCYClaim) error {
	if err := msg.ValidateBasic(); err != nil {
		return errSWCYClaimFailValidation
	}
	claimingHalt := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWCYClaimingHalt)
	if claimingHalt > 0 {
		return fmt.Errorf("tcy claiming is halted")
	}

	if !msg.SwitchAddress.IsChain(common.SWITCHLYChain) {
		return cosmos.ErrUnknownRequest("invalid switch address")
	}

	return nil
}

func (h SWCYClaimHandler) handle(ctx cosmos.Context, msg MsgSWCYClaim) (*cosmos.Result, error) {
	claimingSWCYBalance := h.mgr.Keeper().GetBalanceOfModule(ctx, SWCYClaimingName, common.SWCY.Native())
	if claimingSWCYBalance.IsZero() {
		return &cosmos.Result{}, fmt.Errorf("claiming module doesn't have tcy funds")
	}

	claims, err := h.mgr.Keeper().ListSWCYClaimersFromL1Address(ctx, msg.L1Address)
	if err != nil {
		return &cosmos.Result{}, err
	}

	for _, claim := range claims {
		if claim.Amount.IsZero() {
			ctx.Logger().Info("claimer doesn't have tcy to claim", "address", claim.L1Address.String(), "asset", claim.Asset.String())
			continue
		}

		ctx.Logger().Info("staking tcy claim", "l1_address", claim.L1Address.String(), "rune_address", msg.SwitchAddress.String(), "amount", claim.Amount, "asset", claim.Asset.String())
		coin := common.NewCoin(common.SWCY, claim.Amount)
		err = h.mgr.Keeper().SendFromModuleToModule(ctx, SWCYClaimingName, SWCYStakeName, common.NewCoins(coin))
		if err != nil {
			ctx.Logger().Error("failed to send from claiming to staking module", "err", err)
			continue
		}

		err = h.mgr.Keeper().UpdateSWCYStaker(ctx, msg.SwitchAddress, claim.Amount)
		if err != nil {
			ctx.Logger().Error("failed to update tcy staker", "err", err)
		}

		h.mgr.Keeper().DeleteSWCYClaimer(ctx, claim.L1Address, claim.Asset)

		evt := types.NewEventSWCYClaim(msg.SwitchAddress, msg.L1Address, claim.Amount, claim.Asset)
		if err := h.mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
			ctx.Logger().Error("fail to emit tcy claim event", "error", err)
		}

	}

	return &cosmos.Result{}, err
}
