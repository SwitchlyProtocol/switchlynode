package switchly

import (
	"fmt"
	"sort"
	"strings"

	"github.com/blang/semver"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
	"github.com/hashicorp/go-multierror"

	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/keeper"
	"github.com/switchlyprotocol/switchlynode/v3/x/switchly/types"
)

func refundTx(ctx cosmos.Context, tx ObservedTx, mgr Manager, refundCode uint32, refundReason, sourceModuleName string) error {
	// If SWITCHLYNode recognize one of the coins, and therefore able to refund
	// withholding fees, refund all coins.

	refundCoins := make(common.Coins, 0)
	for _, coin := range tx.Tx.Coins {
		// Do not emit Trade/Secured Asset refund event or attempt refund txout,
		// as Trade/Secured Asset withdrawals take place in the internal handlers
		// (state changes and event emission only if the internal handler succeeds)
		// and not in the deposit handler.
		if coin.Asset.IsTradeAsset() || coin.Asset.IsSecuredAsset() {
			continue
		}

		if coin.IsSwitch() && coin.Asset.GetChain().Equals(common.ETHChain) {
			continue
		}
		pool, err := mgr.Keeper().GetPool(ctx, coin.Asset.GetLayer1Asset())
		if err != nil {
			return fmt.Errorf("fail to get pool: %w", err)
		}

		// Only attempt an outbound if a fee can be taken from the coin.
		if coin.IsSwitch() || !pool.BalanceSwitch.IsZero() {
			toAddr := tx.Tx.FromAddress
			memo, err := ParseMemoWithSWITCHNames(ctx, mgr.Keeper(), tx.Tx.Memo)
			if err == nil && memo.IsType(TxSwap) && !memo.GetRefundAddress().IsEmpty() && !coin.Asset.GetChain().IsSWITCHLYChain() {
				// If the memo specifies a refund address, send the refund to that address. If
				// refund memo can't be parsed or is invalid for the refund chain, it will
				// default back to the sender address
				if memo.GetRefundAddress().IsChain(coin.Asset.GetChain()) {
					toAddr = memo.GetRefundAddress()
				}
			}

			toi := TxOutItem{
				Chain:       coin.Asset.GetChain(),
				InHash:      tx.Tx.ID,
				ToAddress:   toAddr,
				VaultPubKey: tx.ObservedPubKey,
				Coin:        coin,
				Memo:        NewRefundMemo(tx.Tx.ID).String(),
				ModuleName:  sourceModuleName,
			}

			success, err := mgr.TxOutStore().TryAddTxOutItem(ctx, mgr, toi, cosmos.ZeroUint())
			if err != nil {
				ctx.Logger().Error("fail to prepare outbound tx", "error", err)
				// concatenate the refund failure to refundReason
				refundReason = fmt.Sprintf("%s; fail to refund (%s): %s", refundReason, toi.Coin.String(), err)

				unrefundableCoinCleanup(ctx, mgr, toi, "failed_refund")
			}
			if success {
				refundCoins = append(refundCoins, toi.Coin)
			}
		}
		// Zombie coins are just dropped.
	}

	// For refund events, emit the event after the txout attempt in order to include the 'fail to refund' reason if unsuccessful.
	eventRefund := NewEventRefund(refundCode, refundReason, tx.Tx, common.NewFee(common.Coins{}, cosmos.ZeroUint()))
	if len(refundCoins) > 0 {
		// create a new TX based on the coins switchlyprotocol refund , some of the coins switchlyprotocol doesn't refund
		// coin switchlyprotocol doesn't have pool with , likely airdrop
		newTx := common.NewTx(tx.Tx.ID, tx.Tx.FromAddress, tx.Tx.ToAddress, tx.Tx.Coins, tx.Tx.Gas, tx.Tx.Memo)
		eventRefund = NewEventRefund(refundCode, refundReason, newTx, common.Fee{}) // fee param not used in downstream event
	}
	if err := mgr.EventMgr().EmitEvent(ctx, eventRefund); err != nil {
		return fmt.Errorf("fail to emit refund event: %w", err)
	}

	return nil
}

// unrefundableCoinCleanup - update the accounting for a failed outbound of toi.Coin
// native switch: send to the reserve
// native coin besides switch: burn
// non-native coin: donate to its pool
func unrefundableCoinCleanup(ctx cosmos.Context, mgr Manager, toi TxOutItem, burnReason string) {
	coin := toi.Coin

	if coin.Asset.IsTradeAsset() {
		return
	}

	sourceModuleName := toi.GetModuleName() // Ensure that non-"".

	// For context in emitted events, retrieve the original transaction that prompted the cleanup.
	// If there is no retrievable transaction, leave those fields empty.
	voter, err := mgr.Keeper().GetObservedTxInVoter(ctx, toi.InHash)
	if err != nil {
		ctx.Logger().Error("fail to get observed tx in", "error", err, "hash", toi.InHash.String())
		return
	}
	tx := voter.Tx.Tx
	// For emitted events' amounts (such as EventDonate), replace the Coins with the coin being cleaned up.
	tx.Coins = common.NewCoins(toi.Coin)

	// Select course of action according to coin type:
	// External coin, native coin which isn't SWITCH, or native SWITCH (not from the Reserve).
	switch {
	case !coin.Asset.IsNative():
		// If unable to refund external-chain coins, add them to their pools
		// (so they aren't left in the vaults with no reflection in the pools).
		// Failed-refund external coins have earlier been established to have existing pools with non-zero BalanceSwitch.

		ctx.Logger().Error("fail to refund non-native tx, leaving coins in vault", "toi.InHash", toi.InHash, "toi.Coin", toi.Coin)
		return
	case sourceModuleName != ReserveName:
		// If unable to refund SWITCHLY.SWITCH, send it to the Reserve.
		err := mgr.Keeper().SendFromModuleToModule(ctx, sourceModuleName, ReserveName, common.NewCoins(coin))
		if err != nil {
			ctx.Logger().Error("fail to send native coin to Reserve during cleanup", "error", err)
			return
		}

		reserveContributor := NewReserveContributor(tx.FromAddress, coin.Amount)
		reserveEvent := NewEventReserve(reserveContributor, tx)
		if err := mgr.EventMgr().EmitEvent(ctx, reserveEvent); err != nil {
			ctx.Logger().Error("fail to emit reserve event", "error", err)
		}
	default:
		// If not satisfying the other conditions this coin should be a native coin in the Reserve,
		// so leave it there.
	}
}

