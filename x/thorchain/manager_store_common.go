package thorchain

import (
	"crypto/sha256"
	"fmt"

	"gitlab.com/thorchain/thornode/common"
	"gitlab.com/thorchain/thornode/common/cosmos"
	"gitlab.com/thorchain/thornode/constants"
	"gitlab.com/thorchain/thornode/x/thorchain/keeper"
)

// trunk-ignore-all(golangci-lint/unused): may be used in future

// discoverOutbounds is used to select one more TxOutItem with an appropriate VaultPubKey(s) selected for a given TxOutItem
// this will split a large TxOutItem into multiple smaller []TxOutItem needed to fulfill the original TxOutItem
// it should be used when issuing a manual refund in store migrations
func discoverOutbounds(ctx cosmos.Context, mgr *Mgrs, toi TxOutItem) ([]TxOutItem, error) {
	signingTransactionPeriod := mgr.GetConstants().GetInt64Value(constants.SigningTransactionPeriod)
	transactionFee, err := mgr.GasMgr().GetAssetOutboundFee(ctx, toi.Coin.Asset, false)
	if err != nil {
		return []TxOutItem{}, fmt.Errorf("fail to get outbound fee: %w", err)
	}
	maxGasAsset, err := mgr.GasMgr().GetMaxGas(ctx, toi.Chain)
	if err != nil {
		ctx.Logger().Error("fail to get max gas asset", "error", err)
	}

	pendingOutbounds := mgr.Keeper().GetPendingOutbounds(ctx, toi.Coin.Asset)

	// ///////////// COLLECT ACTIVE ASGARD VAULTS ///////////////////
	activeAsgards, err := mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		ctx.Logger().Error("fail to get active vaults", "error", err)
	}

	// All else being equal, prefer lower-security vaults for outbounds.
	activeAsgards = mgr.Keeper().SortBySecurity(ctx, activeAsgards, signingTransactionPeriod)

	for i := range activeAsgards {
		// having sorted by security, deduct the value of any assigned pending outbounds
		activeAsgards[i].DeductVaultPendingOutbounds(pendingOutbounds)
	}
	// //////////////////////////////////////////////////////////////

	// ///////////// COLLECT RETIRING ASGARD VAULTS /////////////////
	retiringAsgards, err := mgr.Keeper().GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		ctx.Logger().Error("fail to get retiring vaults", "error", err)
	}

	// All else being equal, prefer lower-security vaults for outbounds.
	retiringAsgards = mgr.Keeper().SortBySecurity(ctx, retiringAsgards, signingTransactionPeriod)

	for i := range retiringAsgards {
		// having sorted by security, deduct the value of any assigned pending outbounds
		retiringAsgards[i].DeductVaultPendingOutbounds(pendingOutbounds)
	}
	// //////////////////////////////////////////////////////////////

	// iterate over discovered vaults and find vaults to send funds from

	// All else being equal, choose active Asgards over retiring Asgards.
	outputs, remaining := mgr.TxOutStore().DiscoverOutbounds(ctx, transactionFee, maxGasAsset, toi, append(activeAsgards, retiringAsgards...))

	// Check we found enough funds to satisfy the request, error if we didn't
	if !remaining.IsZero() {
		return []TxOutItem{}, fmt.Errorf("insufficient funds for outbound request: %s %s remaining", toi.ToAddress.String(), remaining.String())
	}
	return outputs, nil
}

// removeTransactions is a method used to remove a tx out item in the queue
func removeTransactions(ctx cosmos.Context, mgr Manager, hashes ...string) {
	for _, txID := range hashes {
		inTxID, err := common.NewTxID(txID)
		if err != nil {
			ctx.Logger().Error("fail to parse tx id", "error", err, "tx_id", inTxID)
			continue
		}
		voter, err := mgr.Keeper().GetObservedTxInVoter(ctx, inTxID)
		if err != nil {
			ctx.Logger().Error("fail to get observed tx voter", "error", err)
			continue
		}
		// all outbound action get removed
		voter.Actions = []TxOutItem{}
		if voter.Tx.IsEmpty() {
			continue
		}
		voter.Tx.SetDone(common.BlankTxID, 0)
		// set the tx outbound with a blank txid will mark it as down , and will be skipped in the reschedule logic
		for idx := range voter.Txs {
			voter.Txs[idx].SetDone(common.BlankTxID, 0)
		}
		mgr.Keeper().SetObservedTxInVoter(ctx, voter)
	}
}

type DroppedSwapOutTx struct {
	inboundHash string
	gasAsset    common.Asset
}

