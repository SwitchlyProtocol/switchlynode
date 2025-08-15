package switchly

import (
	"fmt"

	"github.com/blang/semver"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

// LoanOpenHandler a handler to process bond
type LoanOpenHandler struct {
	mgr Manager
}

// NewLoanOpenHandler create new LoanOpenHandler
func NewLoanOpenHandler(mgr Manager) LoanOpenHandler {
	return LoanOpenHandler{
		mgr: mgr,
	}
}

// Run execute the handler
func (h LoanOpenHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgLoanOpen)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("receive MsgLoanOpen",
		"owner", msg.Owner,
		"col_asset", msg.CollateralAsset,
		"col_amount", msg.CollateralAmount,
		"target_address", msg.TargetAddress,
		"target_asset", msg.TargetAsset,
		"affiliate", msg.AffiliateAddress,
		"affiliate_basis_points", msg.AffiliateBasisPoints,
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

func (h LoanOpenHandler) validate(ctx cosmos.Context, msg MsgLoanOpen) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.validateV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h LoanOpenHandler) validateV3_0_0(ctx cosmos.Context, msg MsgLoanOpen) error {
	if err := msg.ValidateBasic(); err != nil {
		return err
	}

	pauseLoans := h.mgr.Keeper().GetConfigInt64(ctx, constants.PauseLoans)
	if pauseLoans > 0 {
		return fmt.Errorf("loans are currently paused")
	}

	if msg.TargetAsset.IsTradeAsset() || msg.CollateralAsset.IsTradeAsset() {
		return fmt.Errorf("trade assets may not be used for loans")
	}

	if msg.TargetAsset.IsSecuredAsset() || msg.CollateralAsset.IsSecuredAsset() {
		return fmt.Errorf("secured assets may not be used for loans")
	}

	// ensure that while derived assets are disabled, borrower cannot receive a
	// derived asset as their debt
	enableDerived := h.mgr.Keeper().GetConfigInt64(ctx, constants.EnableDerivedAssets)
	if enableDerived == 0 && msg.TargetAsset.IsDerivedAsset() {
		return fmt.Errorf("cannot receive derived asset")
	}

	// Do not allow a network module as the target address.
	targetAccAddr, err := msg.TargetAddress.AccAddress()
	// A network module address would be resolvable,
	// so if not resolvable it should not be a network module address.
	if err == nil && IsModuleAccAddress(h.mgr.Keeper(), targetAccAddr) {
		return fmt.Errorf("a network module cannot be the target address of a loan open memo")
	}

	// Circuit Breaker: check if we're hit the max supply
	supply := h.mgr.Keeper().GetTotalSupply(ctx, common.SwitchNative)
	maxAmt := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaxSWITCHSupply)
	if maxAmt <= 0 {
		return fmt.Errorf("no max supply set")
	}
	if supply.GTE(cosmos.NewUint(uint64(maxAmt))) {
		return fmt.Errorf("loans are currently paused, due to switch supply cap (%d/%d)", supply.Uint64(), maxAmt)
	}

	// ensure collateral pool exists
	if !h.mgr.Keeper().PoolExist(ctx, msg.CollateralAsset) {
		return fmt.Errorf("collateral asset does not have a pool")
	}

	// The lending key for the ETH.ETH pool would be LENDING-SWITCHLY-ETH .
	key := "LENDING-" + msg.CollateralAsset.GetDerivedAsset().MimirString()
	val, err := h.mgr.Keeper().GetMimir(ctx, key)
	if err != nil {
		ctx.Logger().Error("fail to fetch LENDING key", "pool", msg.CollateralAsset.GetDerivedAsset().String(), "error", err)
		return err
	}
	if val <= 0 {
		return fmt.Errorf("Lending is not available for this collateral asset")
	}

	// convert collateral asset back to layer1 asset
	// NOTE: if the symbol of a derived asset isn't the chain, this won't work
	// (ie GAIA.ATOM)
	msg.CollateralAsset.Chain, err = common.NewChain(msg.CollateralAsset.Symbol.String())
	if err != nil {
		return err
	}

	totalCollateral, err := h.mgr.Keeper().GetTotalCollateral(ctx, msg.CollateralAsset)
	if err != nil {
		return err
	}
	totalSWITCH, err := h.getTotalLiquiditySWITCHLoanPools(ctx)
	if err != nil {
		return err
	}
	if totalSWITCH.IsZero() {
		return fmt.Errorf("no liquidity, lending unavailable")
	}
	lever := h.mgr.Keeper().GetConfigInt64(ctx, constants.LendingLever)
	runeBurnt := common.SafeSub(cosmos.NewUint(uint64(maxAmt)), supply)
	totalAvailableSWITCHForProtocol := common.GetSafeShare(cosmos.NewUint(uint64(lever)), cosmos.NewUint(10_000), runeBurnt) // calculate how much of that switch is available for loans
	if totalAvailableSWITCHForProtocol.IsZero() {
		return fmt.Errorf("no availability (0), lending unavailable")
	}
	pool, err := h.mgr.Keeper().GetPool(ctx, msg.CollateralAsset)
	if err != nil {
		ctx.Logger().Error("fail to get pool", "error", err)
		return err
	}
	totalAvailableSWITCHForPool := common.GetSafeShare(pool.BalanceSwitch, totalSWITCH, totalAvailableSWITCHForProtocol)
	totalAvailableAssetForPool := pool.SWITCHValueInAsset(totalAvailableSWITCHForPool)
	if totalCollateral.Add(msg.CollateralAmount).GT(totalAvailableAssetForPool) {
		return fmt.Errorf("no availability (%d/%d), lending unavailable", totalCollateral.Add(msg.CollateralAmount).Uint64(), totalAvailableAssetForPool.Uint64())
	}

	return nil
}

