package thorchain

import (
	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/x/thorchain/types"
)

// ErrataTxQuorumHandler is to handle ErrataTxQuorum message
type ErrataTxQuorumHandler struct {
	mgr Manager
}

// NewErrataTxQuorumHandler create new instance of ErrataTxQuorumHandler
func NewErrataTxQuorumHandler(mgr Manager) ErrataTxQuorumHandler {
	return ErrataTxQuorumHandler{
		mgr: mgr,
	}
}

// Run is the main entry point to execute ErrataTxQuorum logic
func (h ErrataTxQuorumHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*types.MsgErrataTxQuorum)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg errata tx failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process MsgErrataTxQuorum", "error", err)
	}
	return result, err
}

func (h ErrataTxQuorumHandler) validate(ctx cosmos.Context, msg types.MsgErrataTxQuorum) error {
	return msg.ValidateBasic()
}

func (h ErrataTxQuorumHandler) handle(ctx cosmos.Context, msg types.MsgErrataTxQuorum) (*cosmos.Result, error) {
	active, err := h.mgr.Keeper().ListActiveValidators(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}

	er := msg.QuoErrata.ErrataTx

	voter, err := h.mgr.Keeper().GetErrataTxVoter(ctx, er.Id, er.Chain)
	if err != nil {
		return nil, err
	}

	defer func() {
		h.mgr.Keeper().SetErrataTxVoter(ctx, voter)
	}()

	signBz, err := er.GetSignablePayload()
	if err != nil {
		ctx.Logger().Error("fail to marshal errata tx sign payload", "error", err)
		return &cosmos.Result{}, nil
	}

	for _, att := range msg.QuoErrata.Attestations {
		accAddr, err := verifyQuorumAttestation(active, signBz, att)
		if err != nil {
			ctx.Logger().Error("fail to verify quorum errata tx attestation", "error", err)
			continue
		}

		if err := processErrataTxAttestation(ctx, h.mgr, &voter, accAddr, active, er, false); err != nil {
			ctx.Logger().Error("fail to process errata tx attestation", "error", err)
		}
	}

	return &cosmos.Result{}, nil
}