// refundDroppedSwapOutFromRUNE refunds a dropped swap out TX that originated from $RUNE

// These txs completed the swap to the EVM gas asset, but bifrost dropped the final swap out outbound
// To refund:
// 1. Credit the gas asset pool the amount of gas asset that never left
// 2. Deduct the corresponding amount of RUNE from the pool, as that will be refunded
// 3. Send the user their RUNE back
func refundDroppedSwapOutFromRUNE(ctx cosmos.Context, mgr *Mgrs, droppedTx DroppedSwapOutTx) error {
	txId, err := common.NewTxID(droppedTx.inboundHash)
	if err != nil {
		return err
	}

	txVoter, err := mgr.Keeper().GetObservedTxInVoter(ctx, txId)
	if err != nil {
		return err
	}

	if txVoter.OutTxs != nil {
		return fmt.Errorf("For a dropped swap out there should be no out_txs")
	}

	// Get the original inbound, if it's not for RUNE, skip
	inboundTx := txVoter.Tx.Tx
	if !inboundTx.Chain.IsTHORChain() {
		return fmt.Errorf("Inbound tx isn't from thorchain")
	}

	inboundCoins := inboundTx.Coins
	if len(inboundCoins) != 1 || !inboundCoins[0].IsRune() {
		return fmt.Errorf("Inbound coin is not native RUNE")
	}

	inboundRUNE := inboundCoins[0]
	swapperRUNEAddr := inboundTx.FromAddress

	if txVoter.Actions == nil || len(txVoter.Actions) == 0 {
		return fmt.Errorf("Tx Voter has empty Actions")
	}

	// gasAssetCoin is the gas asset that was swapped to for the swap out
	// Since the swap out was dropped, this amount of the gas asset never left the pool.
	// This amount should be credited back to the pool since it was originally deducted when thornode sent the swap out
	gasAssetCoin := txVoter.Actions[0].Coin
	if !gasAssetCoin.Asset.Equals(droppedTx.gasAsset) {
		return fmt.Errorf("Tx Voter action coin isn't swap out gas asset")
	}

	gasPool, err := mgr.Keeper().GetPool(ctx, droppedTx.gasAsset)
	if err != nil {
		return err
	}

	totalGasAssetAmt := cosmos.NewUint(0)

	// If the outbound was split between multiple Asgards, add up the full amount here
	for _, action := range txVoter.Actions {
		totalGasAssetAmt = totalGasAssetAmt.Add(action.Coin.Amount)
	}

	// Credit Gas Pool the Gas Asset balance, deduct the RUNE balance
	gasPool.BalanceAsset = gasPool.BalanceAsset.Add(totalGasAssetAmt)
	gasPool.BalanceRune = gasPool.BalanceRune.Sub(inboundRUNE.Amount)

	// Update the pool
	if err := mgr.Keeper().SetPool(ctx, gasPool); err != nil {
		return err
	}

	addrAcct, err := swapperRUNEAddr.AccAddress()
	if err != nil {
		ctx.Logger().Error("fail to create acct in migrate store to v98", "error", err)
	}

	runeCoins := common.NewCoins(inboundRUNE)

	// Send user their funds
	err = mgr.Keeper().SendFromModuleToAccount(ctx, AsgardName, addrAcct, runeCoins)
	if err != nil {
		return err
	}

	memo := fmt.Sprintf("REFUND:%s", inboundTx.ID)

	// Generate a fake TxID from the refund memo for Midgard to record.
	// Since the inbound hash is expected to be unique, the sha256 hash is expected to be unique.
	hash := fmt.Sprintf("%X", sha256.Sum256([]byte(memo)))
	fakeTxID, err := common.NewTxID(hash)
	if err != nil {
		return err
	}

	// create and emit a fake tx and swap event to keep pools balanced in Midgard
	fakeSwapTx := common.Tx{
		ID:          fakeTxID,
		Chain:       common.ETHChain,
		FromAddress: txVoter.Actions[0].ToAddress,
		ToAddress:   common.Address(txVoter.Actions[0].Aggregator),
		Coins:       common.NewCoins(gasAssetCoin),
		Memo:        memo,
	}

	swapEvt := NewEventSwap(
		droppedTx.gasAsset,
		cosmos.ZeroUint(),
		cosmos.ZeroUint(),
		cosmos.ZeroUint(),
		cosmos.ZeroUint(),
		fakeSwapTx,
		inboundRUNE,
		cosmos.ZeroUint(),
	)

	if err := mgr.EventMgr().EmitEvent(ctx, swapEvt); err != nil {
		ctx.Logger().Error("fail to emit fake swap event", "error", err)
	}

	return nil
}

