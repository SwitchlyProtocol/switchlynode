package thorchain

import (
	"context"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

// RagnarokHandler process MsgRagnarok
type RagnarokHandler struct {
	mgr Manager
}

// NewRagnarokHandler create a new instance of RagnarokHandler
func NewRagnarokHandler(mgr Manager) RagnarokHandler {
	return RagnarokHandler{
		mgr: mgr,
	}
}

// Run is the main entry point of ragnarok handler
func (h RagnarokHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgRagnarok)
	if !ok {
		return nil, errInvalidMessage
	}
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("MsgRagnarok failed validation", "error", err)
		return nil, err
	}
	result, err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process MsgRagnarok", "error", err)
	}
	return result, err
}

func (h RagnarokHandler) validate(ctx cosmos.Context, msg MsgRagnarok) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errInvalidVersion
	}
}

func (h RagnarokHandler) validateV3_0_0(ctx cosmos.Context, msg MsgRagnarok) error {
	return msg.ValidateBasic()
}

func (h RagnarokHandler) handle(ctx cosmos.Context, msg MsgRagnarok) (*cosmos.Result, error) {
	ctx.Logger().Info("receive MsgRagnarok", "request tx hash", msg.Tx.Tx.ID)
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return nil, errBadVersion
	}
}

func (h RagnarokHandler) slash(ctx cosmos.Context, tx ObservedTx) error {
	toSlash := make(common.Coins, len(tx.Tx.Coins))
	copy(toSlash, tx.Tx.Coins)
	toSlash = toSlash.Add(tx.Tx.Gas.ToCoins()...)

	ctx = ctx.WithContext(context.WithValue(ctx.Context(), constants.CtxMetricLabels, []metrics.Label{
		telemetry.NewLabel("reason", "failed_ragnarok"),
		telemetry.NewLabel("chain", string(tx.Tx.Chain)),
	}))

	return h.mgr.Slasher().SlashVault(ctx, tx.ObservedPubKey, toSlash, h.mgr)
}

func (h RagnarokHandler) handleV3_0_0(ctx cosmos.Context, msg MsgRagnarok) (*cosmos.Result, error) {
	// for ragnarok on switchlyprotocol ,
	if msg.Tx.Tx.Chain.Equals(common.SWITCHLYChain) {
		return &cosmos.Result{}, nil
	}
	shouldSlash := true
	signingTransPeriod := h.mgr.GetConstants().GetInt64Value(constants.SigningTransactionPeriod)
	decrementedPendingRagnarok := false
	for height := msg.BlockHeight; height <= ctx.BlockHeight(); height += signingTransPeriod {
		// update txOut record with our TxID that sent funds out of the pool
		txOut, err := h.mgr.Keeper().GetTxOut(ctx, height)
		if err != nil {
			return nil, ErrInternal(err, "unable to get txOut record")
		}
		for i, tx := range txOut.TxArray {
			// ragnarok is the memo used by switchlyprotocol to identify fund returns to
			// bonders, LPs, and reserve contributors.
			// it use ragnarok:{block height} to mark a tx out caused by the ragnarok protocol
			// this type of tx out is special, because it doesn't have relevant tx
			// in to trigger it, it is trigger by switchlyprotocol itself.

			fromAddress, _ := tx.VaultPubKey.GetAddress(tx.Chain)

			if tx.InHash.Equals(common.BlankTxID) &&
				tx.OutHash.IsEmpty() &&
				tx.ToAddress.Equals(msg.Tx.Tx.ToAddress) &&
				fromAddress.Equals(msg.Tx.Tx.FromAddress) {

				matchCoin := msg.Tx.Tx.Coins.EqualsEx(common.Coins{tx.Coin})
				// when outbound is gas asset
				if !matchCoin && tx.Coin.Asset.Equals(tx.Chain.GetGasAsset()) {
					asset := tx.Chain.GetGasAsset()
					intendToSpend := tx.Coin.Amount.Add(tx.MaxGas.ToCoins().GetCoin(asset).Amount)
					actualSpend := msg.Tx.Tx.Coins.GetCoin(asset).Amount.Add(msg.Tx.Tx.Gas.ToCoins().GetCoin(asset).Amount)
					if intendToSpend.Equal(actualSpend) {
						maxGasAmt := tx.MaxGas.ToCoins().GetCoin(asset).Amount
						realGasAmt := msg.Tx.Tx.Gas.ToCoins().GetCoin(asset).Amount
						if maxGasAmt.GTE(realGasAmt) {
							matchCoin = true
							ctx.Logger().Info("override match coin", "intend to spend", intendToSpend, "actual spend", actualSpend, "max gas", maxGasAmt, "actual gas", realGasAmt)
						}
						// the network didn't charge fee when it is ragnarok , thus it doesn't need to adjust gas
					}
				}
				if !matchCoin {
					continue
				}
				txOut.TxArray[i].OutHash = msg.Tx.Tx.ID
				shouldSlash = false
				if err := h.mgr.Keeper().SetTxOut(ctx, txOut); nil != err { // trunk-ignore(golangci-lint/govet): shadow
					return nil, ErrInternal(err, "fail to save tx out")
				}
				if !decrementedPendingRagnarok {
					pending, err := h.mgr.Keeper().GetRagnarokPending(ctx) // trunk-ignore(golangci-lint/govet): shadow
					if err != nil {
						ctx.Logger().Error("fail to get ragnarok pending", "error", err)
					} else {
						h.mgr.Keeper().SetRagnarokPending(ctx, pending-1)
						ctx.Logger().Info("remaining ragnarok transaction", "count", pending-1)
					}
					decrementedPendingRagnarok = true
				}
				break

			}
		}
	}

	if shouldSlash {
		if err := h.slash(ctx, msg.Tx); err != nil {
			return nil, ErrInternal(err, "fail to slash account")
		}
	}

	if err := h.mgr.Keeper().SetLastSignedHeight(ctx, msg.BlockHeight); err != nil {
		ctx.Logger().Info("fail to update last signed height", "error", err)
	}

	return &cosmos.Result{}, nil
}
