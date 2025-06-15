package thorchain

import (
	"context"
	"fmt"
	"runtime"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"gitlab.com/thorchain/thornode/v3/x/thorchain/types"
)

type msgServer struct {
	mgr Manager
}

var _ types.MsgServer = msgServer{}

// NewMsgServerImpl returns an implementation of the module MsgServer interface.
func NewMsgServerImpl(mgr Manager) types.MsgServer {
	return &msgServer{mgr: mgr}
}

func (ms msgServer) Ban(goCtx context.Context, msg *types.MsgBan) (*types.MsgEmpty, error) {
	handler := NewBanHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) Deposit(goCtx context.Context, msg *types.MsgDeposit) (*types.MsgEmpty, error) {
	handler := NewDepositHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ErrataTx(goCtx context.Context, msg *types.MsgErrataTx) (*types.MsgEmpty, error) {
	handler := NewErrataTxHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ErrataTxQuorum(goCtx context.Context, msg *types.MsgErrataTxQuorum) (*types.MsgEmpty, error) {
	handler := NewErrataTxQuorumHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) Mimir(goCtx context.Context, msg *types.MsgMimir) (*types.MsgEmpty, error) {
	handler := NewMimirHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) NetworkFee(goCtx context.Context, msg *types.MsgNetworkFee) (*types.MsgEmpty, error) {
	handler := NewNetworkFeeHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) NetworkFeeQuorum(goCtx context.Context, msg *types.MsgNetworkFeeQuorum) (*types.MsgEmpty, error) {
	handler := NewNetworkFeeQuorumHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) NodePauseChain(goCtx context.Context, msg *types.MsgNodePauseChain) (*types.MsgEmpty, error) {
	handler := NewNodePauseChainHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ObservedTxIn(goCtx context.Context, msg *types.MsgObservedTxIn) (*types.MsgEmpty, error) {
	handler := NewObservedTxInHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ObservedTxOut(goCtx context.Context, msg *types.MsgObservedTxOut) (*types.MsgEmpty, error) {
	handler := NewObservedTxOutHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ObservedTxQuorum(goCtx context.Context, msg *types.MsgObservedTxQuorum) (*types.MsgEmpty, error) {
	handler := NewObservedTxQuorumHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ThorSend(goCtx context.Context, msg *types.MsgSend) (*types.MsgEmpty, error) {
	handler := NewSendHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) SetIPAddress(goCtx context.Context, msg *types.MsgSetIPAddress) (*types.MsgEmpty, error) {
	handler := NewIPAddressHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) SetNodeKeys(goCtx context.Context, msg *types.MsgSetNodeKeys) (*types.MsgEmpty, error) {
	handler := NewSetNodeKeysHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) Solvency(goCtx context.Context, msg *types.MsgSolvency) (*types.MsgEmpty, error) {
	handler := NewSolvencyHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) SolvencyQuorum(goCtx context.Context, msg *types.MsgSolvencyQuorum) (*types.MsgEmpty, error) {
	handler := NewSolvencyQuorumHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) TssKeysignFail(goCtx context.Context, msg *types.MsgTssKeysignFail) (*types.MsgEmpty, error) {
	handler := NewTssKeysignHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) TssPool(goCtx context.Context, msg *types.MsgTssPool) (*types.MsgEmpty, error) {
	handler := NewTssHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) SetVersion(goCtx context.Context, msg *types.MsgSetVersion) (*types.MsgEmpty, error) {
	handler := NewVersionHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ProposeUpgrade(goCtx context.Context, msg *types.MsgProposeUpgrade) (*types.MsgEmpty, error) {
	handler := NewProposeUpgradeHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) ApproveUpgrade(goCtx context.Context, msg *types.MsgApproveUpgrade) (*types.MsgEmpty, error) {
	handler := NewApproveUpgradeHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func (ms msgServer) RejectUpgrade(goCtx context.Context, msg *types.MsgRejectUpgrade) (*types.MsgEmpty, error) {
	handler := NewRejectUpgradeHandler(ms.mgr)
	return externalHandler(goCtx, handler, msg)
}

func externalHandler(goCtx context.Context, handler MsgHandler, msg sdk.Msg) (_ *types.MsgEmpty, err error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	defer func() {
		if r := recover(); r != nil {
			// print stack
			stack := make([]byte, 1024)
			length := runtime.Stack(stack, true)
			ctx.Logger().Error("panic", "msg", msg)
			fmt.Println(string(stack[:length]))
			err = fmt.Errorf("panic: %v", r)
		}
	}()

	result, err := handler.Run(ctx, msg)

	if result != nil && result.Size() > 0 {
		return nil, fmt.Errorf("external handler, handler returned non-empty result, %s", msg)
	}
	if err != nil {
		if _, code, _ := errorsmod.ABCIInfo(err, false); code == 1 {
			// This would be redacted, so wrap it.
			err = errorsmod.Wrap(errInternal, err.Error())
		}
		return nil, err
	}

	return &types.MsgEmpty{}, err
}