func getMaxSwapQuantity(ctx cosmos.Context, mgr Manager, sourceAsset, targetAsset common.Asset, swp StreamingSwap) (uint64, error) {
	version := mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return getMaxSwapQuantityV3_0_0(ctx, mgr, sourceAsset, targetAsset, swp)
	default:
		return 0, errBadVersion
	}
}

func getMaxSwapQuantityV3_0_0(ctx cosmos.Context, mgr Manager, sourceAsset, targetAsset common.Asset, swp StreamingSwap) (uint64, error) {
	if swp.Interval == 0 {
		return 0, nil
	}

	// collect pools involved in this swap
	minSwapSize := cosmos.ZeroUint()
	var sourceAssetPool types.Pool
	for i, asset := range []common.Asset{sourceAsset, targetAsset} {
		if asset.IsSwitch() {
			continue
		}

		// get the asset pool
		pool, err := mgr.Keeper().GetPool(ctx, asset.GetLayer1Asset())
		if err != nil {
			ctx.Logger().Error("fail to fetch pool", "error", err)
			return 0, err
		}

		// store the source asset pool for later conversion of SWITCH to asset
		if i == 0 {
			sourceAssetPool = pool
		}

		// get the configured min slip for this asset
		minSlip := getMinSlipBps(ctx, mgr.Keeper(), asset)
		if minSlip.IsZero() {
			continue
		}

		// compute the minimum switch swap size for this leg of the swap
		minSWITCHSwapSize := common.GetSafeShare(minSlip, cosmos.NewUint(constants.MaxBasisPts), pool.BalanceSwitch)
		if minSwapSize.IsZero() || minSWITCHSwapSize.LT(minSwapSize) {
			minSwapSize = minSWITCHSwapSize
		}
	}

	// calculate the max swap quantity
	if !sourceAsset.IsSwitch() {
		minSwapSize = sourceAssetPool.SWITCHValueInAsset(minSwapSize)
	}
	if minSwapSize.IsZero() {
		return 1, nil
	}
	maxSwapQuantity := swp.Deposit.Quo(minSwapSize)

	// make sure maxSwapQuantity doesn't infringe on max length that a
	// streaming swap can exist
	var maxLength int64
	if sourceAsset.IsNative() && targetAsset.IsNative() {
		maxLength = mgr.Keeper().GetConfigInt64(ctx, constants.StreamingSwapMaxLengthNative)
	} else {
		maxLength = mgr.Keeper().GetConfigInt64(ctx, constants.StreamingSwapMaxLength)
	}
	if swp.Interval == 0 {
		return 1, nil
	}
	maxSwapInMaxLength := uint64(maxLength) / swp.Interval
	if maxSwapQuantity.GT(cosmos.NewUint(maxSwapInMaxLength)) {
		return maxSwapInMaxLength, nil
	}

	// sanity check that max swap quantity is not zero
	if maxSwapQuantity.IsZero() {
		return 1, nil
	}

	// if swapping with a derived asset, reduce quantity relative to derived
	// virtual pool depth. The equation for this as follows
	dbps := cosmos.ZeroUint()
	for _, asset := range []common.Asset{sourceAsset, targetAsset} {
		if !asset.IsDerivedAsset() {
			continue
		}

		// get the switch depth of the anchor pool(s)
		switchDepth, _, _ := mgr.NetworkMgr().CalcAnchor(ctx, mgr, asset)
		dpool, _ := mgr.Keeper().GetPool(ctx, asset) // get the derived asset pool
		newDbps := common.GetUncappedShare(dpool.BalanceSwitch, switchDepth, cosmos.NewUint(constants.MaxBasisPts))
		if dbps.IsZero() || newDbps.LT(dbps) {
			dbps = newDbps
		}
	}
	if !dbps.IsZero() {
		// quantity = 1 / (1-dbps)
		// But since we're dealing in basis points (to avoid float math)
		// quantity = 10,000 / (10,000 - dbps)
		maxBasisPoints := cosmos.NewUint(constants.MaxBasisPts)
		diff := common.SafeSub(maxBasisPoints, dbps)
		if !diff.IsZero() {
			newQuantity := maxBasisPoints.Quo(diff)
			if maxSwapQuantity.GT(newQuantity) {
				return newQuantity.Uint64(), nil
			}
		}
	}

	return maxSwapQuantity.Uint64(), nil
}

// getMinSlipBps returns artificial slip floor, expressed in basis points (10000).
func getMinSlipBps(
	ctx cosmos.Context,
	k keeper.Keeper,
	asset common.Asset,
) cosmos.Uint {
	var ref constants.ConstantName
	switch {
	case asset.IsSyntheticAsset():
		ref = constants.SynthSlipMinBps
	case asset.IsTradeAsset():
		ref = constants.TradeAccountsSlipMinBps
	case asset.IsDerivedAsset():
		ref = constants.DerivedSlipMinBps
	case asset.IsSecuredAsset():
		ref = constants.SecuredAssetSlipMinBps
	default:
		ref = constants.L1SlipMinBps
	}
	minFeeMimir := k.GetConfigInt64(ctx, ref)
	return cosmos.SafeUintFromInt64(minFeeMimir)
}

