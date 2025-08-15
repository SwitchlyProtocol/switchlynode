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

// SwitchPoolWithdrawHandler a handler to process withdrawals from SwitchPool
type SwitchPoolWithdrawHandler struct {
	mgr Manager
}

// NewSwitchPoolWithdrawHandler create new SwitchPoolWithdrawHandler
func NewSwitchPoolWithdrawHandler(mgr Manager) SwitchPoolWithdrawHandler {
	return SwitchPoolWithdrawHandler{
		mgr: mgr,
	}
}

// Run execute the handler
func (h SwitchPoolWithdrawHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgSwitchPoolWithdraw)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgSwitchPoolWithdraw",
		"tx_id", msg.Tx.ID,
		"signer", msg.Signer,
		"basis_points", msg.BasisPoints,
		"affiliate_address", msg.AffiliateAddress,
		"affiliate_basis_points", msg.AffiliateBasisPoints,
	)

	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg switch pool withdraw failed validation", "error", err)
		return nil, err
	}

	err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg switch pool withdraw", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h SwitchPoolWithdrawHandler) validate(ctx cosmos.Context, msg MsgSwitchPoolWithdraw) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h SwitchPoolWithdrawHandler) validateV3_0_0(ctx cosmos.Context, msg MsgSwitchPoolWithdraw) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}
	switchPoolEnabled := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWITCHPoolEnabled)
	if switchPoolEnabled <= 0 {
		return fmt.Errorf("SwitchPool disabled")
	}
	switchPoolWithdrawPaused := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWITCHPoolHaltWithdraw)
	if switchPoolWithdrawPaused > 0 && ctx.BlockHeight() >= switchPoolWithdrawPaused {
		return fmt.Errorf("SwitchPool withdraw paused")
	}
	maxAffBasisPts := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaxAffiliateFeeBasisPoints)
	if !msg.AffiliateBasisPoints.IsZero() && msg.AffiliateBasisPoints.GT(cosmos.NewUint(uint64(maxAffBasisPts))) {
		return fmt.Errorf("invalid affiliate basis points, max: %d, request: %d", maxAffBasisPts, msg.AffiliateBasisPoints.Uint64())
	}
	return nil
}

