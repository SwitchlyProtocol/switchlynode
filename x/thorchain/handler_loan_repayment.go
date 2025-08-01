package thorchain

import (
	"fmt"

	"github.com/blang/semver"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

// LoanRepaymentHandler a handler to process bond
type LoanRepaymentHandler struct {
	mgr Manager
}

// NewLoanRepaymentHandler create new LoanRepaymentHandler
func NewLoanRepaymentHandler(mgr Manager) LoanRepaymentHandler {
	return LoanRepaymentHandler{
		mgr: mgr,
	}
}

// Run execute the handler
func (h LoanRepaymentHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgLoanRepayment)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgLoanRepayment",
		"owner", msg.Owner,
		"asset", msg.CollateralAsset,
		"coin", msg.Coin.String(),
		"from", msg.From.String(),
		"min_out", msg.MinOut.String(),
	)

	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg loan fail validation", "error", err)
		return nil, err
	}

	err := h.handle(ctx, *msg)
	if err != nil {
		ctx.Logger().Error("fail to process msg loan", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h LoanRepaymentHandler) validate(ctx cosmos.Context, msg MsgLoanRepayment) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h LoanRepaymentHandler) validateV3_0_0(ctx cosmos.Context, msg MsgLoanRepayment) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	// if a swap is required (ie the coin is NOT TOR), then ensure that the
	// from address passed in the msg matches the chain of the coin.Asset. This
	// is a sanity check that msg.From is not a btc address, while coin.Asset
	// is ETH.ETH
	if !msg.Coin.Asset.Equals(common.TOR) && !msg.From.IsChain(msg.Coin.Asset.GetChain()) {
		return fmt.Errorf("address %s does not match input coin %s", msg.From.String(), msg.Coin.String())
	}

	pauseLoans := h.mgr.Keeper().GetConfigInt64(ctx, constants.PauseLoans)
	if pauseLoans > 0 {
		return fmt.Errorf("loans are currently paused")
	}

	if !h.mgr.Keeper().PoolExist(ctx, msg.CollateralAsset) {
		ctx.Logger().Error("pool does not exist", "asset", msg.CollateralAsset)
		return fmt.Errorf("pool does not exist")
	}

	loan, err := h.mgr.Keeper().GetLoan(ctx, msg.CollateralAsset, msg.Owner)
	if err != nil {
		ctx.Logger().Error("fail to get loan", "error", err)
		return err
	}

	if loan.Collateral().IsZero() {
		return fmt.Errorf("loan contains no collateral to redeem")
	}

	maturity := h.mgr.Keeper().GetConfigInt64(ctx, constants.LoanRepaymentMaturity)
	if loan.LastOpenHeight+maturity > ctx.BlockHeight() {
		return fmt.Errorf("loan repayment is unavailable: loan hasn't reached maturity")
	}

	return nil
}

func (h LoanRepaymentHandler) handle(ctx cosmos.Context, msg MsgLoanRepayment) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h LoanRepaymentHandler) handleV3_0_0(ctx cosmos.Context, msg MsgLoanRepayment) error {
	// if the inbound asset is TOR, then lets repay the loan. If not, lets
	// swap first and try again later
	if msg.Coin.Asset.Equals(common.TOR) {
		return h.repay(ctx, msg)
	} else {
		return h.swap(ctx, msg)
	}
}