func refundBond(ctx cosmos.Context, tx common.Tx, acc cosmos.AccAddress, amt cosmos.Uint, nodeAcc *NodeAccount, mgr Manager) error {
	if nodeAcc.Status == NodeActive {
		ctx.Logger().Info("node still active, cannot refund bond", "node address", nodeAcc.NodeAddress, "node pub key", nodeAcc.PubKeySet.Secp256k1)
		return nil
	}

	// ensures nodes don't return bond while being churned into the network
	// (removing their bond last second)
	if nodeAcc.Status == NodeReady {
		ctx.Logger().Info("node ready, cannot refund bond", "node address", nodeAcc.NodeAddress, "node pub key", nodeAcc.PubKeySet.Secp256k1)
		return nil
	}

	if amt.IsZero() || amt.GT(nodeAcc.Bond) {
		amt = nodeAcc.Bond
	}

	bp, err := mgr.Keeper().GetBondProviders(ctx, nodeAcc.NodeAddress)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get bond providers(%s)", nodeAcc.NodeAddress))
	}

	err = passiveBackfill(ctx, mgr, *nodeAcc, &bp)
	if err != nil {
		return err
	}

	bp.Adjust(nodeAcc.Bond) // redistribute node bond amongst bond providers
	provider := bp.Get(acc)

	if !provider.IsEmpty() && !provider.Bond.IsZero() {
		if amt.GT(provider.Bond) {
			amt = provider.Bond
		}

		bp.Unbond(amt, provider.BondAddress)

		// refund bond;
		// this is always SWITCH, and the MsgDeposit handler should have already deducted a network fee,
		// so this can be done with SendFromModuleToAccount even if under 0.02 SWITCH
		// (which would cause TryAddTxOutItem to fail from no output after fee deduction)
		unbondCoin := common.NewCoin(common.SwitchNative, amt)
		err = mgr.Keeper().SendFromModuleToAccount(ctx, BondName, provider.BondAddress, common.NewCoins(unbondCoin))
		if err != nil {
			return ErrInternal(err, "fail to send unbonded SWITCH to bond address")
		}

		bondEvent := NewEventBond(amt, BondReturned, tx, nodeAcc, provider.BondAddress)
		if err := mgr.EventMgr().EmitEvent(ctx, bondEvent); err != nil {
			ctx.Logger().Error("fail to emit bond event", "error", err)
		}

		nodeAcc.Bond = common.SafeSub(nodeAcc.Bond, amt)
	}

	if nodeAcc.RequestedToLeave {
		// when node already request to leave , it can't come back , here means the node already unbond
		// so set the node to disabled status
		nodeAcc.UpdateStatus(NodeDisabled, ctx.BlockHeight())
	}
	if err := mgr.Keeper().SetNodeAccount(ctx, *nodeAcc); err != nil {
		ctx.Logger().Error(fmt.Sprintf("fail to save node account(%s)", nodeAcc), "error", err)
		return err
	}
	if err := mgr.Keeper().SetBondProviders(ctx, bp); err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to save bond providers(%s)", bp.NodeAddress.String()))
	}

	return nil
}

// isSignedByActiveNodeAccounts check if all signers are active validator nodes
func isSignedByActiveNodeAccounts(ctx cosmos.Context, k keeper.Keeper, signers []cosmos.AccAddress) bool {
	if len(signers) == 0 {
		return false
	}
	for _, signer := range signers {
		if signer.Equals(k.GetModuleAccAddress(AsgardName)) {
			continue
		}
		if err := signedByActiveNodeAccount(ctx, k, signer); err != nil {
			ctx.Logger().Error("unauthorized account", "error", err)
			return false
		}
	}
	return true
}

func activeNodeAccountsSignerPriority(ctx cosmos.Context, k keeper.Keeper, signers []cosmos.AccAddress) (cosmos.Context, error) {
	if isSignedByActiveNodeAccounts(ctx, k, signers) {
		return ctx.WithPriority(ActiveNodePriority), nil
	}
	return ctx, cosmos.ErrUnauthorized(fmt.Sprintf("%+v are not authorized", signers))
}

// signedByActiveNodeAccounts returns an error unless all signers are active validator nodes
func signedByActiveNodeAccount(ctx cosmos.Context, k keeper.Keeper, signer cosmos.AccAddress) error {
	nodeAccount, err := k.GetNodeAccount(ctx, signer)
	if err != nil {
		return fmt.Errorf("error fetching node account: %s: %w", signer.String(), err)
	}
	if nodeAccount.IsEmpty() {
		return fmt.Errorf("node account is unexpectedly empty: %s", signer.String())
	}
	if nodeAccount.Status != NodeActive {
		return fmt.Errorf(
			"node account %s not active: %s",
			signer.String(),
			nodeAccount.Status,
		)
	}
	if nodeAccount.Type != NodeTypeValidator {
		return fmt.Errorf(
			"node account %s must be a validator: %s",
			signer.String(),
			nodeAccount.Type,
		)
	}

	return nil
}

func wrapError(ctx cosmos.Context, err error, wrap string) error {
	err = fmt.Errorf("%s: %w", wrap, err)
	ctx.Logger().Error(err.Error())
	return multierror.Append(errInternal, err)
}

// addGasFees to gas manager and deduct from vault
func addGasFees(ctx cosmos.Context, mgr Manager, tx ObservedTx) error {
	// If there's no gas, then nothing to do.
	if tx.Tx.Gas.IsEmpty() {
		return nil
	}

	// If the transaction wasn't from a known vault, then no relevance for known vaults or pools.
	if !mgr.Keeper().VaultExists(ctx, tx.ObservedPubKey) {
		return nil
	}

	// Since a known vault has spent gas, definitely deduct that gas from the vault's balance
	vault, err := mgr.Keeper().GetVault(ctx, tx.ObservedPubKey)
	if err != nil {
		return err
	}
	vault.SubFunds(tx.Tx.Gas.ToCoins())
	if err := mgr.Keeper().SetVault(ctx, vault); err != nil {
		return err
	}

	// If the vault is an InactiveVault doing an automatic refund,
	// any balance is not represented in the pools,
	// so the Reserve should not reimburse the gas pool.
	if vault.Status == InactiveVault {
		return nil
	}

	// when ragnarok is in progress, if the tx is for gas coin then don't reimburse the pool with reserve
	// liquidity providers they need to pay their own gas
	// if the outbound coin is not gas asset, then reserve will reimburse it , otherwise the gas asset pool will be in a loss
	if mgr.Keeper().RagnarokInProgress(ctx) {
		gasAsset := tx.Tx.Chain.GetGasAsset()
		if !tx.Tx.Coins.GetCoin(gasAsset).IsEmpty() {
			return nil
		}
	}

	// Add the gas to the gas manager to be reimbursed by the Reserve.
	outAsset := common.EmptyAsset
	if len(tx.Tx.Coins) != 0 {
		// Use the first Coin's Asset to indicate the associated outbound Asset for this Gas.
		outAsset = tx.Tx.Coins[0].Asset
	}
	mgr.GasMgr().AddGasAsset(outAsset, tx.Tx.Gas, true)
	return nil
}

func emitPoolBalanceChangedEvent(ctx cosmos.Context, poolMod PoolMod, reason string, mgr Manager) {
	evt := NewEventPoolBalanceChanged(poolMod, reason)
	if err := mgr.EventMgr().EmitEvent(ctx, evt); err != nil {
		ctx.Logger().Error("fail to emit pool balance changed event", "error", err)
	}
}

