package switchly

import (
	"fmt"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

// SwitchPoolDepositHandler a handler to process deposits to SwitchPool
type SwitchPoolDepositHandler struct {
	mgr Manager
}

// NewSwitchPoolDepositHandler create new SwitchPoolDepositHandler
func NewSwitchPoolDepositHandler(mgr Manager) SwitchPoolDepositHandler {
	return SwitchPoolDepositHandler{
		mgr: mgr,
	}
}

// Run execute the handler
func (h SwitchPoolDepositHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgSwitchPoolDeposit)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgSwitchPoolDeposit",
		"tx_id", msg.Tx.ID,
		"rune_address", msg.Signer,
		"deposit_asset", msg.Tx.Coins[0].Asset,
		"deposit_amount", msg.Tx.Coins[0].Amount,
	)

	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg switch pool deposit failed validation", "error", err)
		return nil, err
	}

	err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg switch pool deposit", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h SwitchPoolDepositHandler) validate(ctx cosmos.Context, msg MsgSwitchPoolDeposit) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h SwitchPoolDepositHandler) validateV3_0_0(ctx cosmos.Context, msg MsgSwitchPoolDeposit) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	switchPoolEnabled := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWITCHPoolEnabled)
	if switchPoolEnabled <= 0 {
		return fmt.Errorf("SwitchPool disabled")
	}
	switchPoolDepositPaused := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWITCHPoolHaltDeposit)
	if switchPoolDepositPaused > 0 && ctx.BlockHeight() >= switchPoolDepositPaused {
		return fmt.Errorf("SwitchPool deposit paused")
	}
	return nil
}

func (h SwitchPoolDepositHandler) handle(ctx cosmos.Context, msg MsgSwitchPoolDeposit) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h SwitchPoolDepositHandler) handleV3_0_0(ctx cosmos.Context, msg MsgSwitchPoolDeposit) error {
	// get switch pool value before deposit
	switchPoolValue, err := switchPoolValue(ctx, h.mgr)
	if err != nil {
		return fmt.Errorf("fail to get switch pool value: %s", err)
	}

	// send deposit to switchpool module
	err = h.mgr.Keeper().SendFromModuleToModule(
		ctx,
		AsgardName,
		SwitchPoolName,
		common.Coins{msg.Tx.Coins[0]},
	)
	if err != nil {
		return fmt.Errorf("unable to SendFromModuleToModule: %s", err)
	}

	accAddr, err := cosmos.AccAddressFromBech32(msg.Signer.String())
	if err != nil {
		return fmt.Errorf("unable to decode AccAddressFromBech32: %s", err)
	}
	switchProvider, err := h.mgr.Keeper().GetSWITCHProvider(ctx, accAddr)
	if err != nil {
		return fmt.Errorf("unable to GetSWITCHProvider: %s", err)
	}

	switchProvider.LastDepositHeight = ctx.BlockHeight()
	switchProvider.DepositAmount = switchProvider.DepositAmount.Add(msg.Tx.Coins[0].Amount)

	// switch pool tracks the reserve and pooler unit shares of pol
	switchPool, err := h.mgr.Keeper().GetSwitchPool(ctx)
	if err != nil {
		return fmt.Errorf("fail to get switch pool: %s", err)
	}

	// if there are no units, this is the initial deposit
	depositUnits := msg.Tx.Coins[0].Amount

	// compute deposit units
	if !switchPool.TotalUnits().IsZero() {
		depositSWITCH := msg.Tx.Coins[0].Amount
		depositUnits = common.GetSafeShare(depositSWITCH, switchPoolValue, switchPool.TotalUnits())
	}

	// update the provider and switch pool records
	switchProvider.Units = switchProvider.Units.Add(depositUnits)
	h.mgr.Keeper().SetSWITCHProvider(ctx, switchProvider)
	switchPool.PoolUnits = switchPool.PoolUnits.Add(depositUnits)
	switchPool.SwitchDeposited = switchPool.SwitchDeposited.Add(msg.Tx.Coins[0].Amount)
	h.mgr.Keeper().SetSwitchPool(ctx, switchPool)

	// rebalance ownership from reserve to poolers if able
	err = reserveExitSwitchPool(ctx, h.mgr)
	if err != nil {
		return fmt.Errorf("fail to exit reserve switch pool: %w", err)
	}

	ctx.Logger().Info(
		"switchpool deposit",
		"address", msg.Signer,
		"units", depositUnits,
		"amount", msg.Tx.Coins[0].Amount,
	)

	depositEvent := NewEventSwitchPoolDeposit(
		switchProvider.SwitchAddress,
		msg.Tx.Coins[0].Amount,
		depositUnits,
		msg.Tx.ID,
	)
	if err := h.mgr.EventMgr().EmitEvent(ctx, depositEvent); err != nil {
		ctx.Logger().Error("fail to emit switch pool deposit event", "error", err)
	}

	telemetry.IncrCounterWithLabels(
		[]string{"switchlynode", "rune_pool", "deposit_count"},
		float32(1),
		[]metrics.Label{},
	)
	telemetry.IncrCounterWithLabels(
		[]string{"switchlynode", "rune_pool", "deposit_amount"},
		telem(depositEvent.SwitchAmount),
		[]metrics.Label{},
	)

	return nil
}
