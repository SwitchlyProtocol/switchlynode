package thorchain

import (
	"errors"
	"fmt"
	"strings"

	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

// AddLiquidityHandler is to handle add liquidity
type AddLiquidityHandler struct {
	mgr Manager
}

// NewAddLiquidityHandler create a new instance of AddLiquidityHandler
func NewAddLiquidityHandler(mgr Manager) AddLiquidityHandler {
	return AddLiquidityHandler{
		mgr: mgr,
	}
}

// Run execute the handler
func (h AddLiquidityHandler) Run(ctx cosmos.Context, m cosmos.Msg) (*cosmos.Result, error) {
	msg, ok := m.(*MsgAddLiquidity)
	if !ok {
		return nil, errInvalidMessage
	}
	ctx.Logger().Info("received add liquidity request",
		"asset", msg.Asset.String(),
		"tx", msg.Tx)
	if err := h.validate(ctx, *msg); err != nil {
		ctx.Logger().Error("msg add liquidity fail validation", "error", err)
		return nil, err
	}

	if err := h.handle(ctx, *msg); err != nil {
		ctx.Logger().Error("fail to process msg add liquidity", "error", err)
		return nil, err
	}

	return &cosmos.Result{}, nil
}

func (h AddLiquidityHandler) validate(ctx cosmos.Context, msg MsgAddLiquidity) error {
	if !msg.Tx.ID.IsBlank() { // don't validate tx if internal txn
		if err := msg.ValidateBasic(); err != nil {
			ctx.Logger().Error(err.Error())
			return errAddLiquidityFailValidation
		}
	}

	if msg.Asset.IsTradeAsset() {
		return fmt.Errorf("asset cannot be a trade asset")
	}

	if msg.Asset.IsSecuredAsset() {
		return fmt.Errorf("asset cannot be a secured asset")
	}

	// TODO on hard fork move network check to ValidateBasic
	if !msg.AssetAddress.IsEmpty() {
		if !common.CurrentChainNetwork.SoftEquals(msg.AssetAddress.GetNetwork(msg.AssetAddress.GetChain())) {
			return fmt.Errorf("address(%s) is not same network", msg.AssetAddress)
		}
	}

	if h.mgr.Keeper().IsRagnarok(ctx, []common.Asset{msg.Asset}) {
		return fmt.Errorf("cannot add liquidity to Ragnaroked pool (%s)", msg.Asset.String())
	}

	// Note that GetChain() without GetLayer1Asset() would indicate SwitchlyProtocol for synthetic assets.
	gasAsset := msg.Asset.GetLayer1Asset().GetChain().GetGasAsset()
	// Even if a destination gas asset pool is empty, the first add liquidity has to be symmetrical,
	// and so there is no need to check at this stage for whether the addition is of RUNE or Asset or with needsSwap.
	if !msg.Asset.Equals(gasAsset) && !msg.Asset.IsTCY() && !msg.Asset.IsRUJI() {
		gasPool, err := h.mgr.Keeper().GetPool(ctx, gasAsset)
		// Note that for a synthetic asset msg.Asset.Chain (unlike msg.Asset.GetChain())
		// is intentionally used to be the external chain rather than SWITCHLY.
		// Any destination asset starting with SWITCHLY should be rejected for no SWITCHLY.SWITCH
		// gas asset pool existing.
		if err != nil {
			return ErrInternal(err, "fail to get gas pool")
		}
		// Note that NewPool from GetPool would return a pool with status;
		// use IsEmpty to check for prior existence.
		if gasPool.IsEmpty() {
			return fmt.Errorf("asset (%s)'s gas asset pool (%s) does not exist yet", msg.Asset.String(), gasAsset.String())
		}
	}

	if msg.Asset.IsDerivedAsset() {
		return fmt.Errorf("asset cannot be a derived asset")
	}

	// Check if user is trying to deposit into savers
	// At present, savers are authorized for:
	// * layer 1 assets  (BTC, ETH, etc.)
	// * stable pools (as determined by TOR anchor pool settings)
	if msg.Asset.IsSyntheticAsset() {
		// Check if the asset is in the anchor pools
		isAnchorAsset := false
		for _, asset := range h.mgr.Keeper().GetAnchors(ctx, common.TOR) {
			if msg.Asset.GetLayer1Asset().Equals(asset) {
				isAnchorAsset = true
				break
			}
		}
		// Saver asset must be either L1 gas asset or isAnchorAsset
		if !msg.Asset.GetLayer1Asset().IsGasAsset() && !isAnchorAsset {
			return fmt.Errorf("asset must be a gas asset or TOR anchor pool")
		}
		// Ensure that the message is being sent from an address on a chain that matches the L1 asset chain
		if !msg.AssetAddress.IsChain(msg.Asset.GetLayer1Asset().GetChain()) {
			return fmt.Errorf("asset address must be layer1 chain")
		}
		// Savers vaults are not currently enabled for SWITCH
		if !msg.RuneAmount.IsZero() {
			return fmt.Errorf("cannot deposit switch into a vault")
		}
	}

	if !msg.RuneAddress.IsEmpty() && !msg.RuneAddress.IsChain(common.SWITCHLYChain) {
		ctx.Logger().Error("switch address must be Switchly")
		return errAddLiquidityFailValidation
	}

	if !msg.AssetAddress.IsEmpty() {
		// If the needsSwap check disallows a cross-chain AssetAddress,
		// a position with pending RUNE cannot be completed with Asset,
		// so fail validation here if the AssetAddress chain is different from the Asset's.
		if !msg.AssetAddress.IsChain(msg.Asset.GetLayer1Asset().GetChain()) {
			return errAddLiquidityMismatchAddr
		}

		polAddress, err := h.mgr.Keeper().GetModuleAddress(ReserveName)
		if err != nil {
			return err
		}
		if msg.RuneAddress.Equals(polAddress) {
			return fmt.Errorf("pol lp cannot have asset address")
		}
	}

	// check if swap meets standards
	if h.needsSwap(msg) {
		if !msg.Asset.IsSyntheticAsset() {
			return fmt.Errorf("swap & add liquidity is only available for synthetic pools")
		}
		if !msg.Asset.GetLayer1Asset().Equals(msg.Tx.Coins[0].Asset) {
			return fmt.Errorf("deposit asset must be the layer1 equivalent for the synthetic asset")
		}
	}

	pool, err := h.mgr.Keeper().GetPool(ctx, msg.Asset)
	if err != nil {
		return ErrInternal(err, "fail to get pool")
	}
	if err = pool.EnsureValidPoolStatus(&msg); err != nil {
		ctx.Logger().Error("fail to check pool status", "error", err)
		return errInvalidPoolStatus
	}

	if h.mgr.Keeper().IsChainHalted(ctx, msg.Asset.Chain) || h.mgr.Keeper().IsLPPaused(ctx, msg.Asset.Chain) {
		return fmt.Errorf("unable to add liquidity while chain has paused LP actions")
	}
	if h.mgr.Keeper().IsPoolDepositPaused(ctx, msg.Asset) {
		return fmt.Errorf("unable to add liquidity, deposits are paused for asset (%s)", msg.Asset.String())
	}

	ensureLiquidityNoLargerThanBond := h.mgr.GetConstants().GetBoolValue(constants.StrictBondLiquidityRatio)
	// if the pool is SwitchlyProtocol no need to check economic security
	if msg.Asset.IsSyntheticAsset() || !ensureLiquidityNoLargerThanBond {
		return nil
	}

	// the following is only applicable for mainnet
	totalLiquidityRUNE, err := h.getTotalLiquidityRUNE(ctx)
	if err != nil {
		return ErrInternal(err, "fail to get total liquidity SWITCH")
	}

	// total liquidity SWITCH after current add liquidity
	totalLiquidityRUNE = totalLiquidityRUNE.Add(msg.RuneAmount)
	totalLiquidityRUNE = totalLiquidityRUNE.Add(pool.AssetValueInRune(msg.AssetAmount))
	maximumLiquidityRune := h.mgr.Keeper().GetConfigInt64(ctx, constants.MaximumLiquidityRune)
	if maximumLiquidityRune > 0 {
		if totalLiquidityRUNE.GT(cosmos.NewUint(uint64(maximumLiquidityRune))) {
			return errAddLiquiditySWITCHOverLimit
		}
	}

	coins := common.NewCoins(
		common.NewCoin(common.SwitchNative, msg.RuneAmount),
		common.NewCoin(msg.Asset, msg.AssetAmount),
	)
	if atTVLCap(ctx, coins, h.mgr) {
		return errAddLiquiditySWITCHMoreThanBond
	}

	return nil
}

func (h AddLiquidityHandler) handle(ctx cosmos.Context, msg MsgAddLiquidity) error {
	// check if we need to swap before adding asset
	if h.needsSwap(msg) {
		return h.swap(ctx, msg)
	}

	pool, err := h.mgr.Keeper().GetPool(ctx, msg.Asset)
	if err != nil {
		return ErrInternal(err, "fail to get pool")
	}

	if pool.IsEmpty() {
		ctx.Logger().Info("pool doesn't exist yet, creating a new one...", "symbol", msg.Asset.String(), "creator", msg.RuneAddress)

		pool.Asset = msg.Asset

		defaultPoolStatus := PoolAvailable.String()
		// only set the pool to default pool status if not for gas asset on the chain
		if !pool.Asset.Equals(pool.Asset.GetChain().GetGasAsset()) &&
			!pool.Asset.IsSyntheticAsset() {
			defaultPoolStatus = h.mgr.GetConstants().GetStringValue(constants.DefaultPoolStatus)
		}
		pool.Status = GetPoolStatus(defaultPoolStatus)

		if err = h.mgr.Keeper().SetPool(ctx, pool); err != nil {
			return ErrInternal(err, "fail to save pool to key value store")
		}
	}

	// if the pool decimals hasn't been set, it will still be 0. If we have a
	// pool asset coin, get the decimals from that transaction. This will only
	// set the decimals once.
	if pool.Decimals == 0 {
		coin := msg.GetTx().Coins.GetCoin(pool.Asset)
		if !coin.IsEmpty() && coin.Decimals > 0 {
			pool.Decimals = coin.Decimals
			if err = h.mgr.Keeper().SetPool(ctx, pool); err != nil {
				return ErrInternal(err, "fail to save pool to key value store")
			}
			ctx.Logger().Info("update pool decimals", "asset", msg.Asset, "pool decimals", pool.Decimals)
		}
	}

	// figure out if we need to stage the funds and wait for a follow on
	// transaction to commit all funds atomically. For pools of native assets
	// only, stage is always false
	stage := false
	if !msg.Asset.IsSyntheticAsset() && !msg.Tx.ID.IsBlank() {
		if !msg.AssetAddress.IsEmpty() && msg.AssetAmount.IsZero() {
			stage = true
		}
		if !msg.RuneAddress.IsEmpty() && msg.RuneAmount.IsZero() {
			stage = true
		}
	}

	if msg.AffiliateBasisPoints.IsZero() {
		return h.addLiquidity(
			ctx,
			msg.Asset,
			msg.RuneAmount,
			msg.AssetAmount,
			msg.RuneAddress,
			msg.AssetAddress,
			msg.Tx.ID,
			stage,
			h.mgr.GetConstants())
	}

	// add liquidity has an affiliate fee, add liquidity for both the user and their affiliate
	affiliateRune := common.GetSafeShare(msg.AffiliateBasisPoints, cosmos.NewUint(10000), msg.RuneAmount)
	affiliateAsset := common.GetSafeShare(msg.AffiliateBasisPoints, cosmos.NewUint(10000), msg.AssetAmount)
	userRune := common.SafeSub(msg.RuneAmount, affiliateRune)
	userAsset := common.SafeSub(msg.AssetAmount, affiliateAsset)

	err = h.addLiquidity(
		ctx,
		msg.Asset,
		userRune,
		userAsset,
		msg.RuneAddress,
		msg.AssetAddress,
		msg.Tx.ID,
		stage,
		h.mgr.GetConstants(),
	)
	if err != nil {
		return err
	}

	affiliateRuneAddress := common.NoAddress
	affiliateAssetAddress := common.NoAddress
	if msg.AffiliateAddress.IsChain(common.SWITCHLYChain) {
		affiliateRuneAddress = msg.AffiliateAddress
	} else {
		affiliateAssetAddress = msg.AffiliateAddress
	}

	err = h.addLiquidity(
		ctx,
		msg.Asset,
		affiliateRune,
		affiliateAsset,
		affiliateRuneAddress,
		affiliateAssetAddress,
		msg.Tx.ID,
		false,
		h.mgr.GetConstants(),
	)
	if err != nil {
		ctx.Logger().Error("fail to add liquidity for affiliate", "address", msg.AffiliateAddress, "error", err)
		return err
	}
	return nil
}

func (h AddLiquidityHandler) swap(ctx cosmos.Context, msg MsgAddLiquidity) error {
	// ensure TxID does NOT have a collision with another swap, this could
	// happen if the user submits two identical loan requests in the same
	// block
	if ok := h.mgr.Keeper().HasSwapQueueItem(ctx, msg.Tx.ID, 0); ok {
		return fmt.Errorf("txn hash conflict")
	}

	// sanity check, ensure address or asset doesn't have separator within them
	if strings.Contains(fmt.Sprintf("%s%s", msg.Asset, msg.AffiliateAddress), ":") {
		return fmt.Errorf("illegal character")
	}
	memo := fmt.Sprintf("+:%s::%s:%d", msg.Asset, msg.AffiliateAddress, msg.AffiliateBasisPoints.Uint64())
	msg.Tx.Memo = memo

	// Get streaming swaps interval to use for native -> synth swap
	ssInterval := h.mgr.Keeper().GetConfigInt64(ctx, constants.SaversStreamingSwapsInterval)
	if ssInterval <= 0 {
		ssInterval = 0
	}

	swapMsg := NewMsgSwap(msg.Tx, msg.Asset, common.NoopAddress, cosmos.ZeroUint(), common.NoAddress, cosmos.ZeroUint(), "", "", nil, MarketSwap, 0, uint64(ssInterval), msg.Signer)

	// sanity check swap msg
	handler := NewSwapHandler(h.mgr)
	if err := handler.validate(ctx, *swapMsg); err != nil {
		return err
	}
	if err := h.mgr.Keeper().SetSwapQueueItem(ctx, *swapMsg, 0); err != nil {
		ctx.Logger().Error("fail to add swap to queue", "error", err)
		return err
	}

	return nil
}

// validateAddLiquidityMessage is to do some validation, and make sure it is legit
func (h AddLiquidityHandler) validateAddLiquidityMessage(ctx cosmos.Context, keeper keeper.Keeper, asset common.Asset, requestTxHash common.TxID, runeAddr, assetAddr common.Address) error {
	if asset.IsEmpty() {
		return errors.New("asset is empty")
	}
	if requestTxHash.IsEmpty() {
		return errors.New("request tx hash is empty")
	}
	if runeAddr.IsEmpty() && assetAddr.IsEmpty() {
		return errors.New("rune address and asset address is empty")
	}
	if !keeper.PoolExist(ctx, asset) {
		return fmt.Errorf("%s doesn't exist", asset)
	}
	pool, err := h.mgr.Keeper().GetPool(ctx, asset)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get pool(%s)", asset))
	}
	if pool.Status == PoolStaged && (runeAddr.IsEmpty() || assetAddr.IsEmpty()) {
		return fmt.Errorf("cannot add single sided liquidity while a pool is staged")
	}
	return nil
}