func getSynthSupplyRemaining(ctx cosmos.Context, mgr Manager, asset common.Asset) (cosmos.Uint, error) {
	maxSynths, err := mgr.Keeper().GetMimir(ctx, constants.MaxSynthPerPoolDepth.String())
	if maxSynths < 0 || err != nil {
		maxSynths = mgr.GetConstants().GetInt64Value(constants.MaxSynthPerPoolDepth)
	}

	synthSupply := mgr.Keeper().GetTotalSupply(ctx, asset.GetSyntheticAsset())
	pool, err := mgr.Keeper().GetPool(ctx, asset.GetLayer1Asset())
	if err != nil {
		return cosmos.ZeroUint(), ErrInternal(err, "fail to get pool")
	}

	if pool.BalanceAsset.IsZero() {
		return cosmos.ZeroUint(), fmt.Errorf("pool(%s) has zero asset balance", pool.Asset.String())
	}

	maxSynthSupply := cosmos.NewUint(uint64(maxSynths)).Mul(pool.BalanceAsset.MulUint64(2)).QuoUint64(MaxWithdrawBasisPoints)
	if maxSynthSupply.LT(synthSupply) {
		return cosmos.ZeroUint(), fmt.Errorf("synth supply over target (%d/%d)", synthSupply.Uint64(), maxSynthSupply.Uint64())
	}

	return maxSynthSupply.Sub(synthSupply), nil
}

// isSynthMintPaused fails validation if synth supply is already too high, relative to pool depth
func isSynthMintPaused(ctx cosmos.Context, mgr Manager, targetAsset common.Asset, outputAmt cosmos.Uint) error {
	// check if the pool is in ragnarok
	k := "RAGNAROK-" + targetAsset.MimirString()
	v, err := mgr.Keeper().GetMimir(ctx, k)
	if err != nil {
		return err
	}
	if v > 0 {
		return fmt.Errorf("pool is in ragnarok")
	}

	mintHeight := mgr.Keeper().GetConfigInt64(ctx, constants.MintSynths)
	if mintHeight > 0 && ctx.BlockHeight() > mintHeight {
		return fmt.Errorf("minting synthetics has been disabled")
	}

	remaining, err := getSynthSupplyRemaining(ctx, mgr, targetAsset)
	if err != nil {
		return err
	}

	if remaining.LT(outputAmt) {
		return fmt.Errorf("insufficient synth capacity: want=%d have=%d", outputAmt.Uint64(), remaining.Uint64())
	}

	return nil
}

func telem(input cosmos.Uint) float32 {
	if !input.BigInt().IsUint64() {
		return 0
	}
	i := input.Uint64()
	return float32(i) / 100000000
}

func telemInt(input cosmos.Int) float32 {
	if !input.BigInt().IsInt64() {
		return 0
	}
	i := input.Int64()
	return float32(i) / 100000000
}