// When an ObservedTxInVoter has dangling Actions items swallowed by the vaults, requeue
// them. The voter.OutboundHeight should be set so that the outbound is not considered "expired",
// which causes the nodes to be slashed and requeed outbound to remain in the outbound queue.
func requeueDanglingActions(ctx cosmos.Context, mgr *Mgrs, txIDs []common.TxID) {
	// Select the least secure ActiveVault Asgard for all outbounds.
	// Even if it fails (as in if the version changed upon the keygens-complete block of a churn),
	// updating the voter's FinalisedHeight allows another MaxOutboundAttempts for LackSigning vault selection.
	activeAsgards, err := mgr.Keeper().GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil || len(activeAsgards) == 0 {
		ctx.Logger().Error("fail to get active asgard vaults", "error", err)
		return
	}
	if len(activeAsgards) > 1 {
		signingTransactionPeriod := mgr.GetConstants().GetInt64Value(constants.SigningTransactionPeriod)
		activeAsgards = mgr.Keeper().SortBySecurity(ctx, activeAsgards, signingTransactionPeriod)
	}
	vaultPubKey := activeAsgards[0].PubKey

	for _, txID := range txIDs {
		voter, err := mgr.Keeper().GetObservedTxInVoter(ctx, txID)
		if err != nil {
			ctx.Logger().Error("fail to get observed tx voter", "error", err)
			continue
		}

		if len(voter.OutTxs) >= len(voter.Actions) {
			log := fmt.Sprintf("(%d) OutTxs present for (%s), despite expecting fewer than the (%d) Actions.", len(voter.OutTxs), txID.String(), len(voter.Actions))
			ctx.Logger().Debug(log)
			continue
		}

		var indices []int
		for i := range voter.Actions {
			if isActionsItemDangling(voter, i) {
				indices = append(indices, i)
			}
		}
		if len(indices) == 0 {
			log := fmt.Sprintf("No dangling Actions item found for (%s)", txID.String())
			ctx.Logger().Debug(log)
			continue
		}

		if len(voter.Actions)-len(voter.OutTxs) != len(indices) {
			log := fmt.Sprintf("(%d) Actions and (%d) OutTxs present for (%s), yet there appeared to be (%d) dangling Actions.", len(voter.Actions), len(voter.OutTxs), txID.String(), len(indices))
			ctx.Logger().Debug(log)
			continue
		}

		// Update the voter's FinalisedHeight to give another MaxOutboundAttempts.
		voter.FinalisedHeight = ctx.BlockHeight()
		voter.OutboundHeight = ctx.BlockHeight()

		for _, index := range indices {
			// Use a pointer to update the voter as well.
			actionItem := &voter.Actions[index]

			// Update the vault pubkey.
			actionItem.VaultPubKey = vaultPubKey

			// Update the Actions item's MaxGas and GasRate.
			// Note that nothing in this function should require a GasManager BeginBlock.
			gasCoin, err := mgr.GasMgr().GetMaxGas(ctx, actionItem.Chain)
			if err != nil {
				ctx.Logger().Error("fail to get max gas", "chain", actionItem.Chain, "error", err)
				continue
			}
			actionItem.MaxGas = common.Gas{gasCoin}
			actionItem.GasRate = int64(mgr.GasMgr().GetGasRate(ctx, actionItem.Chain).Uint64())

			// UnSafeAddTxOutItem is used to queue the txout item directly, without for instance deducting another fee.
			err = mgr.TxOutStore().UnSafeAddTxOutItem(ctx, mgr, *actionItem, ctx.BlockHeight())
			if err != nil {
				ctx.Logger().Error("fail to add outbound tx", "error", err)
				continue
			}
		}

		// Having requeued all dangling Actions items, set the updated voter.
		mgr.Keeper().SetObservedTxInVoter(ctx, voter)
	}
}

// makeFakeTxInObservation - accepts an array of unobserved inbounds, queries for active node accounts, and makes
// a fake observation for each validator and unobserved TxIn. Once enough nodes have "observed" each inbound the tx will be
// processed as normal.
func makeFakeTxInObservation(ctx cosmos.Context, mgr *Mgrs, txs ObservedTxs) error {
	activeNodes, err := mgr.Keeper().ListActiveValidators(ctx)
	if err != nil {
		ctx.Logger().Error("Failed to get active nodes", "err", err)
		return err
	}

	handler := NewObservedTxInHandler(mgr)

	for _, na := range activeNodes {
		txInMsg := NewMsgObservedTxIn(txs, na.NodeAddress)
		_, err := handler.handle(ctx, *txInMsg)
		if err != nil {
			ctx.Logger().Error("failed ObservedTxIn handler", "error", err)
			continue
		}
	}

	return nil
}