// r = rune provided;
// a = asset provided
// R = rune Balance (before)
// A = asset Balance (before)
// P = Pool Units (before)
// units / (P + units) = (1/2) * ((r / (R + r)) + (a / (A + a)))
// units = P * (r*A + a*R + 2*r*a) / (r*A + a*R + 2*R*A)
func calculatePoolUnits(oldPoolUnits, poolRune, poolAsset, addRune, addAsset cosmos.Uint) (cosmos.Uint, cosmos.Uint, error) {
	if addRune.Add(poolRune).IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("total RUNE in the pool is zero")
	}
	if addAsset.Add(poolAsset).IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("total asset in the pool is zero")
	}

	if poolRune.IsZero() || poolAsset.IsZero() || oldPoolUnits.IsZero() {
		sUnits := addRune
		if addRune.IsZero() {
			if !poolAsset.IsZero() {
				// Where possible, keep a RUNE scale for new units.
				sUnits = addAsset.Mul(poolRune).Quo(poolAsset)
			} else {
				sUnits = addAsset
			}
		}
		return oldPoolUnits.Add(sUnits), sUnits, nil
	}

	P := cosmos.NewDecFromBigInt(oldPoolUnits.BigInt())
	R := cosmos.NewDecFromBigInt(poolRune.BigInt())
	A := cosmos.NewDecFromBigInt(poolAsset.BigInt())
	r := cosmos.NewDecFromBigInt(addRune.BigInt())
	a := cosmos.NewDecFromBigInt(addAsset.BigInt())

	// r*A + a*R
	cross := (r.Mul(A)).Add(a.Mul(R))

	// P * (r*A + a*R + 2*r*a)
	numerator := P.Mul(cross.Add(cosmos.NewDec(2).Mul(r).Mul(a)))
	// r*A + a*R + 2*R*A
	denominator := cross.Add(cosmos.NewDec(2).Mul(A).Mul(R))
	if denominator.IsZero() {
		return cosmos.ZeroUint(), cosmos.ZeroUint(), errors.New("denominator of new pool units calculation is zero")
	}

	liquidityUnits := numerator.Quo(denominator)
	newPoolUnit := P.Add(liquidityUnits)

	pUnits := cosmos.NewUintFromBigInt(newPoolUnit.TruncateInt().BigInt())
	sUnits := cosmos.NewUintFromBigInt(liquidityUnits.TruncateInt().BigInt())

	return pUnits, sUnits, nil
}