func emitEndBlockTelemetry(ctx cosmos.Context, mgr Manager) error {
	// capture panics
	defer func() {
		if err := recover(); err != nil {
			ctx.Logger().Error("panic while emitting end block telemetry", "error", err)
		}
	}()

	// emit network data
	network, err := mgr.Keeper().GetNetwork(ctx)
	if err != nil {
		return err
	}

	telemetry.SetGauge(telem(network.BondRewardSwitch), "switchlynode", "network", "bond_reward_rune")
	telemetry.SetGauge(float32(network.TotalBondUnits.Uint64()), "switchlynode", "network", "total_bond_units")

	// emit protocol owned liquidity data
	pol, err := mgr.Keeper().GetPOL(ctx)
	if err != nil {
		return err
	}
	telemetry.SetGauge(telem(pol.SwitchDeposited), "switchlynode", "pol", "rune_deposited")
	telemetry.SetGauge(telem(pol.SwitchWithdrawn), "switchlynode", "pol", "rune_withdrawn")
	telemetry.SetGauge(telemInt(pol.CurrentDeposit()), "switchlynode", "pol", "current_deposit")
	polValue, err := polPoolValue(ctx, mgr)
	if err == nil {
		telemetry.SetGauge(telem(polValue), "switchlynode", "pol", "value")
		telemetry.SetGauge(telemInt(pol.PnL(polValue)), "switchlynode", "pol", "pnl")
	}

	// emit module balances
	for _, name := range []string{ReserveName, AsgardName, BondName} {
		modAddr := mgr.Keeper().GetModuleAccAddress(name)
		bal := mgr.Keeper().GetBalance(ctx, modAddr)
		for _, coin := range bal {
			modLabel := telemetry.NewLabel("module", name)
			denom := telemetry.NewLabel("denom", coin.Denom)
			telemetry.SetGaugeWithLabels(
				[]string{"switchlynode", "module", "balance"},
				telem(cosmos.NewUint(coin.Amount.Uint64())),
				[]metrics.Label{modLabel, denom},
			)
		}
	}

	// emit node metrics
	nodes, err := mgr.Keeper().ListValidatorsWithBond(ctx)
	if err != nil {
		return err
	}
	for _, node := range nodes {
		telemetry.SetGaugeWithLabels(
			[]string{"switchlynode", "node", "bond"},
			telem(cosmos.NewUint(node.Bond.Uint64())),
			[]metrics.Label{telemetry.NewLabel("node_address", node.NodeAddress.String()), telemetry.NewLabel("status", node.Status.String())},
		)
		pts, err := mgr.Keeper().GetNodeAccountSlashPoints(ctx, node.NodeAddress)
		if err != nil {
			continue
		}
		telemetry.SetGaugeWithLabels(
			[]string{"switchlynode", "node", "slash_points"},
			float32(pts),
			[]metrics.Label{telemetry.NewLabel("node_address", node.NodeAddress.String())},
		)

		age := cosmos.NewUint(uint64((ctx.BlockHeight() - node.StatusSince) * common.One))
		if pts > 0 {
			leaveScore := age.QuoUint64(uint64(pts))
			telemetry.SetGaugeWithLabels(
				[]string{"switchlynode", "node", "leave_score"},
				float32(leaveScore.Uint64()),
				[]metrics.Label{telemetry.NewLabel("node_address", node.NodeAddress.String())},
			)
		}
	}

	// get 1 SWITCH price in USD
	switchUSDPrice := telem(mgr.Keeper().DollarsPerSWITCH(ctx))
	telemetry.SetGauge(switchUSDPrice, "switchlynode", "price", "usd", "switchly", "switch")

	// emit pool metrics
	pools, err := mgr.Keeper().GetPools(ctx)
	if err != nil {
		return err
	}
	for _, pool := range pools {
		if pool.LPUnits.IsZero() {
			continue
		}
		synthSupply := mgr.Keeper().GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
		labels := []metrics.Label{telemetry.NewLabel("pool", pool.Asset.String()), telemetry.NewLabel("status", pool.Status.String())}
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "balance", "synth"}, telem(synthSupply), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "balance", "switch"}, telem(pool.BalanceSwitch), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "balance", "asset"}, telem(pool.BalanceAsset), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "pending", "switch"}, telem(pool.PendingInboundSwitch), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "pending", "asset"}, telem(pool.PendingInboundAsset), labels)

		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "units", "pool"}, telem(pool.CalcUnits(synthSupply)), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "units", "lp"}, telem(pool.LPUnits), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "units", "synth"}, telem(pool.SynthUnits), labels)

		// pricing
		price := float32(0)
		if !pool.BalanceAsset.IsZero() {
			price = switchUSDPrice * telem(pool.BalanceSwitch) / telem(pool.BalanceAsset)
		}
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "price", "usd"}, price, labels)

		// trade accounts
		tu, err := mgr.Keeper().GetTradeUnit(ctx, pool.Asset.GetTradeAsset())
		if err != nil {
			ctx.Logger().Error("fail to get trade unit", "error", err)
			continue
		}
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "tradeasset", "units"}, telem(tu.Units), labels)
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "pool", "tradeasset", "depth"}, telem(tu.Depth), labels)
	}

	// emit vault metrics
	asgards, _ := mgr.Keeper().GetAsgardVaults(ctx)
	for _, vault := range asgards {
		if vault.Status != ActiveVault && vault.Status != RetiringVault {
			continue
		}

		// calculate the total value of this vault
		totalValue := cosmos.ZeroUint()
		for _, coin := range vault.Coins {
			if coin.IsSwitch() {
				totalValue = totalValue.Add(coin.Amount)
			} else {
				pool, err := mgr.Keeper().GetPool(ctx, coin.Asset.GetLayer1Asset())
				if err != nil {
					continue
				}
				totalValue = totalValue.Add(pool.AssetValueInSWITCH(coin.Amount))
			}
		}
		labels := []metrics.Label{telemetry.NewLabel("vault_type", vault.Type.String()), telemetry.NewLabel("pubkey", vault.PubKey.String())}
		telemetry.SetGaugeWithLabels([]string{"switchlynode", "vault", "total_value"}, telem(totalValue), labels)

		for _, coin := range vault.Coins {
			vaultCoinLabel := []metrics.Label{
				telemetry.NewLabel("vault_type", vault.Type.String()),
				telemetry.NewLabel("pubkey", vault.PubKey.String()),
				telemetry.NewLabel("asset", coin.Asset.String()),
			}
			telemetry.SetGaugeWithLabels([]string{"switchlynode", "vault", "balance"}, telem(coin.Amount), vaultCoinLabel)
		}
	}

	// emit queue metrics
	signingTransactionPeriod := mgr.GetConstants().GetInt64Value(constants.SigningTransactionPeriod)
	startHeight := ctx.BlockHeight() - signingTransactionPeriod
	txOutDelayMax, err := mgr.Keeper().GetMimir(ctx, constants.TxOutDelayMax.String())
	if txOutDelayMax <= 0 || err != nil {
		txOutDelayMax = mgr.GetConstants().GetInt64Value(constants.TxOutDelayMax)
	}
	maxTxOutOffset, err := mgr.Keeper().GetMimir(ctx, constants.MaxTxOutOffset.String())
	if maxTxOutOffset <= 0 || err != nil {
		maxTxOutOffset = mgr.GetConstants().GetInt64Value(constants.MaxTxOutOffset)
	}
	var queueSwap, queueInternal, queueOutbound int64
	queueScheduledOutboundValue := cosmos.ZeroUint()
	iterator := mgr.Keeper().GetSwapQueueIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var msg MsgSwap
		if err := mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &msg); err != nil {
			continue
		}
		queueSwap++
	}
	for height := startHeight; height <= ctx.BlockHeight(); height++ {
		txs, err := mgr.Keeper().GetTxOut(ctx, height)
		if err != nil {
			continue
		}
		for _, tx := range txs.TxArray {
			if tx.OutHash.IsEmpty() {
				memo, _ := ParseMemo(mgr.GetVersion(), tx.Memo)
				if memo.IsInternal() {
					queueInternal++
				} else if memo.IsOutbound() {
					queueOutbound++
				}
			}
		}
	}
	cloutSpent := cosmos.ZeroUint()
	for height := ctx.BlockHeight() + 1; height <= ctx.BlockHeight()+txOutDelayMax; height++ {
		value, clout, err := mgr.Keeper().GetTxOutValue(ctx, height)
		if err != nil {
			ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
			continue
		}
		if height > ctx.BlockHeight()+maxTxOutOffset && value.IsZero() {
			// we've hit our max offset, and an empty block, we can assume the
			// rest will be empty as well
			break
		}
		queueScheduledOutboundValue = queueScheduledOutboundValue.Add(value)
		cloutSpent = cloutSpent.Add(clout)
	}
	telemetry.SetGauge(float32(queueInternal), "switchlynode", "queue", "internal")
	telemetry.SetGauge(float32(queueOutbound), "switchlynode", "queue", "outbound")
	telemetry.SetGauge(float32(queueSwap), "switchlynode", "queue", "swap")
	telemetry.SetGauge(telem(cloutSpent), "switchlynode", "queue", "scheduled", "clout", "switch")
	telemetry.SetGauge(telem(cloutSpent)*switchUSDPrice, "switchlynode", "queue", "scheduled", "clout", "usd")
	telemetry.SetGauge(telem(queueScheduledOutboundValue), "switchlynode", "queue", "scheduled", "value", "switch")
	telemetry.SetGauge(telem(queueScheduledOutboundValue)*switchUSDPrice, "switchlynode", "queue", "scheduled", "value", "usd")

	return nil
}

func getAvailablePoolsSWITCH(ctx cosmos.Context, keeper keeper.Keeper) (Pools, cosmos.Uint, error) {
	// Get Available layer 1 pools and sum their SWITCH balances.
	availablePoolsSWITCH := cosmos.ZeroUint()
	var availablePools Pools
	iterator := keeper.GetPoolIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		if err := keeper.Cdc().Unmarshal(iterator.Value(), &pool); err != nil {
			return nil, cosmos.ZeroUint(), fmt.Errorf("fail to unmarshal pool: %w", err)
		}
		if !pool.IsAvailable() {
			continue
		}
		if pool.Asset.IsNative() {
			continue
		}
		if pool.BalanceSwitch.IsZero() {
			continue
		}
		availablePoolsSWITCH = availablePoolsSWITCH.Add(pool.BalanceSwitch)
		availablePools = append(availablePools, pool)
	}
	return availablePools, availablePoolsSWITCH, nil
}