func (h LoanOpenHandler) handle(ctx cosmos.Context, msg MsgLoanOpen) error {
	version := h.mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return h.handleV3_0_0(ctx, msg)
	default:
		return errBadVersion
	}
}

func (h LoanOpenHandler) handleV3_0_0(ctx cosmos.Context, msg MsgLoanOpen) error {
	// if the inbound asset is SWITCHLY, then lets repay the loan. If not, lets
	// swap first and try again later
	if msg.CollateralAsset.IsDerivedAsset() {
		return h.openLoan(ctx, msg)
	} else {
		return h.swap(ctx, msg)
	}
}

func (h LoanOpenHandler) openLoan(ctx cosmos.Context, msg MsgLoanOpen) error {
	var err error
	zero := cosmos.ZeroUint()

	// convert collateral asset back to layer1 asset
	// NOTE: if the symbol of a derived asset isn't the chain, this won't work
	// (ie GAIA.ATOM)
	msg.CollateralAsset.Chain, err = common.NewChain(msg.CollateralAsset.Symbol.String())
	if err != nil {
		return err
	}

	pool, err := h.mgr.Keeper().GetPool(ctx, msg.CollateralAsset)
	if err != nil {
		ctx.Logger().Error("fail to get pool", "error", err)
		return err
	}
	loan, err := h.mgr.Keeper().GetLoan(ctx, msg.CollateralAsset, msg.Owner)
	if err != nil {
		ctx.Logger().Error("fail to get loan", "error", err)
		return err
	}
	totalCollateral, err := h.mgr.Keeper().GetTotalCollateral(ctx, msg.CollateralAsset)
	if err != nil {
		return err
	}

	// move derived asset collateral into lending module
	// TODO: on hard fork, change lending module to an actual module (created as account)
	lendingAcc := h.mgr.Keeper().GetModuleAccAddress(LendingName)
	collateral := common.NewCoin(msg.CollateralAsset.GetDerivedAsset(), msg.CollateralAmount)
	// trunk-ignore(golangci-lint/govet): shadow
	if err := h.mgr.Keeper().SendFromModuleToAccount(ctx, AsgardName, lendingAcc, common.NewCoins(collateral)); err != nil {
		return fmt.Errorf("fail to send collateral funds: %w", err)
	}

	// get configs
	enableDerived := h.mgr.Keeper().GetConfigInt64(ctx, constants.EnableDerivedAssets)

	// calculate CR
	cr, err := h.getPoolCR(ctx, pool, msg.CollateralAmount)
	if err != nil {
		return err
	}

	price := h.mgr.Keeper().DollarsPerSWITCH(ctx)
	if price.IsZero() {
		return fmt.Errorf("SWITCHLY price cannot be zero")
	}

	collateralValueInSWITCH := pool.AssetValueInSWITCH(msg.CollateralAmount)
	collateralValueInSWITCHLY := collateralValueInSWITCH.Mul(price).QuoUint64(1e8)
	debt := collateralValueInSWITCHLY.Quo(cr).MulUint64(10_000)
	ctx.Logger().Info("Loan Details", "collateral", common.NewCoin(msg.CollateralAsset, msg.CollateralAmount), "debt", debt.Uint64(), "switch price", price.Uint64(), "colSWITCH", collateralValueInSWITCH.Uint64(), "colSWITCHLY", collateralValueInSWITCHLY.Uint64())

	// sanity checks
	if debt.IsZero() {
		return fmt.Errorf("debt cannot be zero")
	}

	// if the user has over-repayed the loan, credit the difference on the next open
	cumulativeDebt := debt
	if loan.DebtRepaid.GT(loan.DebtIssued) {
		cumulativeDebt = cumulativeDebt.Add(loan.DebtRepaid.Sub(loan.DebtIssued))
	}

	// update Loan record
	loan.DebtIssued = loan.DebtIssued.Add(cumulativeDebt)
	loan.CollateralDeposited = loan.CollateralDeposited.Add(msg.CollateralAmount)
	loan.LastOpenHeight = ctx.BlockHeight()

	if msg.TargetAsset.Equals(common.SWITCHLY) && enableDerived > 0 {
		toi := TxOutItem{
			Chain:      msg.TargetAsset.GetChain(),
			ToAddress:  msg.TargetAddress,
			Coin:       common.NewCoin(common.SWITCHLY, cumulativeDebt),
			ModuleName: ModuleName,
		}
		ok, err := h.mgr.TxOutStore().TryAddTxOutItem(ctx, h.mgr, toi, zero) // trunk-ignore(golangci-lint/govet): shadow
		if err != nil {
			return err
		}
		if !ok {
			return errFailAddOutboundTx
		}
	} else {
		txID, ok := ctx.Value(constants.CtxLoanTxID).(common.TxID)
		if !ok {
			return fmt.Errorf("fail to get txid")
		}

		torCoin := common.NewCoin(common.SWITCHLY, cumulativeDebt)

		if err := h.mgr.Keeper().MintToModule(ctx, ModuleName, torCoin); err != nil { // trunk-ignore(golangci-lint/govet): shadow
			return fmt.Errorf("fail to mint loan tor debt: %w", err)
		}
		mintEvt := NewEventMintBurn(MintSupplyType, torCoin.Asset.Native(), torCoin.Amount, "swap")
		if err := h.mgr.EventMgr().EmitEvent(ctx, mintEvt); err != nil {
			ctx.Logger().Error("fail to emit mint event", "error", err)
		}

		if err := h.mgr.Keeper().SendFromModuleToModule(ctx, ModuleName, AsgardName, common.NewCoins(torCoin)); err != nil {
			return fmt.Errorf("fail to send SWITCHLY vault funds: %w", err)
		}

		// Get streaming swaps interval to use for loan swap
		ssInterval := h.mgr.Keeper().GetConfigInt64(ctx, constants.LoanStreamingSwapsInterval)
		if ssInterval <= 0 || !msg.MinOut.IsZero() {
			ssInterval = 0
		}

		// As this is to be a swap from SWITCHLY which has been sent to AsgardName, the ToAddress should be AsgardName's address.
		tx := common.NewTx(txID, common.NoopAddress, common.NoopAddress, common.NewCoins(torCoin), nil, "noop")
		// we do NOT pass affiliate info here as it was already taken out on the swap of the collateral to derived asset
		swapMsg := NewMsgSwap(tx, msg.TargetAsset, msg.TargetAddress, msg.MinOut, common.NoAddress, zero, msg.Aggregator, msg.AggregatorTargetAddress, &msg.AggregatorTargetLimit, 0, 0, uint64(ssInterval), msg.Signer)
		if ssInterval == 0 {
			handler := NewSwapHandler(h.mgr)
			if _, err := handler.Run(ctx, swapMsg); err != nil {
				ctx.Logger().Error("fail to make second swap when opening a loan", "error", err)
				return err
			}
		} else {
			if err := h.mgr.Keeper().SetSwapQueueItem(ctx, *swapMsg, 1); err != nil {
				ctx.Logger().Error("fail to add swap to queue", "error", err)
				return err
			}
		}
	}

	// update kvstore
	h.mgr.Keeper().SetLoan(ctx, loan)
	h.mgr.Keeper().SetTotalCollateral(ctx, msg.CollateralAsset, totalCollateral.Add(msg.CollateralAmount))

	// emit events and metrics
	evt := NewEventLoanOpen(msg.CollateralAmount, cr, debt, msg.CollateralAsset, msg.TargetAsset, msg.Owner, msg.TxID)
	if err := h.mgr.EventMgr().EmitEvent(ctx, evt); nil != err {
		ctx.Logger().Error("fail to emit loan open event", "error", err)
	}

	return nil
}