func calculateVaultUnits(oldPoolUnits, poolAmt, addAmt cosmos.Uint) (cosmos.Uint, cosmos.Uint) {
	if oldPoolUnits.IsZero() || poolAmt.IsZero() {
		return addAmt, addAmt
	}
	if addAmt.IsZero() {
		return oldPoolUnits, cosmos.ZeroUint()
	}
	lpUnits := common.GetUncappedShare(addAmt, poolAmt, oldPoolUnits)
	return oldPoolUnits.Add(lpUnits), lpUnits
}

func (h AddLiquidityHandler) addLiquidity(ctx cosmos.Context,
	asset common.Asset,
	addRuneAmount, addAssetAmount cosmos.Uint,
	runeAddr, assetAddr common.Address,
	requestTxHash common.TxID,
	stage bool,
	constAccessor constants.ConstantValues,
) (err error) {
	ctx.Logger().Info("liquidity provision", "asset", asset, "rune amount", addRuneAmount, "asset amount", addAssetAmount)
	if err = h.validateAddLiquidityMessage(ctx, h.mgr.Keeper(), asset, requestTxHash, runeAddr, assetAddr); err != nil {
		return fmt.Errorf("add liquidity message fail validation: %w", err)
	}

	pool, err := h.mgr.Keeper().GetPool(ctx, asset)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get pool(%s)", asset))
	}
	synthSupply := h.mgr.Keeper().GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
	originalUnits := pool.CalcUnits(synthSupply)

	fetchAddr := runeAddr
	if fetchAddr.IsEmpty() {
		fetchAddr = assetAddr
	}
	su, err := h.mgr.Keeper().GetLiquidityProvider(ctx, asset, fetchAddr)
	if err != nil {
		return ErrInternal(err, "fail to get liquidity provider")
	}

	su.LastAddHeight = ctx.BlockHeight()
	if su.Units.IsZero() {
		if su.PendingTxID.IsEmpty() {
			if su.RuneAddress.IsEmpty() {
				su.RuneAddress = runeAddr
			}
			if su.AssetAddress.IsEmpty() {
				su.AssetAddress = assetAddr
			}
		}

		if asset.IsSyntheticAsset() {
			// new SU, by default, places the thor address to the rune address,
			// but here we want it to be on the asset address only
			su.AssetAddress = assetAddr
			su.RuneAddress = common.NoAddress // no rune to add/withdraw
		} else {
			// ensure input addresses match LP position addresses
			if !runeAddr.Equals(su.RuneAddress) {
				return errAddLiquidityMismatchAddr
			}
			if !assetAddr.Equals(su.AssetAddress) {
				return errAddLiquidityMismatchAddr
			}
		}
	}

	if asset.IsSyntheticAsset() {
		if su.AssetAddress.IsEmpty() || !su.AssetAddress.IsChain(asset.GetLayer1Asset().GetChain()) {
			return errAddLiquidityMismatchAddr
		}
	} else if !assetAddr.IsEmpty() && !su.AssetAddress.Equals(assetAddr) {
		// mismatch of asset addresses from what is known to the address
		// given. Refund it.
		return errAddLiquidityMismatchAddr
	}

	// get tx hashes
	runeTxID := requestTxHash
	assetTxID := requestTxHash
	if addRuneAmount.IsZero() {
		runeTxID = su.PendingTxID
	} else {
		assetTxID = su.PendingTxID
	}

	pendingRuneAmt := su.PendingRune.Add(addRuneAmount)
	pendingAssetAmt := su.PendingAsset.Add(addAssetAmount)

	// if we have an asset address and no asset amount, put the rune pending
	if stage && pendingAssetAmt.IsZero() {
		pool.PendingInboundRune = pool.PendingInboundRune.Add(addRuneAmount)
		su.PendingRune = pendingRuneAmt
		su.PendingTxID = requestTxHash
		h.mgr.Keeper().SetLiquidityProvider(ctx, su)
		if err = h.mgr.Keeper().SetPool(ctx, pool); err != nil {
			ctx.Logger().Error("fail to save pool pending inbound rune", "error", err)
		}

		// add pending liquidity event
		evt := NewEventPendingLiquidity(pool.Asset, AddPendingLiquidity, su.RuneAddress, addRuneAmount, su.AssetAddress, cosmos.ZeroUint(), requestTxHash, common.TxID(""))
		if err = h.mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
			return ErrInternal(err, "fail to emit partial add liquidity event")
		}
		return nil
	}

	// if we have a rune address and no rune asset, put the asset in pending
	if stage && pendingRuneAmt.IsZero() {
		pool.PendingInboundAsset = pool.PendingInboundAsset.Add(addAssetAmount)
		su.PendingAsset = pendingAssetAmt
		su.PendingTxID = requestTxHash
		h.mgr.Keeper().SetLiquidityProvider(ctx, su)
		if err = h.mgr.Keeper().SetPool(ctx, pool); err != nil {
			ctx.Logger().Error("fail to save pool pending inbound asset", "error", err)
		}
		evt := NewEventPendingLiquidity(pool.Asset, AddPendingLiquidity, su.RuneAddress, cosmos.ZeroUint(), su.AssetAddress, addAssetAmount, common.TxID(""), requestTxHash)
		if err = h.mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
			return ErrInternal(err, "fail to emit partial add liquidity event")
		}
		return nil
	}

	pool.PendingInboundRune = common.SafeSub(pool.PendingInboundRune, su.PendingRune)
	pool.PendingInboundAsset = common.SafeSub(pool.PendingInboundAsset, su.PendingAsset)
	su.PendingAsset = cosmos.ZeroUint()
	su.PendingRune = cosmos.ZeroUint()
	su.PendingTxID = ""

	ctx.Logger().Info("pre add liquidity", "pool", pool.Asset, "rune", pool.BalanceRune, "asset", pool.BalanceAsset, "LP units", pool.LPUnits, "synth units", pool.SynthUnits)
	ctx.Logger().Info("adding liquidity", "rune", addRuneAmount, "asset", addAssetAmount)

	balanceRune := pool.BalanceRune
	balanceAsset := pool.BalanceAsset

	oldPoolUnits := pool.GetPoolUnits()
	var newPoolUnits, liquidityUnits cosmos.Uint
	if asset.IsSyntheticAsset() {
		pendingRuneAmt = cosmos.ZeroUint() // sanity check
		newPoolUnits, liquidityUnits = calculateVaultUnits(oldPoolUnits, balanceAsset, pendingAssetAmt)
	} else {
		newPoolUnits, liquidityUnits, err = calculatePoolUnits(oldPoolUnits, balanceRune, balanceAsset, pendingRuneAmt, pendingAssetAmt)
		if err != nil {
			return ErrInternal(err, "fail to calculate pool unit")
		}
	}

	ctx.Logger().Info("current pool status", "pool units", newPoolUnits, "liquidity units", liquidityUnits)
	poolRune := balanceRune.Add(pendingRuneAmt)
	poolAsset := balanceAsset.Add(pendingAssetAmt)
	pool.LPUnits = pool.LPUnits.Add(liquidityUnits)
	pool.BalanceRune = poolRune
	pool.BalanceAsset = poolAsset
	ctx.Logger().Info("post add liquidity", "pool", pool.Asset, "rune", pool.BalanceRune, "asset", pool.BalanceAsset, "LP units", pool.LPUnits, "synth units", pool.SynthUnits, "add liquidity units", liquidityUnits)
	if (pool.BalanceRune.IsZero() && !asset.IsSyntheticAsset()) || pool.BalanceAsset.IsZero() {
		return ErrInternal(err, "pool cannot have zero rune or asset balance")
	}

	// CalcUnits to set the correct SynthUnits before the DepositValue calculations
	pool.CalcUnits(synthSupply)

	if err = h.mgr.Keeper().SetPool(ctx, pool); err != nil {
		return ErrInternal(err, "fail to save pool")
	}
	if originalUnits.IsZero() && !pool.GetPoolUnits().IsZero() {
		poolEvent := NewEventPool(pool.Asset, pool.Status)
		if err = h.mgr.EventMgr().EmitEvent(ctx, poolEvent); err != nil {
			ctx.Logger().Error("fail to emit pool event", "error", err)
		}
	}

	su.Units = su.Units.Add(liquidityUnits)
	if pool.Status == PoolAvailable {
		if su.AssetDepositValue.IsZero() && su.RuneDepositValue.IsZero() {
			su.RuneDepositValue = common.GetSafeShare(su.Units, pool.GetPoolUnits(), pool.BalanceRune)
			su.AssetDepositValue = common.GetSafeShare(su.Units, pool.GetPoolUnits(), pool.BalanceAsset)
		} else {
			su.RuneDepositValue = su.RuneDepositValue.Add(common.GetSafeShare(liquidityUnits, pool.GetPoolUnits(), pool.BalanceRune))
			su.AssetDepositValue = su.AssetDepositValue.Add(common.GetSafeShare(liquidityUnits, pool.GetPoolUnits(), pool.BalanceAsset))
		}
	}
	h.mgr.Keeper().SetLiquidityProvider(ctx, su)

	evt := NewEventAddLiquidity(asset, liquidityUnits, su.RuneAddress, pendingRuneAmt, pendingAssetAmt, runeTxID, assetTxID, su.AssetAddress)
	if err = h.mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
		return ErrInternal(err, "fail to emit add liquidity event")
	}

	// if its the POL is adding, track rune added
	polAddress, err := h.mgr.Keeper().GetModuleAddress(ReserveName)
	if err != nil {
		return err
	}

	if polAddress.Equals(su.RuneAddress) {
		var pol ProtocolOwnedLiquidity
		pol, err = h.mgr.Keeper().GetPOL(ctx)
		if err != nil {
			return err
		}
		pol.RuneDeposited = pol.RuneDeposited.Add(pendingRuneAmt)

		if err = h.mgr.Keeper().SetPOL(ctx, pol); err != nil {
			return err
		}

		ctx.Logger().Info("POL deposit", "pool", pool.Asset, "rune", pendingRuneAmt)
		telemetry.IncrCounterWithLabels(
			[]string{"thornode", "pol", "pool", "rune_deposited"},
			telem(pendingRuneAmt),
			[]metrics.Label{telemetry.NewLabel("pool", pool.Asset.String())},
		)
	}
	return nil
}

// getTotalLiquiditySWITCH we have in all pools (legacy function name for backward compatibility)
func (h AddLiquidityHandler) getTotalLiquidityRUNE(ctx cosmos.Context) (cosmos.Uint, error) {
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
		total = total.Add(p.BalanceRune)
	}
	return total, nil
}

func (h AddLiquidityHandler) needsSwap(msg MsgAddLiquidity) bool {
	return len(msg.Tx.Coins) == 1 && !msg.Tx.Coins[0].IsSwitch() && !msg.Asset.Equals(msg.Tx.Coins[0].Asset)
}