func getVaultsLiquiditySWITCH(ctx cosmos.Context, keeper keeper.Keeper) (cosmos.Uint, error) {
	// Sum the SWITCH values of non-Inactive vault Coins.
	vaultsLiquiditySWITCH := cosmos.ZeroUint()
	poolCache := map[common.Asset]Pool{}
	vaults, err := keeper.GetAsgardVaults(ctx)
	if err != nil {
		return cosmos.ZeroUint(), fmt.Errorf("fail to get vaults: %w", err)
	}
	for i := range vaults {
		// cleanupAsgardIndex removes InactiveVaults from the index on churn,
		// but RetiringVaults which become InactiveVaults and later receive inbounds
		// are not cleared from the index until the next churn,
		// so check nevertheless.
		// Similarly, an InactiveVault inbound (to be automatically refunded)
		// re-adds that InactiveVault to the Asgard Index with SetVault
		// until cleared again in the next churn.
		if vaults[i].Status == InactiveVault {
			continue
		}

		for _, coin := range vaults[i].Coins {
			if coin.IsSwitch() {
				vaultsLiquiditySWITCH = vaultsLiquiditySWITCH.Add(coin.Amount)
				continue
			}

			pool, ok := poolCache[coin.Asset]
			if !ok {
				pool, err = keeper.GetPool(ctx, coin.Asset)
				if err != nil {
					return cosmos.ZeroUint(), fmt.Errorf("fail to get pool for asset %s, err:%w", coin.Asset, err)
				}
				poolCache[coin.Asset] = pool
			}

			vaultsLiquiditySWITCH = vaultsLiquiditySWITCH.Add(pool.AssetValueInSWITCH(coin.Amount))
		}
	}
	return vaultsLiquiditySWITCH, nil
}

// get the total bond of the bottom 2/3rds active nodes
func getEffectiveSecurityBond(nas NodeAccounts) cosmos.Uint {
	amt := cosmos.ZeroUint()
	sort.SliceStable(nas, func(i, j int) bool {
		return nas[i].Bond.LT(nas[j].Bond)
	})
	t := len(nas) * 2 / 3
	if len(nas)%3 == 0 {
		t -= 1
	}
	for i, na := range nas {
		if i <= t {
			amt = amt.Add(na.Bond)
		}
	}
	return amt
}

// Calculates total "effective bond" - the total bond when taking into account the
// Bond-weighted hard-cap
func getTotalEffectiveBond(nas NodeAccounts) (cosmos.Uint, cosmos.Uint) {
	bondHardCap := getHardBondCap(nas)

	totalEffectiveBond := cosmos.ZeroUint()
	for _, item := range nas {
		b := item.Bond
		if item.Bond.GT(bondHardCap) {
			b = bondHardCap
		}

		totalEffectiveBond = totalEffectiveBond.Add(b)
	}

	return totalEffectiveBond, bondHardCap
}

// find the bond size the highest of the bottom 2/3rds node bonds
func getHardBondCap(nas NodeAccounts) cosmos.Uint {
	if len(nas) == 0 {
		return cosmos.ZeroUint()
	}
	sort.SliceStable(nas, func(i, j int) bool {
		return nas[i].Bond.LT(nas[j].Bond)
	})
	i := len(nas) * 2 / 3
	if len(nas)%3 == 0 {
		i -= 1
	}
	return nas[i].Bond
}

// From a list of (active) nodes, get a list of those not in a list (of signers).
func getNonSigners(nas []NodeAccount, signers []cosmos.AccAddress) []cosmos.AccAddress {
	var nonSigners []cosmos.AccAddress
	var signed bool

	for _, na := range nas {
		signed = false
		for _, signer := range signers {
			if na.NodeAddress.Equals(signer) {
				signed = true
				break
			}
		}

		if !signed {
			nonSigners = append(nonSigners, na.NodeAddress)
		}
	}
	return nonSigners
}

// In the case where the max gas of the chain of a queued outbound tx has changed
// Update the ObservedTxVoter so the network can still match the outbound with
// the observed inbound
func updateTxOutGas(ctx cosmos.Context, keeper keeper.Keeper, txOut types.TxOutItem, gas common.Gas) error {
	// When txOut.InHash is 0000000000000000000000000000000000000000000000000000000000000000 , which means the outbound is trigger by the network internally
	// For example , migration, etc. there is no related inbound observation , thus doesn't need to try to find it and update anything
	if txOut.InHash == common.BlankTxID {
		return nil
	}
	voter, err := keeper.GetObservedTxInVoter(ctx, txOut.InHash)
	if err != nil {
		return err
	}

	txOutIndex := -1
	for i, tx := range voter.Actions {
		if tx.Equals(txOut) {
			txOutIndex = i
			voter.Actions[txOutIndex].MaxGas = gas
			keeper.SetObservedTxInVoter(ctx, voter)
			break
		}
	}

	if txOutIndex == -1 {
		return fmt.Errorf("fail to find tx out in ObservedTxVoter %s", txOut.InHash)
	}

	return nil
}

// In the case where the gas rate of the chain of a queued outbound tx has changed
// Update the ObservedTxVoter so the network can still match the outbound with
// the observed inbound
func updateTxOutGasRate(ctx cosmos.Context, keeper keeper.Keeper, txOut types.TxOutItem, gasRate int64) error {
	// When txOut.InHash is 0000000000000000000000000000000000000000000000000000000000000000 , which means the outbound is trigger by the network internally
	// For example , migration, etc. there is no related inbound observation , thus doesn't need to try to find it and update anything
	if txOut.InHash == common.BlankTxID {
		return nil
	}
	voter, err := keeper.GetObservedTxInVoter(ctx, txOut.InHash)
	if err != nil {
		return err
	}

	txOutIndex := -1
	for i, tx := range voter.Actions {
		if tx.Equals(txOut) {
			txOutIndex = i
			voter.Actions[txOutIndex].GasRate = gasRate
			keeper.SetObservedTxInVoter(ctx, voter)
			break
		}
	}

	if txOutIndex == -1 {
		return fmt.Errorf("fail to find tx out in ObservedTxVoter %s", txOut.InHash)
	}

	return nil
}

// backfill bond provider information (passive migration code)
func passiveBackfill(ctx cosmos.Context, mgr Manager, nodeAccount NodeAccount, bp *BondProviders) error {
	if len(bp.Providers) == 0 {
		// no providers yet, add node operator bond address to the bond provider list
		nodeOpBondAddr, err := nodeAccount.BondAddress.AccAddress()
		if err != nil {
			return ErrInternal(err, fmt.Sprintf("fail to parse bond address(%s)", nodeAccount.BondAddress))
		}
		p := NewBondProvider(nodeOpBondAddr)
		p.Bond = nodeAccount.Bond
		bp.Providers = append(bp.Providers, p)
		defaultNodeOperationFee := mgr.Keeper().GetConfigInt64(ctx, constants.NodeOperatorFee)
		bp.NodeOperatorFee = cosmos.NewUint(uint64(defaultNodeOperationFee))
	}

	return nil
}