func (h SwitchPoolWithdrawHandler) handle(ctx cosmos.Context, msg MsgSwitchPoolWithdraw) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h SwitchPoolWithdrawHandler) handleV3_0_0(ctx cosmos.Context, msg MsgSwitchPoolWithdraw) error {
	accAddr, err := cosmos.AccAddressFromBech32(msg.Signer.String())
	if err != nil {
		return fmt.Errorf("unable to AccAddressFromBech32: %s", err)
	}
	switchProvider, err := h.mgr.Keeper().GetSWITCHProvider(ctx, accAddr)
	if err != nil {
		return fmt.Errorf("unable to GetSWITCHProvider: %s", err)
	}

	// ensure the deposit has reached maturity
	depositMaturity := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWITCHPoolDepositMaturityBlocks)
	currentBlockHeight := ctx.BlockHeight()
	blocksSinceLastDeposit := currentBlockHeight - switchProvider.LastDepositHeight
	if blocksSinceLastDeposit < depositMaturity {
		return fmt.Errorf("deposit reaches maturity in %d blocks", depositMaturity-blocksSinceLastDeposit)
	}

	// switch pool tracks the reserve and pooler unit shares of pol
	switchPool, err := h.mgr.Keeper().GetSwitchPool(ctx)
	if err != nil {
		return fmt.Errorf("fail to get switch pool: %s", err)
	}

	// compute withdraw units
	maxBps := cosmos.NewUint(constants.MaxBasisPts)
	withdrawUnits := common.GetSafeShare(msg.BasisPoints, maxBps, switchProvider.Units)

	totalSwitchPoolValue, err := switchPoolValue(ctx, h.mgr)
	if err != nil {
		return fmt.Errorf("fail to get switch pool value: %w", err)
	}

	// determine the profit of the withdraw amount to share with affiliate
	affiliateAmount := cosmos.ZeroUint()
	if !msg.AffiliateBasisPoints.IsZero() {
		totalUnits := switchPool.TotalUnits()
		currentValue := common.GetSafeShare(switchProvider.Units, totalUnits, totalSwitchPoolValue)
		depositRemaining := common.SafeSub(switchProvider.DepositAmount, switchProvider.WithdrawAmount)
		currentYield := common.SafeSub(currentValue, depositRemaining)
		withdrawYield := common.GetSafeShare(msg.BasisPoints, maxBps, currentYield)
		affiliateAmount = common.GetSafeShare(msg.AffiliateBasisPoints, maxBps, withdrawYield)
	}

	// compute withdraw amount
	withdrawAmount := common.GetSafeShare(withdrawUnits, switchPool.TotalUnits(), totalSwitchPoolValue)

	// if insufficient pending units, reserve should enter to create space for withdraw
	pendingSWITCH := h.mgr.Keeper().GetSWITCHBalanceOfModule(ctx, SwitchPoolName)
	if withdrawAmount.GT(pendingSWITCH) {
		reserveEnterSWITCH := common.SafeSub(withdrawAmount, pendingSWITCH)

		// There may be cases where providers are in a state of profit, and for the reserve
		// to buy their share of POL it must exceed POLMaxNetworkDeposit to cover the profit
		// of the provider. We allow exceeding this limit up to SwitchPoolMaxReserveBackstop
		// beyond the POLMaxNetworkDeposit as a circuit breaker. If the circuit breaker is
		// reached, withdraws will fail pending governance to increase the limit or extend
		// logic to trigger POL withdraw and sacrifice pool depth to satisfy withdrawals.
		maxReserveBackstop := h.mgr.Keeper().GetConfigInt64(ctx, constants.SWITCHPoolMaxReserveBackstop)
		polMaxNetworkDeposit := h.mgr.Keeper().GetConfigInt64(ctx, constants.POLMaxNetworkDeposit)
		maxReserveUsage := cosmos.NewInt(maxReserveBackstop + polMaxNetworkDeposit)
		pol, err := h.mgr.Keeper().GetPOL(ctx)
		if err != nil {
			return fmt.Errorf("fail to get POL: %w", err)
		}
		currentReserveDeposit := pol.CurrentDeposit().
			Sub(switchPool.CurrentDeposit()).
			Add(cosmos.NewIntFromBigInt(pendingSWITCH.BigInt()))
		newReserveDeposit := currentReserveDeposit.Add(cosmos.NewIntFromBigInt(reserveEnterSWITCH.BigInt()))
		if newReserveDeposit.GT(maxReserveUsage) {
			return fmt.Errorf("reserve enter %d switch exceeds backstop", reserveEnterSWITCH.Uint64())
		}

		err = reserveEnterSwitchPool(ctx, h.mgr, reserveEnterSWITCH)
		if err != nil {
			return fmt.Errorf("fail to reserve enter switch pool: %w", err)
		}

		// fetch switch pool after reserve enter for updated units
		switchPool, err = h.mgr.Keeper().GetSwitchPool(ctx)
		if err != nil {
			return fmt.Errorf("fail to get switch pool: %w", err)
		}
	}

	// update provider and switch pool records
	switchProvider.Units = common.SafeSub(switchProvider.Units, withdrawUnits)
	switchProvider.WithdrawAmount = switchProvider.WithdrawAmount.Add(withdrawAmount)
	switchProvider.LastWithdrawHeight = ctx.BlockHeight()
	h.mgr.Keeper().SetSWITCHProvider(ctx, switchProvider)
	switchPool.PoolUnits = common.SafeSub(switchPool.PoolUnits, withdrawUnits)
	switchPool.SwitchWithdrawn = switchPool.SwitchWithdrawn.Add(withdrawAmount)
	h.mgr.Keeper().SetSwitchPool(ctx, switchPool)

	// send the affiliate fee
	userAmount := common.SafeSub(withdrawAmount, affiliateAmount)
	if !affiliateAmount.IsZero() {
		affiliateCoins := common.NewCoins(common.NewCoin(common.SwitchNative, affiliateAmount))
		affiliateAddress, err := msg.AffiliateAddress.AccAddress()
		if err != nil {
			return fmt.Errorf("fail to get affiliate address: %w", err)
		}
		err = h.mgr.Keeper().SendFromModuleToAccount(ctx, SwitchPoolName, affiliateAddress, affiliateCoins)
		if err != nil {
			return fmt.Errorf("fail to send affiliate fee: %w", err)
		}
	}

	// send the user the withdraw
	userCoins := common.NewCoins(common.NewCoin(common.SwitchNative, userAmount))
	err = h.mgr.Keeper().SendFromModuleToAccount(ctx, SwitchPoolName, msg.Signer, userCoins)
	if err != nil {
		return fmt.Errorf("fail to send user withdraw: %w", err)
	}

	ctx.Logger().Info(
		"switchpool withdraw",
		"address", msg.Signer,
		"units", withdrawUnits,
		"amount", userAmount,
		"affiliate_amount", affiliateAmount,
	)

	withdrawEvent := NewEventSwitchPoolWithdraw(
		switchProvider.SwitchAddress,
		int64(msg.BasisPoints.Uint64()),
		withdrawAmount,
		withdrawUnits,
		msg.Tx.ID,
		msg.AffiliateAddress,
		int64(msg.AffiliateBasisPoints.Uint64()),
		affiliateAmount,
	)
	if err := h.mgr.EventMgr().EmitEvent(ctx, withdrawEvent); err != nil {
		ctx.Logger().Error("fail to emit switch pool withdraw event", "error", err)
	}

	telemetry.IncrCounterWithLabels(
		[]string{"switchlynode", "rune_pool", "withdraw_count"},
		float32(1),
		[]metrics.Label{},
	)
	telemetry.IncrCounterWithLabels(
		[]string{"switchlynode", "rune_pool", "withdraw_amount"},
		telem(withdrawEvent.SwitchAmount),
		[]metrics.Label{},
	)

	return nil
}
