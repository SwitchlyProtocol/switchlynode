package thorchain

import (
	"strings"

	"github.com/blang/semver"

	"gitlab.com/thorchain/thornode/v3/common/cosmos"
)

// NoOpHandler is to handle donate message
type NoOpHandler struct {
	mgr Manager
}

// NewNoOpHandler create a new instance of NoOpHandler
func NewNoOpHandler(mgr Manager) NoOpHandler {
	return NoOpHandler{
		mgr: mgr,
	}
}

// Run is the main entry point to execute donate logic
func (h NoOpHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgNoOp)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive msg no op", "tx_id", msg.ObservedTx.Tx.ID)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg no op failed validation", "error", err)
		return nil, err
	}

	if err := h.handle(ctx, *msg); err != nil {
		ctx.Logger().Error("fail to process msg noop", "error", err)
		return nil, err
	}
	return &cosmos.Result{}, nil
}

func (h NoOpHandler) validate(ctx cosmos.Context, msg MsgNoOp) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h NoOpHandler) validateV3_0_0(ctx cosmos.Context, msg MsgNoOp) error {
	return msg.ValidateBasic()
}

// handle process MsgNoOp, MsgNoOp add asset and RUNE to the asset pool
// it simply increase the pool asset/RUNE balance but without taking any of the pool units
func (h NoOpHandler) handle(ctx cosmos.Context, msg MsgNoOp) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h NoOpHandler) handleV3_0_0(ctx cosmos.Context, msg MsgNoOp) error {
	action := msg.GetAction()
	if len(action) == 0 {
		return nil
	}
	if !strings.EqualFold(action, "novault") {
		return nil
	}
	vault, err := h.mgr.Keeper().GetVault(ctx, msg.ObservedTx.ObservedPubKey)
	if err != nil {
		ctx.Logger().Error("fail to get vault", "error", err, "pub_key", msg.ObservedTx.ObservedPubKey)
		return nil
	}
	// subtract the coins from vault , as it has been added to
	vault.SubFunds(msg.ObservedTx.Tx.Coins)
	if err := h.mgr.Keeper().SetVault(ctx, vault); err != nil { // trunk-ignore(golangci-lint/govet): shadow
		ctx.Logger().Error("fail to save vault", "error", err)
	}
	return nil
}