// atTVLCap - returns bool on if we've hit the TVL hard cap. Coins passed in
// are included in the calculation
func atTVLCap(ctx cosmos.Context, coins common.Coins, mgr Manager) bool {
	vaults, err := mgr.Keeper().GetAsgardVaults(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get vaults for atTVLCap", "error", err)
		return true
	}

	// coins must be copied to a new variable to avoid modifying the original
	coins = coins.Copy()
	for _, vault := range vaults {
		if vault.IsAsgard() && (vault.IsActive() || vault.IsRetiring()) {
			coins = coins.Add(vault.Coins...)
		}
	}

	switchCoin := coins.GetCoin(common.SwitchNative)
	totalSWITCHValue := switchCoin.Amount
	for _, coin := range coins {
		if coin.IsEmpty() {
			continue
		}
		asset := coin.Asset
		// while asgard vaults don't contain native assets, the `coins`
		// parameter might
		if asset.IsSyntheticAsset() {
			asset = asset.GetLayer1Asset()
		}
		pool, err := mgr.Keeper().GetPool(ctx, asset)
		if err != nil {
			ctx.Logger().Error("fail to get pool for atTVLCap", "asset", coin.Asset, "error", err)
			continue
		}
		if !pool.IsAvailable() && !pool.IsStaged() {
			continue
		}
		if pool.BalanceSwitch.IsZero() || pool.BalanceAsset.IsZero() {
			continue
		}
		if pool.Asset.IsNative() {
			continue
		}
		totalSWITCHValue = totalSWITCHValue.Add(pool.AssetValueInSWITCH(coin.Amount))
	}

	// get effectiveSecurity
	nodeAccounts, err := mgr.Keeper().ListActiveValidators(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get validators to calculate TVL cap", "error", err)
		return true
	}

	tvlCapBasisPoints := mgr.Keeper().GetConfigInt64(ctx, constants.TVLCapBasisPoints)
	security := cosmos.ZeroUint()
	if tvlCapBasisPoints > 0 {
		for _, na := range nodeAccounts {
			security = security.Add(na.Bond)
		}
		security = common.GetUncappedShare(cosmos.NewUint(uint64(tvlCapBasisPoints)), cosmos.NewUint(constants.MaxBasisPts), security)
	} else {
		security = getEffectiveSecurityBond(nodeAccounts)
	}

	if totalSWITCHValue.GT(security) {
		ctx.Logger().Debug("reached TVL cap", "total switch value", totalSWITCHValue.String(), "security", security.String())
		return true
	}
	return false
}

// trunk-ignore(golangci-lint/unused): used by store helper
func isActionsItemDangling(voter ObservedTxVoter, i int) bool {
	if i < 0 || i > len(voter.Actions)-1 {
		// No such Actions item exists in the voter.
		return false
	}

	toi := voter.Actions[i]

	// If any OutTxs item matches an Actions item, deem it to be not dangling.
	for _, outboundTx := range voter.OutTxs {
		// The comparison code is based on matchActionItem, as matchActionItem is unimportable.
		// note: Coins.Contains will match amount as well
		matchCoin := outboundTx.Coins.Contains(toi.Coin)
		if !matchCoin && toi.Coin.Asset.Equals(toi.Chain.GetGasAsset()) {
			asset := toi.Chain.GetGasAsset()
			intendToSpend := toi.Coin.Amount.Add(toi.MaxGas.ToCoins().GetCoin(asset).Amount)
			actualSpend := outboundTx.Coins.GetCoin(asset).Amount.Add(outboundTx.Gas.ToCoins().GetCoin(asset).Amount)
			if intendToSpend.Equal(actualSpend) {
				matchCoin = true
			}
		}
		if strings.EqualFold(toi.Memo, outboundTx.Memo) &&
			toi.ToAddress.Equals(outboundTx.ToAddress) &&
			toi.Chain.Equals(outboundTx.Chain) &&
			matchCoin {
			return false
		}
	}
	return true
}