func (h LoanOpenHandler) getPoolCR(ctx cosmos.Context, pool Pool, collateralAmount cosmos.Uint) (cosmos.Uint, error) {
	minCR := h.mgr.Keeper().GetConfigInt64(ctx, constants.MinCR)
	maxCR := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaxCR)
	lever := h.mgr.Keeper().GetConfigInt64(ctx, constants.LendingLever)

	currentSWITCHSupply := h.mgr.Keeper().GetTotalSupply(ctx, common.SwitchNative)
	maxSWITCHSupply := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaxSWITCHSupply)
	if maxSWITCHSupply <= 0 {
		return cosmos.ZeroUint(), fmt.Errorf("no max supply set")
	}
	runeBurnt := common.SafeSub(cosmos.NewUint(uint64(maxSWITCHSupply)), currentSWITCHSupply)
	totalAvailableSWITCHForProtocol := common.GetSafeShare(cosmos.NewUint(uint64(lever)), cosmos.NewUint(10_000), runeBurnt) // calculate how much of that switch is available for loans
	if totalAvailableSWITCHForProtocol.IsZero() {
		return cosmos.ZeroUint(), fmt.Errorf("no availability (0), lending unavailable")
	}

	totalCollateral, err := h.mgr.Keeper().GetTotalCollateral(ctx, pool.Asset)
	if err != nil {
		return cosmos.ZeroUint(), err
	}

	totalSWITCH, err := h.getTotalLiquiditySWITCHLoanPools(ctx)
	if err != nil {
		return cosmos.ZeroUint(), err
	}
	if totalSWITCH.IsZero() {
		return cosmos.ZeroUint(), fmt.Errorf("no liquidity, lending unavailable")
	}

	totalAvailableSWITCHForPool := common.GetSafeShare(pool.BalanceSwitch, totalSWITCH, totalAvailableSWITCHForProtocol)
	totalAvailableAssetForPool := pool.SWITCHValueInAsset(totalAvailableSWITCHForPool)
	if totalCollateral.Add(collateralAmount).GT(totalAvailableAssetForPool) {
		return cosmos.ZeroUint(), fmt.Errorf("no availability (%d/%d), lending unavailable", totalCollateral.Add(collateralAmount).Uint64(), totalAvailableAssetForPool.Uint64())
	}
	cr := h.calcCR(totalCollateral.Add(collateralAmount), totalAvailableAssetForPool, minCR, maxCR)

	return cr, nil
}

