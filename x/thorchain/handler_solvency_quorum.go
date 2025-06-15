package thorchain

import (
	"fmt"

	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/x/thorchain/types"
)

// SolvencyQuorumHandler is to process MsgSolvencyQuorum message from bifrost
// Bifrost constantly monitor the account balance , and report to THORNode
// If it detect that wallet is short of fund , much less than vault, the network should automatically halt trading
type SolvencyQuorumHandler struct {
	mgr Manager
}

// NewSolvencyQuorumHandler create a new instance of solvency handler
func NewSolvencyQuorumHandler(mgr Manager) SolvencyQuorumHandler {
	return SolvencyQuorumHandler{
		mgr: mgr,
	}
}

// Run is the main entry point to process MsgSolvencyQuorum
func (h SolvencyQuorumHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*types.MsgSolvencyQuorum)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg solvency failed validation", "error", err)
		return nil, err
	}
	return h.handle(ctx, *msg)
}

func (h SolvencyQuorumHandler) validate(ctx cosmos.Context, msg types.MsgSolvencyQuorum) error {
	return msg.ValidateBasic()
}

// handleCurrent is the logic to process MsgSolvencyQuorum, the feature works like this
//  1. Bifrost report MsgSolvencyQuorum to thornode , which is the balance of asgard wallet on each individual chain
//  2. once MsgSolvencyQuorum reach consensus , then the network compare the wallet balance against wallet
//     if wallet has less fund than asgard vault , and the gap is more than 1% , then the chain
//     that is insolvent will be halt
//  3. When chain is halt , bifrost will not observe inbound , and will not sign outbound txs until the issue has been investigated , and enabled it again using mimir
func (h SolvencyQuorumHandler) handle(ctx cosmos.Context, msg types.MsgSolvencyQuorum) (*cosmos.Result, error) {
	s := msg.QuoSolvency.Solvency

	ctx.Logger().Debug("handle Solvency request", "id", s.Id.String(), "signer", msg.Signer.String())

	k := h.mgr.Keeper()

	active, err := k.ListActiveValidators(ctx)
	if err != nil {
		return nil, wrapError(ctx, err, "fail to get list of active node accounts")
	}

	voter, err := k.GetSolvencyVoter(ctx, s.Id, s.Chain)
	if err != nil {
		return &cosmos.Result{}, fmt.Errorf("fail to get solvency voter, err: %w", err)
	}
	if voter.Empty() {
		voter = NewSolvencyVoter(s.Id, s.Chain, s.PubKey, s.Coins, s.Height)
	}

	defer func() {
		k.SetSolvencyVoter(ctx, voter)
	}()

	signBz, err := s.GetSignablePayload()
	if err != nil {
		ctx.Logger().Error("fail to marshal solvency sign payload", "error", err)
		return &cosmos.Result{}, nil
	}

	for _, att := range msg.QuoSolvency.Attestations {
		accAddr, err := verifyQuorumAttestation(active, signBz, att)
		if err != nil {
			ctx.Logger().Error("fail to verify quorum solvency attestation", "error", err)
			continue
		}

		if err := processSolvencyAttestation(ctx, h.mgr, &voter, accAddr, active, s, false); err != nil {
			ctx.Logger().Error("fail to process solvency attestation", "error", err)
		}
	}

	return &cosmos.Result{}, nil
}