func IsModuleAccAddress(keeper keeper.Keeper, accAddr cosmos.AccAddress) bool {
	return accAddr.Equals(keeper.GetModuleAccAddress(AsgardName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(BondName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(ReserveName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(LendingName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(AffiliateCollectorName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(ModuleName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(SWCYClaimingName)) ||
		accAddr.Equals(keeper.GetModuleAccAddress(SWCYStakeName))
}

func NewSwapMemo(ctx cosmos.Context, mgr Manager, targetAsset common.Asset, destination common.Address, limit cosmos.Uint, affiliate string, affiliateBps cosmos.Uint) string {
	return fmt.Sprintf("=:%s:%s:%s:%s:%s", targetAsset, destination, limit.String(), affiliate, affiliateBps.String())
}

// willSwapOutputExceedLimitAndFees returns true if the swap output will exceed the
// limit (if provided) + the outbound fee on the destination chain
func willSwapOutputExceedLimitAndFees(ctx cosmos.Context, mgr Manager, msg MsgSwap) bool {
	swapper, err := GetSwapper(mgr.GetVersion())
	if err != nil {
		panic(err)
	}

	source := msg.Tx.Coins[0]
	target := common.NewCoin(msg.TargetAsset, msg.TradeTarget)

	var emit cosmos.Uint
	switch {
	case !source.IsSwitch() && !target.IsSwitch():
		sourcePool, err := mgr.Keeper().GetPool(ctx, source.Asset.GetLayer1Asset())
		if err != nil {
			return false
		}
		targetPool, err := mgr.Keeper().GetPool(ctx, target.Asset.GetLayer1Asset())
		if err != nil {
			return false
		}
		emit = swapper.CalcAssetEmission(sourcePool.BalanceAsset, source.Amount, sourcePool.BalanceSwitch)
		emit = swapper.CalcAssetEmission(targetPool.BalanceSwitch, emit, targetPool.BalanceAsset)
	case source.IsSwitch():
		pool, err := mgr.Keeper().GetPool(ctx, target.Asset.GetLayer1Asset())
		if err != nil {
			return false
		}
		emit = swapper.CalcAssetEmission(pool.BalanceSwitch, source.Amount, pool.BalanceAsset)
	case target.IsSwitch():
		pool, err := mgr.Keeper().GetPool(ctx, source.Asset.GetLayer1Asset())
		if err != nil {
			return false
		}
		emit = swapper.CalcAssetEmission(pool.BalanceAsset, source.Amount, pool.BalanceSwitch)
	}

	// Check that the swap will emit more than the limit amount + outbound fee
	transactionFeeAsset, err := mgr.GasMgr().GetAssetOutboundFee(ctx, msg.TargetAsset, false)
	return err == nil && emit.GT(target.Amount.Add(transactionFeeAsset))
}

////////////////////////////////////////////////////////////////////////////////////////
// SwitchPool and POL
////////////////////////////////////////////////////////////////////////////////////////

// reserveExitSwitchPool will release as much reserve ownership of the switchpool as
// possible to providers. The amount is limited by the reserve units and pending switch -
// whichever is less. The ownership units are adjusted and a corresponding amount of
// switch is transferred from the switchpool module to the reserve.
func reserveExitSwitchPool(ctx cosmos.Context, mgr Manager) error {
	switchPool, err := mgr.Keeper().GetSwitchPool(ctx)
	if err != nil {
		return fmt.Errorf("unable to get switchpool: %w", err)
	}

	pendingSWITCH := mgr.Keeper().GetSWITCHBalanceOfModule(ctx, SwitchPoolName)

	// if the reserve owns no pol or there are no pending switch, nothing to do
	if switchPool.ReserveUnits.IsZero() || pendingSWITCH.IsZero() {
		return nil
	}

	// the reserve will exit as much as possible, limited by reserve share or pending switch
	reserveExitSWITCH := pendingSWITCH
	switchPoolValue, err := switchPoolValue(ctx, mgr)
	if err != nil {
		return fmt.Errorf("fail to get pol pool value: %w", err)
	}
	reserveSWITCHpoolValue := common.GetSafeShare(switchPool.ReserveUnits, switchPool.TotalUnits(), switchPoolValue)
	if reserveSWITCHpoolValue.LT(reserveExitSWITCH) {
		reserveExitSWITCH = reserveSWITCHpoolValue
	}
	reserveExitUnits := common.GetSafeShare(reserveExitSWITCH, reserveSWITCHpoolValue, switchPool.ReserveUnits)

	// transfer the corresponding switch from switchpool module to reserve
	reserveExitCoins := common.NewCoins(common.NewCoin(common.SwitchNative, reserveExitSWITCH))
	err = mgr.Keeper().SendFromModuleToModule(ctx, SwitchPoolName, ReserveName, reserveExitCoins)
	if err != nil {
		return fmt.Errorf("unable to SendFromModuleToModule: %s", err)
	}

	ctx.Logger().Info("reserve exited switchpool", "units", reserveExitUnits, "switch", reserveExitSWITCH)

	// update switchpool units
	switchPool.ReserveUnits = common.SafeSub(switchPool.ReserveUnits, reserveExitUnits)
	mgr.Keeper().SetSwitchPool(ctx, switchPool)

	return nil
}

// reserveEnterSwitchPool will acquire the provided switch value of the switchpool from the
// providers to the reserve. The ownership units are adjusted and a corresponding amount
// of switch is transferred from the reserve to the switchpool module. This allows the
// reserve to reclaim units to allow providers to withdraw.
func reserveEnterSwitchPool(ctx cosmos.Context, mgr Manager, switchAmount cosmos.Uint) error {
	switchPool, err := mgr.Keeper().GetSwitchPool(ctx)
	if err != nil {
		return fmt.Errorf("fail to get switchpool: %w", err)
	}

	// determine the switch value of the units
	switchPoolValue, err := switchPoolValue(ctx, mgr)
	if err != nil {
		return fmt.Errorf("fail to get switchpool value: %w", err)
	}
	units := common.GetSafeShare(switchAmount, switchPoolValue, switchPool.TotalUnits())
	coins := common.NewCoins(common.NewCoin(common.SwitchNative, switchAmount))

	// transfer the switch from the reserve to the switchpool
	err = mgr.Keeper().SendFromModuleToModule(ctx, ReserveName, SwitchPoolName, coins)
	if err != nil {
		return fmt.Errorf("fail to transfer switch from reserve to switchpool: %w", err)
	}

	ctx.Logger().Info("reserve entered switchpool", "units", units, "switch", switchAmount)

	// update switchpool units
	switchPool.ReserveUnits = switchPool.ReserveUnits.Add(units)
	mgr.Keeper().SetSwitchPool(ctx, switchPool)

	return nil
}

// switchPoolValue is the POL value in SWITCH plus undeployed SWITCH in the switchpool module.
func switchPoolValue(ctx cosmos.Context, mgr Manager) (cosmos.Uint, error) {
	polValue, err := polPoolValue(ctx, mgr)
	if err != nil {
		return cosmos.ZeroUint(), err
	}

	// get the total switch in the switchpool module
	pending := mgr.Keeper().GetSWITCHBalanceOfModule(ctx, SwitchPoolName)

	return polValue.Add(pending), nil
}

// polPoolValue - calculates how much the POL is worth in switch
func polPoolValue(ctx cosmos.Context, mgr Manager) (cosmos.Uint, error) {
	total := cosmos.ZeroUint()

	polAddress, err := mgr.Keeper().GetModuleAddress(ReserveName)
	if err != nil {
		return total, err
	}

	pools, err := mgr.Keeper().GetPools(ctx)
	if err != nil {
		return total, err
	}
	for _, pool := range pools {
		if pool.Asset.IsNative() {
			continue
		}
		if pool.BalanceSwitch.IsZero() {
			continue
		}
		synthSupply := mgr.Keeper().GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
		pool.CalcUnits(synthSupply)
		lp, err := mgr.Keeper().GetLiquidityProvider(ctx, pool.Asset, polAddress)
		if err != nil {
			return total, err
		}
		share := common.GetSafeShare(lp.Units, pool.GetPoolUnits(), pool.BalanceSwitch)
		total = total.Add(share.MulUint64(2))
	}

	return total, nil
}

// This removes the first prefix ending with "//" (if there is one) from a KVStore key,
// such as when obtained through an Iterator, whatever the prefix may be.
// The "/" at the end of every prefix, together with the "/" added by KVStore GetKey
// (to ensure that no prefix ever contains another prefix)
// should ensure that each prefix ends with "//".
func trimKeyPrefix(key []byte) string {
	keyString := string(key)
	if _, after, found := strings.Cut(keyString, "//"); found {
		return after
	}
	return keyString
}

func IsPeriodLastBlock(ctx cosmos.Context, blocksPerPeriod int64) bool {
	return ctx.BlockHeight()%blocksPerPeriod == 0
}