func (h LoanOpenHandler) calcCR(a, b cosmos.Uint, minCR, maxCR int64) cosmos.Uint {
	// (maxCR - minCR) / (b / a) + minCR
	// NOTE: a should include the collateral currently being deposited
	crCalc := cosmos.NewUint(uint64(maxCR - minCR))
	cr := common.GetUncappedShare(a, b, crCalc)
	return cr.AddUint64(uint64(minCR))
}

func (h LoanOpenHandler) swap(ctx cosmos.Context, msg MsgLoanOpen) error {
	txID, ok := ctx.Value(constants.CtxLoanTxID).(common.TxID)
	if !ok {
		return fmt.Errorf("fail to get txid")
	}
	// ensure TxID does NOT have a collision with another swap, this could
	// happen if the user submits two identical loan requests in the same
	// block
	// trunk-ignore(golangci-lint/govet): shadow
	if ok := h.mgr.Keeper().HasSwapQueueItem(ctx, txID, 0); ok {
		return fmt.Errorf("txn hash conflict")
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

	collateral := common.NewCoin(msg.CollateralAsset, msg.CollateralAmount)
	maxAffPoints := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaxAffiliateFeeBasisPoints)

	// only take affiliate fee if parameters are set and it's the original swap (not the derived asset swap)
	if !msg.AffiliateBasisPoints.IsZero() && msg.AffiliateBasisPoints.LTE(cosmos.NewUint(uint64(maxAffPoints))) && !msg.AffiliateAddress.IsEmpty() && !msg.CollateralAsset.IsNative() {
		newAmt, err := h.handleAffiliateSwap(ctx, msg, collateral)
		if err != nil {
			ctx.Logger().Error("fail to handle affiliate swap", "error", err)
		} else {
			collateral.Amount = newAmt
		}
	}

	memo := fmt.Sprintf("loan+:%s:%s:%d:%s:%d:%s:%s:%d", msg.TargetAsset, msg.TargetAddress, msg.MinOut.Uint64(), msg.AffiliateAddress, msg.AffiliateBasisPoints.Uint64(), msg.Aggregator, msg.AggregatorTargetAddress, msg.AggregatorTargetLimit.Uint64())
	fakeGas := common.NewCoin(msg.CollateralAsset.GetChain().GetGasAsset(), cosmos.OneUint())
	tx := common.NewTx(txID, msg.Owner, toAddress, common.NewCoins(collateral), common.Gas{fakeGas}, memo)
	swapMsg := NewMsgSwap(tx, msg.CollateralAsset.GetDerivedAsset(), common.NoopAddress, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, uint64(ssInterval), msg.Signer)
	if err := h.mgr.Keeper().SetSwapQueueItem(ctx, *swapMsg, 0); err != nil {
		ctx.Logger().Error("fail to add swap to queue", "error", err)
		return err
	}

	return nil
}