func (h LoanRepaymentHandler) repay(ctx cosmos.Context, msg MsgLoanRepayment) error {
	// collect data
	loan, err := h.mgr.Keeper().GetLoan(ctx, msg.CollateralAsset, msg.Owner)
	if err != nil {
		ctx.Logger().Error("fail to get loan", "error", err)
		return err
	}
	totalCollateral, err := h.mgr.Keeper().GetTotalCollateral(ctx, msg.CollateralAsset)
	if err != nil {
		return err
	}

	// update Loan record
	loan.DebtRepaid = loan.DebtRepaid.Add(msg.Coin.Amount)
	loan.LastRepayHeight = ctx.BlockHeight()

	// burn TOR coins
	if err = h.mgr.Keeper().SendFromModuleToModule(ctx, AsgardName, ModuleName, common.NewCoins(msg.Coin)); err != nil {
		ctx.Logger().Error("fail to move coins during loan repayment", "error", err)
		return err
	} else {
		if err = h.mgr.Keeper().BurnFromModule(ctx, ModuleName, msg.Coin); err != nil {
			ctx.Logger().Error("fail to burn coins during loan repayment", "error", err)
			return err
		}
		burnEvt := NewEventMintBurn(BurnSupplyType, msg.Coin.Asset.Native(), msg.Coin.Amount, "loan_repayment")
		if err = h.mgr.EventMgr().EmitEvent(ctx, burnEvt); err != nil {
			ctx.Logger().Error("fail to emit burn event", "error", err)
		}
	}

	// loan must be fully repaid to return collateral
	if !loan.Debt().IsZero() {
		h.mgr.Keeper().SetLoan(ctx, loan)

		// emit events and metrics
		evt := NewEventLoanRepayment(cosmos.ZeroUint(), msg.Coin.Amount, msg.CollateralAsset, msg.Owner, msg.TxID)
		if err = h.mgr.EventMgr().EmitEvent(ctx, evt); nil != err {
			ctx.Logger().Error("fail to emit repayment open event", "error", err)
		}

		return nil
	}

	redeem := loan.Collateral()
	// only return collateral when collateral is non-zero
	if redeem.IsZero() {
		return nil
	}

	loan.CollateralWithdrawn = loan.CollateralWithdrawn.Add(redeem)

	txID, ok := ctx.Value(constants.CtxLoanTxID).(common.TxID)
	if !ok {
		return fmt.Errorf("fail to get txid")
	}

	coins := common.NewCoins(common.NewCoin(msg.CollateralAsset.GetDerivedAsset(), redeem))

	// transfer derived asset from the lending to asgard before swap to L1 collateral
	err = h.mgr.Keeper().SendFromModuleToModule(ctx, LendingName, AsgardName, coins)
	if err != nil {
		ctx.Logger().Error("fail to send from lending to asgard", "error", err)
		return err
	}

	// Get streaming swaps interval to use for loan swap
	ssInterval := h.mgr.Keeper().GetConfigInt64(ctx, constants.LoanStreamingSwapsInterval)
	if ssInterval <= 0 || !msg.MinOut.IsZero() {
		ssInterval = 0
	}

	fakeGas := common.NewCoin(msg.Coin.Asset, cosmos.OneUint())
	// As this is to be a swap from derived asset which has been sent to AsgardName, the ToAddress should be AsgardName's address.
	tx := common.NewTx(txID, msg.Owner, common.NoopAddress, coins, common.Gas{fakeGas}, "noop")
	swapMsg := NewMsgSwap(tx, msg.CollateralAsset, msg.Owner, msg.MinOut, common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, uint64(ssInterval), msg.Signer)
	if ssInterval == 0 {
		handler := NewSwapHandler(h.mgr)
		if _, err = handler.Run(ctx, swapMsg); err != nil {
			ctx.Logger().Error("fail to make second swap when closing a loan", "error", err)
			return err
		}
	} else {
		if err = h.mgr.Keeper().SetSwapQueueItem(ctx, *swapMsg, 1); err != nil {
			ctx.Logger().Error("fail to add swap to queue", "error", err)
			return err
		}
	}

	// update kvstore
	h.mgr.Keeper().SetLoan(ctx, loan)
	h.mgr.Keeper().SetTotalCollateral(ctx, msg.CollateralAsset, common.SafeSub(totalCollateral, redeem))

	// emit events and metrics
	evt := NewEventLoanRepayment(redeem, msg.Coin.Amount, msg.CollateralAsset, msg.Owner, msg.TxID)
	if err = h.mgr.EventMgr().EmitEvent(ctx, evt); nil != err {
		ctx.Logger().Error("fail to emit loan repayment event", "error", err)
	}
	return nil
}

func (h LoanRepaymentHandler) swap(ctx cosmos.Context, msg MsgLoanRepayment) error {
	txID, ok := ctx.Value(constants.CtxLoanTxID).(common.TxID)
	if !ok {
		return fmt.Errorf("fail to get txid")
	}

	toAddress, ok := ctx.Value(constants.CtxLoanToAddress).(common.Address)
	// An empty ToAddress fails Tx validation,
	// and a querier quote or unit test has no provided ToAddress.
	// As this only affects emitted swap event contents, do not return an error.
	if !ok || toAddress.IsEmpty() {
		toAddress = "no to address available"
	}

	// Get streaming swaps interval to use for loan swap
	ssInterval := h.mgr.Keeper().GetConfigInt64(ctx, constants.LoanStreamingSwapsInterval)
	if ssInterval <= 0 || !msg.MinOut.IsZero() {
		ssInterval = 0
	}

	memo := fmt.Sprintf("loan-:%s:%s:%s", msg.CollateralAsset, msg.Owner, msg.MinOut)
	fakeGas := common.NewCoin(msg.Coin.Asset, cosmos.OneUint())
	tx := common.NewTx(txID, msg.From, toAddress, common.NewCoins(msg.Coin), common.Gas{fakeGas}, memo)
	swapMsg := NewMsgSwap(tx, common.TOR, common.NoopAddress, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, uint64(ssInterval), msg.Signer)
	if err := h.mgr.Keeper().SetSwapQueueItem(ctx, *swapMsg, 0); err != nil {
		ctx.Logger().Error("fail to add swap to queue", "error", err)
		return err
	}

	return nil
}