// resetObservationHeights will force reset the last chain and last observed heights for
// all active nodes.
func resetObservationHeights(ctx cosmos.Context, mgr *Mgrs, version int, chain common.Chain, height int64) {
	defer func() {
		if err := recover(); err != nil {
			ctx.Logger().Error(fmt.Sprintf("fail to migrate store to v%d", version), "error", err)
		}
	}()

	// get active nodes
	activeNodes, err := mgr.Keeper().ListActiveValidators(ctx)
	if err != nil {
		ctx.Logger().Error("failed to get active nodes", "err", err)
		return
	}

	// force set last observed height on all nodes
	for _, node := range activeNodes {
		mgr.Keeper().ForceSetLastObserveHeight(ctx, chain, node.NodeAddress, height)
	}

	// force set chain height
	mgr.Keeper().ForceSetLastChainHeight(ctx, chain, height)
}

// send RuneToTransfer from ModuleName to RuneRecipient
// burn SynthsToBurn from ModuleName
type ModuleBalanceAction struct {
	ModuleName     string
	RuneRecipient  string
	RuneToTransfer cosmos.Uint
	SynthsToBurn   common.Coins
}

func processModuleBalanceActions(ctx cosmos.Context, k keeper.Keeper, actions []ModuleBalanceAction) {
	for _, action := range actions {
		// transfer rune between modules if required
		if !action.RuneToTransfer.IsZero() {
			coin := common.NewCoin(common.RuneAsset(), action.RuneToTransfer)
			coins := common.NewCoins(coin)
			err := k.SendFromModuleToModule(ctx, action.ModuleName, action.RuneRecipient, coins)
			if err != nil {
				ctx.Logger().Error("fail to migrate rune", err)
			}
		}

		// burn synths from module if required
		if len(action.SynthsToBurn) > 0 {
			err := k.SendFromModuleToModule(ctx, action.ModuleName, ModuleName, action.SynthsToBurn)
			if err != nil {
				ctx.Logger().Error("fail to migrate synths", err)
				continue
			}
			for _, coin := range action.SynthsToBurn {
				err := k.BurnFromModule(ctx, ModuleName, coin)
				if err != nil {
					ctx.Logger().Error("fail to burn migrated synths", err)
				}
			}
		}
	}
}

// changeLPOwnership finds all LPs with oldOwner rune_address and changes to newOwner
// This helper also zeroes out the asset_address, effectively converting the LP to asymmetric
func changeLPOwnership(ctx cosmos.Context, mgr *Mgrs, oldOwner common.Address, newOwner common.Address) {
	pools, err := mgr.Keeper().GetPools(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get pools", "error", err)
		return
	}

	for _, pool := range pools {
		iterator := mgr.Keeper().GetLiquidityProviderIterator(ctx, pool.Asset)
		defer iterator.Close()
		for ; iterator.Valid(); iterator.Next() {
			var lp LiquidityProvider
			if err := mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &lp); err != nil {
				ctx.Logger().Error("fail to get unmarshal LP", "error", err)
				return
			}
			// Existing treasury RUNE address
			if lp.RuneAddress == oldOwner {
				// LPs cannot upsert RUNE address. Delete existing LP and re-create.
				mgr.Keeper().RemoveLiquidityProvider(ctx, lp)

				// Update LP RUNE address to Treasury module RUNE address
				lp.RuneAddress = newOwner
				// Unset AssetAddress, effectively converting LP to asymmetric (RUNE-only).
				// N.B. this does not change pool ratio or LP amounts
				lp.AssetAddress = ""
				mgr.Keeper().SetLiquidityProvider(ctx, lp)
			}
		}
	}
}

func restoreTotalCollateral(ctx cosmos.Context, mgr *Mgrs) {
	assets := []common.Asset{common.BTCAsset, common.ETHAsset}
	for _, asset := range assets {
		total := cosmos.ZeroUint()
		it := mgr.Keeper().GetLoanIterator(ctx, asset)
		defer it.Close()
		for ; it.Valid(); it.Next() {
			var loan Loan
			mgr.Keeper().Cdc().MustUnmarshal(it.Value(), &loan)
			total = total.Add(loan.CollateralDeposited)
			total = common.SafeSub(total, loan.CollateralWithdrawn)
		}
		mgr.Keeper().SetTotalCollateral(ctx, asset, total)
	}
}