// handleAffiliateSwap handles the affiliate swap for the loan open and returns updated
// collateral amount for the loan with affiliate amount deducted
func (h LoanOpenHandler) handleAffiliateSwap(ctx cosmos.Context, msg MsgLoanOpen, collateral common.Coin) (cosmos.Uint, error) {
	// Setup affiliate swap
	affAmt := common.GetSafeShare(
		msg.AffiliateBasisPoints,
		cosmos.NewUint(constants.MaxBasisPts),
		msg.CollateralAmount,
	)

	affCoin := common.NewCoin(msg.CollateralAsset, affAmt)
	gasCoin := common.NewCoin(msg.CollateralAsset.GetChain().GetGasAsset(), cosmos.OneUint())
	fakeTx := common.NewTx(msg.TxID, common.NoopAddress, common.NoopAddress, common.NewCoins(affCoin), common.Gas{gasCoin}, "noop")
	affiliateSwap := NewMsgSwap(fakeTx, common.SwitchNative, msg.AffiliateAddress, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, 0, 0, 0, msg.Signer)

	var affSwitchlyname *types.SWITCHName
	voter, err := h.mgr.Keeper().GetObservedTxInVoter(ctx, msg.TxID)
	if err == nil {
		// trunk-ignore(golangci-lint/govet): shadow
		memo, err := ParseMemoWithSWITCHNames(ctx, h.mgr.Keeper(), voter.Tx.Tx.Memo)
		if err != nil {
			ctx.Logger().Error("fail to parse memo", "error", err)
		}
		affSwitchlyname = memo.GetAffiliateSWITCHName()
	}

	// PreferredAsset set, swap to the AffiliateCollector Module + check if the
	// preferred asset swap should be triggered
	if affSwitchlyname != nil && !affSwitchlyname.PreferredAsset.IsEmpty() {
		// trunk-ignore(golangci-lint/govet): shadow
		affcol, err := h.mgr.Keeper().GetAffiliateCollector(ctx, affSwitchlyname.Owner)
		if err != nil {
			return collateral.Amount, err
		}
		affColAddress, err := h.mgr.Keeper().GetModuleAddress(AffiliateCollectorName)
		if err != nil {
			return collateral.Amount, err
		}
		// Set AffiliateCollector Module as destination and populate the AffiliateAddress
		// so that the swap handler can increment the emitted SWITCH for the affiliate in
		// the AffiliateCollector KVStore.
		affiliateSwap.Destination = affColAddress
		affiliateSwap.AffiliateAddress = msg.AffiliateAddress
		// Need to set the memo as a normal swap, so that the swap handler doesn't run the
		// open loan handler for the affiliate swaps
		affiliateSwap.Tx.Memo = NewSwapMemo(ctx, h.mgr, common.SwitchNative, msg.AffiliateAddress, cosmos.ZeroUint(), affSwitchlyname.Name, cosmos.ZeroUint())

		// Check if accrued SWITCH is 100x current outbound fee of preferred asset, if
		// so trigger the preferred asset swap
		ofSWITCH, err := h.mgr.GasMgr().GetAssetOutboundFee(ctx, affSwitchlyname.PreferredAsset, true)
		if err != nil {
			ctx.Logger().Error("fail to get outbound fee", "err", err)
		}
		multiplier := h.mgr.Keeper().GetConfigInt64(ctx, constants.PreferredAssetOutboundFeeMultiplier)
		threshold := ofSWITCH.Mul(cosmos.NewUint(uint64(multiplier)))
		if err == nil && affcol.SwitchAmount.GT(threshold) {
			if err = triggerPreferredAssetSwap(ctx, h.mgr, msg.AffiliateAddress, msg.TxID, *affSwitchlyname, affcol, 3); err != nil {
				ctx.Logger().Error("fail to queue preferred asset swap", "switchlyname", affSwitchlyname.Name, "err", err)
			}
		}
	}

	// If the affiliate swap would exceed the native tx fee, add it to the queue
	if willSwapOutputExceedLimitAndFees(ctx, h.mgr, *affiliateSwap) {
		// trunk-ignore(golangci-lint/govet): shadow
		if err := h.mgr.Keeper().SetSwapQueueItem(ctx, *affiliateSwap, 2); err != nil {
			return collateral.Amount, fmt.Errorf("fail to add affiliate swap to queue: %w", err)
		}
		collateral.Amount = common.SafeSub(collateral.Amount, affAmt)
	}

	return collateral.Amount, nil
}

