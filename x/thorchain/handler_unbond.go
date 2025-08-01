package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

// UnBondHandler a handler to process unbond request
type UnBondHandler struct {
	mgr Manager
}

// NewUnBondHandler create new UnBondHandler
func NewUnBondHandler(mgr Manager) UnBondHandler {
	return UnBondHandler{
		mgr: mgr,
	}
}

// Run execute the handler
func (h UnBondHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgUnBond)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgUnBond",
		"node address", msg.NodeAddress,
		"request hash", msg.TxIn.ID,
		"amount", msg.Amount)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg unbond fail validation", "error", err)
		return nil, err
	}
	if err := h.handle(ctx, *msg); err != nil {
		ctx.Logger().Error("msg unbond fail handler", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h UnBondHandler) validate(ctx cosmos.Context, msg MsgUnBond) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	if !msg.TxIn.Coins.IsEmpty() {
		return cosmos.ErrUnknownRequest("unbond message cannot have a non-zero coin amount")
	}

	na, err := h.mgr.Keeper().GetNodeAccount(ctx, msg.NodeAddress)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get node account(%s)", msg.NodeAddress))
	}

	if na.Status == NodeActive || na.Status == NodeReady {
		return cosmos.ErrUnknownRequest("cannot unbond while node is in active or ready status")
	}

	if h.mgr.Keeper().GetConfigInt64(ctx, constants.PauseUnbond) > 0 {
		return ErrInternal(err, "unbonding has been paused")
	}

	bp, err := h.mgr.Keeper().GetBondProviders(ctx, msg.NodeAddress)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get bond providers(%s)", msg.NodeAddress))
	}
	from, err := msg.BondAddress.AccAddress()
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to parse bond address(%s)", msg.BondAddress))
	}
	if !bp.Has(from) && !na.BondAddress.Equals(msg.BondAddress) {
		return cosmos.ErrUnauthorized(fmt.Sprintf("%s are not authorized to manage %s", msg.BondAddress, msg.NodeAddress))
	}

	return nil
}

func (h UnBondHandler) handle(ctx cosmos.Context, msg MsgUnBond) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h UnBondHandler) handleV3_0_0(ctx cosmos.Context, msg MsgUnBond) error {
	na, err := h.mgr.Keeper().GetNodeAccount(ctx, msg.NodeAddress)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get node account(%s)", msg.NodeAddress))
	}

	bondLockPeriod, err := h.mgr.Keeper().GetMimir(ctx, constants.BondLockupPeriod.String())
	if err != nil || bondLockPeriod < 0 {
		bondLockPeriod = h.mgr.GetConstants().GetInt64Value(constants.BondLockupPeriod)
	}
	if ctx.BlockHeight()-na.StatusSince < bondLockPeriod {
		return fmt.Errorf("node can not unbond before %d", na.StatusSince+bondLockPeriod)
	}
	vaults, err := h.mgr.Keeper().GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		return ErrInternal(err, "fail to get retiring vault")
	}
	isMemberOfRetiringVault := false
	for _, v := range vaults {
		if v.GetMembership().Contains(na.PubKeySet.Secp256k1) {
			isMemberOfRetiringVault = true
			ctx.Logger().Info("node account is still part of the retiring vault,can't return bond yet")
			break
		}
	}
	if isMemberOfRetiringVault {
		return ErrInternal(err, "fail to unbond, still part of the retiring vault")
	}

	from, err := cosmos.AccAddressFromBech32(msg.BondAddress.String())
	if err != nil {
		return ErrInternal(err, "fail to parse from address")
	}

	// remove/unbonding bond provider
	// check that 1) requester is node operator, 2) references
	if msg.BondAddress.Equals(na.BondAddress) && !msg.BondProviderAddress.Empty() {
		if err := refundBond(ctx, msg.TxIn, msg.BondProviderAddress, msg.Amount, &na, h.mgr); err != nil {
			return ErrInternal(err, "fail to unbond")
		}

		// remove bond provider (if bond is now zero)
		bondAddr, err := na.BondAddress.AccAddress()
		if err != nil {
			return ErrInternal(err, "fail to refund bond")
		}
		if !bondAddr.Equals(msg.BondProviderAddress) {
			bp, err := h.mgr.Keeper().GetBondProviders(ctx, na.NodeAddress)
			if err != nil {
				return ErrInternal(err, fmt.Sprintf("fail to get bond providers(%s)", na.NodeAddress))
			}
			provider := bp.Get(msg.BondProviderAddress)
			if !provider.IsEmpty() && provider.Bond.IsZero() {
				if ok := bp.Remove(msg.BondProviderAddress); ok {
					if err := h.mgr.Keeper().SetBondProviders(ctx, bp); err != nil {
						return ErrInternal(err, fmt.Sprintf("fail to save bond providers(%s)", bp.NodeAddress.String()))
					}
				}
			}
		}
	} else if err := refundBond(ctx, msg.TxIn, from, msg.Amount, &na, h.mgr); err != nil {
		return ErrInternal(err, "fail to unbond")
	}

	coin := msg.TxIn.Coins.GetCoin(common.SwitchNative)
	if !coin.IsEmpty() {
		na.Bond = na.Bond.Add(coin.Amount)
		if err := h.mgr.Keeper().SetNodeAccount(ctx, na); err != nil {
			return ErrInternal(err, "fail to save node account to key value store")
		}
	}

	return nil
}