// getTotalLiquiditySWITCH we have in all pools
func (h LoanOpenHandler) getTotalLiquiditySWITCHLoanPools(ctx cosmos.Context) (cosmos.Uint, error) {
	pools, err := h.mgr.Keeper().GetPools(ctx)
	if err != nil {
		return cosmos.ZeroUint(), fmt.Errorf("fail to get pools from data store: %w", err)
	}
	total := cosmos.ZeroUint()
	for _, p := range pools {
		// ignore suspended pools
		if p.Status == PoolSuspended {
			continue
		}
		if p.Asset.IsSyntheticAsset() {
			continue
		}
		if p.Asset.IsDerivedAsset() {
			continue
		}

		key := "LENDING-" + p.Asset.GetDerivedAsset().MimirString()
		// trunk-ignore(golangci-lint/govet): shadow
		val, err := h.mgr.Keeper().GetMimir(ctx, key)
		if err != nil {
			continue
		}
		if val <= 0 {
			continue
		}
		total = total.Add(p.BalanceSwitch)
	}
	return total, nil
}

func (h LoanOpenHandler) GetLoanCollateralRemainingForPool(ctx cosmos.Context, pool Pool) (cosmos.Uint, error) {
	lever := h.mgr.Keeper().GetConfigInt64(ctx, constants.LendingLever)

	currentSWITCHSupply := h.mgr.Keeper().GetTotalSupply(ctx, common.SwitchNative)
	maxSWITCHSupply := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaxSWITCHSupply)
	if maxSWITCHSupply <= 0 {
		return cosmos.ZeroUint(), fmt.Errorf("no max supply set")
	}
	runeBurnt := common.SafeSub(cosmos.NewUint(uint64(maxSWITCHSupply)), currentSWITCHSupply)
	// calculate total switch available for loans
	totalAvailableSWITCHForProtocol := common.GetSafeShare(cosmos.NewUint(uint64(lever)), cosmos.NewUint(constants.MaxBasisPts), runeBurnt)
	totalCollateral, err := h.mgr.Keeper().GetTotalCollateral(ctx, pool.Asset)
	if err != nil {
		return cosmos.ZeroUint(), err
	}

	totalSWITCH, err := h.getTotalLiquiditySWITCHLoanPools(ctx)
	if err != nil {
		return cosmos.ZeroUint(), err
	}

	totalAvailableSWITCHForPool := common.GetSafeShare(pool.BalanceSwitch, totalSWITCH, totalAvailableSWITCHForProtocol)
	totalAvailableAssetForPool := pool.SWITCHValueInAsset(totalAvailableSWITCHForPool)

	loanCollateralRemainingForPool := common.SafeSub(totalAvailableAssetForPool, totalCollateral)

	return loanCollateralRemainingForPool, nil
}
