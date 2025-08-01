package thorchain

import (
	"errors"
	"fmt"
	"net"
	"sort"

	abci "github.com/cometbft/cometbft/abci/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
)

// ValidatorMgrVCUR is to manage a list of validators , and rotate them
type ValidatorMgrVCUR struct {
	k                  keeper.Keeper
	networkMgr         NetworkManager
	txOutStore         TxOutStore
	eventMgr           EventManager
	existingValidators []string
}

// newValidatorMgrVCUR create a new instance of ValidatorMgrVCUR
func newValidatorMgrVCUR(k keeper.Keeper, networkMgr NetworkManager, txOutStore TxOutStore, eventMgr EventManager) *ValidatorMgrVCUR {
	return &ValidatorMgrVCUR{
		k:          k,
		networkMgr: networkMgr,
		txOutStore: txOutStore,
		eventMgr:   eventMgr,
	}
}

// BeginBlock when block begin
func (vm *ValidatorMgrVCUR) BeginBlock(ctx cosmos.Context, mgr Manager, existingValidators []string) error {
	vm.existingValidators = existingValidators
	height := ctx.BlockHeight()
	if height == genesisBlockHeight {
		if err := vm.setupValidatorNodes(ctx, height); err != nil {
			ctx.Logger().Error("fail to setup validator nodes", "error", err)
		}
	}
	if vm.k.RagnarokInProgress(ctx) {
		// ragnarok is in progress, no point to check node rotation
		return nil
	}

	lastChurnHeight := vm.getLastChurnHeight(ctx)
	churnInterval := vm.k.GetConfigInt64(ctx, constants.ChurnInterval)
	churnRetryInterval := vm.k.GetConfigInt64(ctx, constants.ChurnRetryInterval)
	onChurnTick := (ctx.BlockHeight()-lastChurnHeight-churnInterval)%churnRetryInterval == 0
	if !onChurnTick {
		return nil
	}

	halt, err := vm.k.GetMimir(ctx, "HaltChurning")
	if halt > 0 && halt <= ctx.BlockHeight() && err == nil {
		ctx.Logger().Info("churn event skipped due to mimir has halted churning")
		return nil
	}

	vaults, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		ctx.Logger().Error("Failed to get Asgard vaults", "error", err)
		return err
	}

	// calculate if we need to retry a churn because we are overdue for a
	// successful one
	nas, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return err
	}
	asgardSize := vm.k.GetConfigInt64(ctx, constants.AsgardSize)
	expectedActiveVaults := int64(len(nas)) / asgardSize
	if int64(len(nas))%asgardSize > 0 {
		expectedActiveVaults++
	}
	incompleteChurnCheck := int64(len(vaults)) != expectedActiveVaults
	oldVaultCheck := ctx.BlockHeight()-lastChurnHeight > churnInterval
	retryChurn := (oldVaultCheck || incompleteChurnCheck) && onChurnTick

	// skip churn if any active chain is halted
	shouldChurn := lastChurnHeight+churnInterval == ctx.BlockHeight() || retryChurn
	if !shouldChurn {
		return nil
	}

	// collect all chains for active vaults
	activeChains := make(common.Chains, 0)
	for _, v := range vaults {
		activeChains = append(activeChains, v.GetChains()...)
	}
	activeChains = activeChains.Distinct()

	for _, chain := range activeChains {
		if mgr.Keeper().IsChainHalted(ctx, chain) {
			ctx.Logger().Info("Skipping node account rotation for halted chain", "chain", chain)
			return nil
		}
	}

	// don't churn if we have retiring asgard vaults that still have funds
	retiringVaults, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
	if err != nil {
		return err
	}
	if len(retiringVaults) > 0 {
		ctx.Logger().Info("Skipping rotation due to retiring vaults still have funds.")
		return nil
	}

	if retryChurn {
		ctx.Logger().Info("Checking for node account rotation... (retry)")
	} else {
		ctx.Logger().Info("Checking for node account rotation...")
	}
	return vm.churn(ctx)
}

func (vm *ValidatorMgrVCUR) churn(ctx cosmos.Context) error {
	desiredValidatorSet := vm.k.GetConfigInt64(ctx, constants.DesiredValidatorSet)
	asgardSize := vm.k.GetConfigInt64(ctx, constants.AsgardSize)
	redline := vm.k.GetConfigInt64(ctx, constants.BadValidatorRedline)
	minSlashPointsForBadValidator := vm.k.GetConfigInt64(ctx, constants.MinSlashPointsForBadValidator)

	// update list of ready actors
	if err := vm.markReadyActors(ctx); err != nil {
		return err
	}

	// clear leave scores
	if err := vm.clearLeaveScores(ctx); err != nil {
		return err
	}

	// Mark bad, old, low bond, and old version validators
	// mark someone to get churned out for bad behavior
	_, err := vm.markBadActor(ctx, minSlashPointsForBadValidator, redline)
	if err != nil {
		return err
	}

	// mark someone to get churned out for low bond
	if err := vm.markLowBondActor(ctx); err != nil {
		return err
	}

	// mark someone to get churned out for low version
	if err := vm.markLowVersionValidators(ctx); err != nil {
		return err
	}

	// mark someone to get churned out for age
	if err := vm.markOldActor(ctx); err != nil {
		return err
	}

	// mark someone(s) for not signing blocks
	if err := vm.markMissingActors(ctx); err != nil {
		return err
	}

	next, ok, err := vm.nextVaultNodeAccounts(ctx, int(desiredValidatorSet))
	if err != nil {
		return err
	}
	if ok {
		for _, nodeAccSet := range vm.splitNext(ctx, next, asgardSize) {
			if err := vm.networkMgr.TriggerKeygen(ctx, nodeAccSet); err != nil {
				return err
			}
		}
	}
	return nil
}

// splits given list of node accounts into separate list of nas, for separate
// asgard vaults
func (vm *ValidatorMgrVCUR) splitNext(ctx cosmos.Context, nas NodeAccounts, asgardSize int64) []NodeAccounts {
	if asgardSize <= 0 { // sanity check
		return nil
	}
	// calculate the number of asgard vaults we'll need to support the given
	// list of node accounts
	groupNum := int64(len(nas)) / asgardSize
	if int64(len(nas))%asgardSize > 0 {
		groupNum++
	}
	if groupNum <= 0 { // sanity check
		return nil
	}

	// we want to ensure that a single node operator (designated by bond
	// address) doesn't get too many tss shares for a single Asgard vault. So we
	// first break out our node accounts into two groups. First, duplicate bond
	// addresses (multi-node operators), and second non-duplicate (single node
	// operators). Then we sort the duplicate group by bond address, then by
	// bond size (large to small). Then we sort the non-duplicate group by bond size (large
	// to small). Then iterate over the first group into asgard vaults first,
	// then the second group. In the end multi-node operators are spread out
	// against as many asgard vaults as possible. This also makes it more
	// difficult for a malicious actor to acquire enough spots in a single
	// asgard to steal as enough are taken by "good actors" that they can't
	// acquire enough tss shares.

	// Check for duplicates
	bondAddrMap := make(map[string]int)
	for _, na := range nas {
		bondAddrMap[na.BondAddress.String()]++
	}
	var duplicateNas, nonDuplicateNas NodeAccounts
	for _, na := range nas {
		if bondAddrMap[na.BondAddress.String()] > 1 {
			duplicateNas = append(duplicateNas, na)
		} else {
			nonDuplicateNas = append(nonDuplicateNas, na)
		}
	}

	sort.SliceStable(duplicateNas, func(i, j int) bool {
		// Check if the bond address counts are the same
		if bondAddrMap[duplicateNas[i].BondAddress.String()] == bondAddrMap[duplicateNas[j].BondAddress.String()] {
			// Check if bond addresses are the same
			if duplicateNas[i].BondAddress.String() == duplicateNas[j].BondAddress.String() {
				// Sort by bond size
				return duplicateNas[i].Bond.GT(duplicateNas[j].Bond)
			}
			// Sort by bond address
			return duplicateNas[i].BondAddress.String() < duplicateNas[j].BondAddress.String()
		}
		// Sort by bond address count
		return bondAddrMap[duplicateNas[i].BondAddress.String()] > bondAddrMap[duplicateNas[j].BondAddress.String()]
	})

	// sort by bond size for non-duplicates
	sort.SliceStable(nonDuplicateNas, func(i, j int) bool {
		return nonDuplicateNas[i].Bond.LT(nonDuplicateNas[j].Bond)
	})

	groups := make([]NodeAccounts, groupNum)
	for i, na := range append(duplicateNas, nonDuplicateNas...) {
		groups[i%len(groups)] = append(groups[i%len(groups)], na)
	}

	// sanity checks
	for i, group := range groups {
		// ensure no group is more than the max
		if int64(len(group)) > asgardSize {
			ctx.Logger().Info("Skipping rotation due to an Asgard group is larger than the max size.")
			return nil
		}
		// ensure no group is less than the min
		if int64(len(group)) < 2 {
			ctx.Logger().Info("Skipping rotation due to an Asgard group is smaller than the min size.")
			return nil
		}
		// ensure a single group is significantly larger than another
		if i > 0 {
			diff := len(groups[i]) - len(groups[i-1])
			if diff < 0 {
				diff = -diff
			}
			if diff > 1 {
				ctx.Logger().Info("Skipping rotation due to an Asgard groups having dissimilar membership size.")
				return nil
			}
		}
	}

	return groups
}

// EndBlock when block commit
func (vm *ValidatorMgrVCUR) EndBlock(ctx cosmos.Context, mgr Manager) []abci.ValidatorUpdate {
	height := ctx.BlockHeight()
	activeNodes, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		ctx.Logger().Error("fail to get all active nodes", "error", err)
		return nil
	}

	// when ragnarok is in progress, just process ragnarok
	if vm.k.RagnarokInProgress(ctx) {
		// process ragnarok
		if err := vm.processRagnarok(ctx, mgr); err != nil {
			ctx.Logger().Error("fail to process ragnarok protocol", "error", err)
		}
		return nil
	}

	newNodes, removedNodes, err := vm.getChangedNodes(ctx, activeNodes)
	if err != nil {
		ctx.Logger().Error("fail to get node changes", "error", err)
		return nil
	}

	artificialRagnarokBlockHeight := vm.k.GetConfigInt64(ctx, constants.ArtificialRagnarokBlockHeight)
	if artificialRagnarokBlockHeight > 0 {
		ctx.Logger().Info("Artificial Ragnarok is planned", "height", artificialRagnarokBlockHeight)
	}
	minimumNodesForBFT := vm.k.GetConstants().GetInt64Value(constants.MinimumNodesForBFT)
	nodesAfterChange := len(activeNodes) + len(newNodes) - len(removedNodes)
	if (len(activeNodes) >= int(minimumNodesForBFT) && nodesAfterChange < int(minimumNodesForBFT)) ||
		(artificialRagnarokBlockHeight > 0 && ctx.BlockHeight() >= artificialRagnarokBlockHeight) {
		// THORNode don't have enough validators for BFT

		// Check we're not migrating funds
		retiring, err := vm.k.GetAsgardVaultsByStatus(ctx, RetiringVault)
		if err != nil {
			ctx.Logger().Error("fail to get retiring vaults", "error", err)
		}

		if len(retiring) == 0 { // wait until all funds are migrated before starting ragnarok
			if err := vm.processRagnarok(ctx, mgr); err != nil {
				ctx.Logger().Error("fail to process ragnarok protocol", "error", err)
			}
			return nil
		}
	}

	// If there's been a churn (the nodes have changed), continue; if there hasn't, end the function.
	if len(newNodes) == 0 && len(removedNodes) == 0 {
		return nil
	}

	// remove low bond node accounts
	if err := vm.k.RemoveLowBondValidatorAccounts(ctx); err != nil {
		ctx.Logger().Error("fail to remove low bond node accounts", "error", err)
	}

	// payout all active node accounts their rewards
	// This including nodes churning out, and takes place before changing the activity status below.
	if err := vm.distributeBondReward(ctx, mgr); err != nil {
		ctx.Logger().Error("fail to pay node bond rewards", "error", err)
	}

	validators := make([]abci.ValidatorUpdate, 0, len(newNodes)+len(removedNodes))
	for _, na := range newNodes {
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("UpdateNodeAccountStatus",
				cosmos.NewAttribute("Address", na.NodeAddress.String()),
				cosmos.NewAttribute("Former:", na.Status.String()),
				cosmos.NewAttribute("Current:", NodeActive.String())))
		na.UpdateStatus(NodeActive, height)
		na.LeaveScore = 0
		na.RequestedToLeave = false
		na.MissingBlocks = 0 // zero missing blocks that weren't signed (if any)

		vm.k.ResetNodeAccountSlashPoints(ctx, na.NodeAddress)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			ctx.Logger().Error("fail to save node account", "error", err)
		}
		pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, na.ValidatorConsPubKey)
		if err != nil {
			ctx.Logger().Error("fail to parse consensus public key", "key", na.ValidatorConsPubKey, "error", err)
			continue
		}
		validators = append(validators, abci.Ed25519ValidatorUpdate(pk.Bytes(), 100))
	}
	for _, na := range removedNodes {
		// retrieve the node from key value store again , as the node might get paid bond, thus the node properties has been changed
		nodeRemove, err := vm.k.GetNodeAccount(ctx, na.NodeAddress)
		if err != nil {
			ctx.Logger().Error("fail to get node account from key value store", "node address", na.NodeAddress)
			continue
		}

		status := NodeStandby
		if nodeRemove.ForcedToLeave {
			status = NodeDisabled
		}
		// if removed node requested to leave , unset it , so they can join back again
		if nodeRemove.RequestedToLeave {
			nodeRemove.RequestedToLeave = false
		}
		ctx.EventManager().EmitEvent(
			cosmos.NewEvent("UpdateNodeAccountStatus",
				cosmos.NewAttribute("Address", nodeRemove.NodeAddress.String()),
				cosmos.NewAttribute("Former:", nodeRemove.Status.String()),
				cosmos.NewAttribute("Current:", status.String())))
		nodeRemove.UpdateStatus(status, height)
		if err := vm.k.SetNodeAccount(ctx, nodeRemove); err != nil {
			ctx.Logger().Error("fail to save node account", "error", err)
		}

		pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, nodeRemove.ValidatorConsPubKey)
		if err != nil {
			ctx.Logger().Error("fail to parse consensus public key", "key", nodeRemove.ValidatorConsPubKey, "error", err)
			continue
		}
		caddr := sdk.ValAddress(pk.Address()).String()
		found := false
		for _, exist := range vm.existingValidators {
			if exist == caddr {
				validators = append(validators, abci.Ed25519ValidatorUpdate(pk.Bytes(), 0))
				found = true
				break
			}
		}
		if !found {
			ctx.Logger().Info("validator is not present, so can't be removed", "validator address", caddr)
		}

	}
	// reset all nodes in ready status back to standby status
	ready, err := vm.k.ListValidatorsByStatus(ctx, NodeReady)
	if err != nil {
		ctx.Logger().Error("fail to get list of ready node accounts", "error", err)
	}
	for _, na := range ready {
		na.UpdateStatus(NodeStandby, ctx.BlockHeight())
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			ctx.Logger().Error("fail to set node account", "error", err)
		}
	}

	// Now that the node statuses have been updated, update the stored MinJoinVersion.
	vm.k.SetMinJoinLast(ctx)

	// On each churn, purge OperationalMimir node votes
	// (without changing the set Mimir values themselves).
	// This is to stop any OperationalMimir key's threshold for change
	// from creeping up inconveniently high over time.
	// If a small number of nodes repeatedly uses this purge to go against the majority preference,
	// the EconomicMimir OperationalVotesMin could be set to a higher threshold.
	vm.k.PurgeOperationalNodeMimirs(ctx)

	return validators
}

// getChangedNodes to identify which node had been removed ,and which one had been added
// newNodes , removed nodes,err
func (vm *ValidatorMgrVCUR) getChangedNodes(ctx cosmos.Context, activeNodes NodeAccounts) (NodeAccounts, NodeAccounts, error) {
	var newActive NodeAccounts    // store the list of new active users
	var removedNodes NodeAccounts // nodes that had been removed

	activeVaults, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		ctx.Logger().Error("fail to get active asgards", "error", err)
		return newActive, removedNodes, fmt.Errorf("fail to get active asgards: %w", err)
	}
	if len(activeVaults) == 0 {
		return newActive, removedNodes, errors.New("no active vault")
	}
	var membership common.PubKeys
	for _, vault := range activeVaults {
		membership = append(membership, vault.GetMembership()...)
	}

	// find active node accounts that are no longer active
	for _, na := range activeNodes {
		found := false
		for _, vault := range activeVaults {
			if vault.Contains(na.PubKeySet.Secp256k1) {
				found = true
				break
			}
		}
		if na.ForcedToLeave {
			found = false
		}
		if !found && len(membership) > 0 {
			removedNodes = append(removedNodes, na)
		}
	}

	// find ready nodes that change to active
	for _, pk := range membership {
		na, err := vm.k.GetNodeAccountByPubKey(ctx, pk)
		if err != nil {
			ctx.Logger().Error("fail to get node account", "error", err)
			continue
		}
		// Disabled account can't go back , it should not be include in the newActive
		if na.Status != NodeActive && na.Status != NodeDisabled {
			newActive = append(newActive, na)
		}
	}

	return newActive, removedNodes, nil
}

// payNodeAccountBondAward pay
func (vm *ValidatorMgrVCUR) payNodeAccountBondAward(ctx cosmos.Context, lastChurnHeight int64, na NodeAccount, totalBondReward, totalEffectiveBond, bondHardCap cosmos.Uint, mgr Manager) error {
	if na.ActiveBlockHeight == 0 || na.Bond.IsZero() {
		return nil
	}

	network, err := vm.k.GetNetwork(ctx)
	if err != nil {
		return fmt.Errorf("fail to get network: %w", err)
	}

	slashPts, err := vm.k.GetNodeAccountSlashPoints(ctx, na.NodeAddress)
	if err != nil {
		return fmt.Errorf("fail to get node slash points: %w", err)
	}

	// Find number of blocks since the last churn (the last bond reward payout)
	totalActiveBlocks := ctx.BlockHeight() - lastChurnHeight

	// find number of blocks they were well behaved (ie active - slash points)
	earnedBlocks := totalActiveBlocks - slashPts
	if earnedBlocks < 0 {
		earnedBlocks = 0
	}

	naEffectiveBond := na.Bond
	if naEffectiveBond.GT(bondHardCap) {
		naEffectiveBond = bondHardCap
	}

	// reward = totalBondReward * (naEffectiveBond / totalEffectiveBond) * (unslashed blocks since last churn / blocks since last churn)
	reward := common.GetUncappedShare(naEffectiveBond, totalEffectiveBond, totalBondReward)
	reward = common.GetUncappedShare(cosmos.NewUint(uint64(earnedBlocks)), cosmos.NewUint(uint64(totalActiveBlocks)), reward)

	// Record the node operator's bond (in bond provider form) before the reward
	bp, err := vm.k.GetBondProviders(ctx, na.NodeAddress)
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to get bond providers(%s)", na.NodeAddress))
	}
	nodeOperatorAccAddr, err := na.BondAddress.AccAddress()
	if err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to parse bond address(%s)", na.BondAddress))
	}
	err = passiveBackfill(ctx, mgr, na, &bp)
	if err != nil {
		return err
	}
	// Ensure that previous rewards are already accounted for
	bp.Adjust(na.Bond)
	nodeOperatorProvider := bp.Get(nodeOperatorAccAddr)
	// Sanity check that the node operator is accounted for
	if nodeOperatorProvider.IsEmpty() {
		return ErrInternal(err, fmt.Sprintf("node operator address(%s) not listed in bond providers", na.BondAddress))
	}
	lastNodeOperatorProviderBond := nodeOperatorProvider.Bond

	// Add to their bond the amount rewarded
	na.Bond = na.Bond.Add(reward)

	// Minus the number of rune THORNode have awarded them
	network.BondRewardRune = common.SafeSub(network.BondRewardRune, reward)

	// Minus the number of units na has (do not include slash points)
	network.TotalBondUnits = common.SafeSub(
		network.TotalBondUnits,
		cosmos.NewUint(uint64(totalActiveBlocks)),
	)

	if err := vm.k.SetNetwork(ctx, network); err != nil {
		return fmt.Errorf("fail to save network data: %w", err)
	}

	// minus slash points used in this calculation
	vm.k.SetNodeAccountSlashPoints(ctx, na.NodeAddress, slashPts-totalActiveBlocks)

	// Distribute reward to bond providers and remove the NodeOperatorFee portion for node operator payout.
	// (This is the full fee from other bond providers' rewards, plus an equivalent proportion of the node operator's rewards.)
	bp.Adjust(na.Bond)
	nodeOperatorProvider = bp.Get(nodeOperatorAccAddr)
	nodeOperatorFees := common.GetSafeShare(bp.NodeOperatorFee, cosmos.NewUint(10000), reward)
	// Sanity check:  Fees to pay out should never exceed the increase of the node operator's bond.
	nodeOperatorBondIncrease := common.SafeSub(nodeOperatorProvider.Bond, lastNodeOperatorProviderBond)
	if nodeOperatorFees.GT(nodeOperatorBondIncrease) {
		nodeOperatorFees = nodeOperatorBondIncrease
	}
	if !nodeOperatorFees.IsZero() {
		na.Bond = common.SafeSub(na.Bond, nodeOperatorFees)
		bp.Unbond(nodeOperatorFees, nodeOperatorAccAddr)
	}

	// Set node account and bond providers, then emit BondReward event (for the full pre-payout reward)
	if err := vm.k.SetNodeAccount(ctx, na); err != nil {
		return fmt.Errorf("fail to save node account: %w", err)
	}
	if err := mgr.Keeper().SetBondProviders(ctx, bp); err != nil {
		return ErrInternal(err, fmt.Sprintf("fail to save bond providers(%s)", bp.NodeAddress.String()))
	}
	bondRewardEvent := NewEventBond(reward, BondReward, common.Tx{}, &na, nil)
	if err := mgr.EventMgr().EmitEvent(ctx, bondRewardEvent); err != nil {
		ctx.Logger().Error("fail to emit bond event", "error", err)
	}

	// Transfer node operator fees
	if !nodeOperatorFees.IsZero() {
		coin := common.NewCoin(common.SwitchNative, nodeOperatorFees)
		sdkErr := vm.k.SendFromModuleToAccount(ctx, BondName, nodeOperatorAccAddr, common.NewCoins(coin))
		if sdkErr != nil {
			return errors.New(sdkErr.Error())
		}

		// TODO: Remove this NewAddress call entirely in the next version of this manager.
		_, err := common.NewAddress(na.NodeAddress.String())
		if err != nil {
			return fmt.Errorf("fail to parse node address: %w", err)
		}
		// emit BondReturned event
		bondReturnedEvent := NewEventBond(nodeOperatorFees, BondReturned, common.Tx{}, &na, nodeOperatorAccAddr)
		if err := mgr.EventMgr().EmitEvent(ctx, bondReturnedEvent); err != nil {
			ctx.Logger().Error("fail to emit bond event", "error", err)
		}
	}

	return nil
}

// determines when/if to run each part of the ragnarok process
func (vm *ValidatorMgrVCUR) processRagnarok(ctx cosmos.Context, mgr Manager) error {
	// execute Ragnarok protocol, no going back
	// THORNode have to request the fund back now, because once it get to the rotate block height ,
	// THORNode won't have validators anymore
	ragnarokHeight, err := vm.k.GetRagnarokBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok height: %w", err)
	}

	if ragnarokHeight == 0 {
		ragnarokHeight = ctx.BlockHeight()
		vm.k.SetRagnarokBlockHeight(ctx, ragnarokHeight)
		if err := vm.distributeBondReward(ctx, mgr); err != nil {
			return fmt.Errorf("when ragnarok triggered, fail to give all active node bond reward %w", err)
		}
		return nil
	}

	nth, err := vm.k.GetRagnarokNth(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok nth: %w", err)
	}

	position, err := vm.k.GetRagnarokWithdrawPosition(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok position: %w", err)
	}
	if !position.IsEmpty() {
		if err := vm.ragnarokPools(ctx, nth, mgr); err != nil {
			ctx.Logger().Error("fail to ragnarok pools", "error", err)
		}
		return nil
	}

	// check if we have any pending ragnarok transactions
	pending, err := vm.k.GetRagnarokPending(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok pending: %w", err)
	}
	if pending > 0 {
		txOutQueue, err := vm.getPendingTxOut(ctx)
		if err != nil {
			ctx.Logger().Error("fail to get pending tx out item", "error", err)
			return nil
		}
		if txOutQueue > 0 {
			ctx.Logger().Info("awaiting previous ragnarok transaction to clear before continuing", "nth", nth, "count", pending)
			return nil
		}
	}

	nth++ // increment by 1
	ctx.Logger().Info("starting next ragnarok iteration", "iteration", nth)
	err = vm.ragnarokProtocolStage2(ctx, nth, mgr)
	if err != nil {
		ctx.Logger().Error("fail to execute ragnarok protocol step 2", "error", err)
		return err
	}
	vm.k.SetRagnarokNth(ctx, nth)

	return nil
}

func (vm *ValidatorMgrVCUR) getPendingTxOut(ctx cosmos.Context) (int64, error) {
	signingTransactionPeriod := vm.k.GetConstants().GetInt64Value(constants.SigningTransactionPeriod)
	startHeight := ctx.BlockHeight() - signingTransactionPeriod
	count := int64(0)
	for height := startHeight; height <= ctx.BlockHeight(); height++ {
		txs, err := vm.k.GetTxOut(ctx, height)
		if err != nil {
			ctx.Logger().Error("fail to get tx out array from key value store", "error", err)
			return 0, fmt.Errorf("fail to get tx out array from key value store: %w", err)
		}
		for _, tx := range txs.TxArray {
			if tx.OutHash.IsEmpty() {
				count++
			}
		}
	}
	return count, nil
}

func (vm *ValidatorMgrVCUR) ragnarokProtocolStage2(ctx cosmos.Context, nth int64, mgr Manager) error {
	// Ragnarok Protocol
	// If THORNode can no longer be BFT, do a graceful shutdown of the entire network.
	// 1) THORNode will refund the validator's bond
	// 2) return all fund to liquidity providers

	// refund bonders
	if err := vm.ragnarokBond(ctx, nth, mgr); err != nil {
		ctx.Logger().Error("fail to ragnarok bond", "error", err)
	}

	// refund liquidity providers. This is last to ensure there is likely gas for the
	// returning bond and reserve
	if err := vm.ragnarokPools(ctx, nth, mgr); err != nil {
		ctx.Logger().Error("fail to ragnarok pools", "error", err)
	}

	return nil
}

func (vm *ValidatorMgrVCUR) distributeBondReward(ctx cosmos.Context, mgr Manager) error {
	var resultErr error
	active, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return fmt.Errorf("fail to get all active node account: %w", err)
	}

	// Note that unlike estimated CurrentAward distribution in querier.go ,
	// this estimate treats lastChurnHeight as the active_block_height of the youngest active node,
	// rather than the block_height of the first (oldest) Asgard vault.
	// As an example, note from the below URLs that these 5293733 and 5293728 respectively in block 5336942.
	// https://thornode.ninerealms.com/thorchain/nodes?height=5336942
	// (Nodes .cxmy and .uy3a .)
	// https://thornode.ninerealms.com/thorchain/vaults/asgard?height=5336942
	lastChurnHeight := int64(0)
	for _, node := range active {
		if node.ActiveBlockHeight > lastChurnHeight {
			lastChurnHeight = node.ActiveBlockHeight
		}
	}

	totalEffectiveBond, bondHardCap := getTotalEffectiveBond(active)

	network, err := vm.k.GetNetwork(ctx)
	if err != nil {
		return fmt.Errorf("fail to get network: %w", err)
	}

	for _, item := range active {
		if err := vm.payNodeAccountBondAward(ctx, lastChurnHeight, item, network.BondRewardRune, totalEffectiveBond, bondHardCap, mgr); err != nil {
			resultErr = err
			ctx.Logger().Error("fail to pay node account bond award", "node address", item.NodeAddress.String(), "error", err)
		}
	}
	return resultErr
}

func (vm *ValidatorMgrVCUR) ragnarokBond(ctx cosmos.Context, nth int64, mgr Manager) error {
	// bond should be returned on the back 10, not the first 10
	nth -= 10
	if nth < 1 {
		return nil
	}

	nas, err := vm.k.ListValidatorsWithBond(ctx)
	if err != nil {
		ctx.Logger().Error("can't get nodes", "error", err)
		return err
	}
	// nth * 10 == the amount of the bond we want to send
	for i, na := range nas {
		if na.Bond.IsZero() {
			continue
		}

		if nth >= 9 { // cap at 10
			nth = 10
		}
		amt := na.Bond.MulUint64(uint64(nth)).QuoUint64(10)

		// refund bond
		txOutItem := TxOutItem{
			Chain:      common.SwitchNative.Chain,
			ToAddress:  na.BondAddress,
			InHash:     common.BlankTxID,
			Coin:       common.NewCoin(common.SwitchNative, amt),
			Memo:       NewRagnarokMemo(ctx.BlockHeight()).String(),
			ModuleName: BondName,
		}
		ok, err := vm.txOutStore.TryAddTxOutItem(ctx, mgr, txOutItem, cosmos.ZeroUint())
		if err != nil {
			if !errors.Is(err, ErrNotEnoughToPayFee) {
				return err
			}
			ok = true
		}
		if !ok {
			continue
		}

		// add a pending rangarok transaction
		pending, err := vm.k.GetRagnarokPending(ctx)
		if err != nil {
			return fmt.Errorf("fail to get ragnarok pending: %w", err)
		}
		vm.k.SetRagnarokPending(ctx, pending+1)

		na.Bond = common.SafeSub(na.Bond, amt)
		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}

		bondEvent := NewEventBond(amt, BondCost, common.Tx{}, &nas[i], nil)
		if err := mgr.EventMgr().EmitEvent(ctx, bondEvent); err != nil {
			return fmt.Errorf("fail to emit bond event: %w", err)
		}
	}

	return nil
}

func (vm *ValidatorMgrVCUR) ragnarokPools(ctx cosmos.Context, nth int64, mgr Manager) error {
	nas, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return fmt.Errorf("fail to get active nodes: %w", err)
	}
	if len(nas) == 0 {
		return fmt.Errorf("can't find any active nodes")
	}
	na := nas[0]

	position, err := vm.k.GetRagnarokWithdrawPosition(ctx)
	if err != nil {
		return fmt.Errorf("fail to get ragnarok position: %w", err)
	}
	basisPoints := MaxWithdrawBasisPoints
	// go through all the pools
	pools, err := vm.k.GetPools(ctx)
	if err != nil {
		return fmt.Errorf("fail to get pools: %w", err)
	}
	// set all pools to staged status
	for _, pool := range pools {
		if pool.Status != PoolStaged {
			poolEvent := NewEventPool(pool.Asset, PoolStaged)
			if err := vm.eventMgr.EmitEvent(ctx, poolEvent); err != nil {
				ctx.Logger().Error("fail to emit pool event", "error", err)
			}

			pool.Status = PoolStaged
			if err := vm.k.SetPool(ctx, pool); err != nil {
				return fmt.Errorf("fail to set pool %s to Stage status: %w", pool.Asset, err)
			}
		}
	}

	// the following line is pointless, granted. But in this case, removing it
	// would cause a consensus failure
	_ = vm.k.GetLowestActiveVersion(ctx)

	nextPool := false
	maxWithdrawsPerBlock := 20
	count := 0

	for i := len(pools) - 1; i >= 0; i-- { // iterate backwards
		pool := pools[i]

		if nextPool { // we've iterated to the next pool after our position pool
			position.Pool = pool.Asset
		}

		if !position.Pool.IsEmpty() && !pool.Asset.Equals(position.Pool) {
			continue
		}

		nextPool = true
		position.Pool = pool.Asset

		// withdraw gas asset pool on the back 10 nths
		if nth <= 10 && pool.Asset.IsGasAsset() {
			continue
		}

		j := int64(-1)
		iterator := vm.k.GetLiquidityProviderIterator(ctx, pool.Asset)
		for ; iterator.Valid(); iterator.Next() {
			j++
			if j == position.Number {
				position.Number++
				var lp LiquidityProvider
				if err := vm.k.Cdc().Unmarshal(iterator.Value(), &lp); err != nil {
					ctx.Logger().Error("fail to unmarshal liquidity provider", "error", err)
					continue
				}

				if lp.Units.IsZero() {
					continue
				}
				var withdrawAddr common.Address
				withdrawAsset := common.EmptyAsset
				if !lp.RuneAddress.IsEmpty() {
					withdrawAddr = lp.RuneAddress
					// if liquidity provider only add RUNE , then asset address will be empty
					if lp.AssetAddress.IsEmpty() {
						withdrawAsset = common.SwitchNative
					}
				} else {
					// if liquidity provider only add Asset, then RUNE Address will be empty
					withdrawAddr = lp.AssetAddress
					withdrawAsset = lp.Asset
				}
				withdrawMsg := NewMsgWithdrawLiquidity(
					common.GetRagnarokTx(pool.Asset.Chain, withdrawAddr, withdrawAddr),
					withdrawAddr,
					cosmos.NewUint(uint64(basisPoints)),
					pool.Asset,
					withdrawAsset,
					na.NodeAddress,
				)

				handler := NewInternalHandler(mgr)
				_, err = handler(ctx, withdrawMsg)
				if err != nil {
					ctx.Logger().Error("fail to withdraw", "liquidity provider", lp.RuneAddress, "error", err)
				} else if !withdrawAsset.Equals(common.SwitchNative) {
					// when withdraw asset is only RUNE , then it should process more , because RUNE asset doesn't leave THORChain
					count++
					pending, err := vm.k.GetRagnarokPending(ctx)
					if err != nil {
						return fmt.Errorf("fail to get ragnarok pending: %w", err)
					}
					vm.k.SetRagnarokPending(ctx, pending+1)
					if count >= maxWithdrawsPerBlock {
						break
					}
				}
			}
		}
		if err := iterator.Close(); err != nil {
			ctx.Logger().Error("fail to close iterator", "error", err)
		}
		if count >= maxWithdrawsPerBlock {
			break
		}
		position.Number = 0
	}

	if count < maxWithdrawsPerBlock { // we've completed all pools/liquidity providers, reset the position
		position = RagnarokWithdrawPosition{}
	}
	vm.k.SetRagnarokWithdrawPosition(ctx, position)

	return nil
}

// setupValidatorNodes it is one off it only get called when genesis
func (vm *ValidatorMgrVCUR) setupValidatorNodes(ctx cosmos.Context, height int64) error {
	if height != genesisBlockHeight {
		ctx.Logger().Info("only need to setup validator node when start up", "height", height)
		return nil
	}

	iter := vm.k.GetNodeAccountIterator(ctx)
	defer iter.Close()
	readyNodes := NodeAccounts{}
	activeCandidateNodes := NodeAccounts{}
	for ; iter.Valid(); iter.Next() {
		var na NodeAccount
		if err := vm.k.Cdc().Unmarshal(iter.Value(), &na); err != nil {
			return fmt.Errorf("fail to unmarshal node account, %w", err)
		}
		// when THORNode first start , THORNode only care about these two status
		switch na.Status {
		case NodeReady:
			readyNodes = append(readyNodes, na)
		case NodeActive:
			activeCandidateNodes = append(activeCandidateNodes, na)
		}
	}
	totalActiveValidators := len(activeCandidateNodes)
	totalNominatedValidators := len(readyNodes)
	if totalActiveValidators == 0 && totalNominatedValidators == 0 {
		return errors.New("no validators available")
	}

	sort.Sort(activeCandidateNodes)
	sort.Sort(readyNodes)
	activeCandidateNodes = append(activeCandidateNodes, readyNodes...)
	desiredValidatorSet := vm.k.GetConfigInt64(ctx, constants.DesiredValidatorSet)
	for idx, item := range activeCandidateNodes {
		if int64(idx) < desiredValidatorSet {
			item.UpdateStatus(NodeActive, ctx.BlockHeight())
		} else {
			item.UpdateStatus(NodeStandby, ctx.BlockHeight())
		}
		if err := vm.k.SetNodeAccount(ctx, item); err != nil {
			return fmt.Errorf("fail to save node account: %w", err)
		}
	}
	return nil
}

func (vm *ValidatorMgrVCUR) getLastChurnHeight(ctx cosmos.Context) int64 {
	vaults, err := vm.k.GetAsgardVaultsByStatus(ctx, ActiveVault)
	if err != nil {
		ctx.Logger().Error("Failed to get Asgard vaults", "error", err)
		return ctx.BlockHeight()
	}
	// calculate last churn block height
	var lastChurnHeight int64 // the last block height we had a successful churn
	for _, vault := range vaults {
		if vault.StatusSince > lastChurnHeight {
			lastChurnHeight = vault.StatusSince
		}
	}
	return lastChurnHeight
}

func (vm *ValidatorMgrVCUR) getScore(ctx cosmos.Context, slashPts, lastChurnHeight int64) cosmos.Uint {
	// get to the 8th decimal point, but keep numbers integers for safer math
	score := cosmos.NewUint(uint64((ctx.BlockHeight() - lastChurnHeight) * common.One))
	if slashPts == 0 {
		return score
	}
	return score.QuoUint64(uint64(slashPts))
}

// Iterate over active node accounts, finding bad actors with high slash points
func (vm *ValidatorMgrVCUR) findBadActors(ctx cosmos.Context, minSlashPointsForBadValidator, badValidatorRedline int64) (NodeAccounts, error) {
	badActors := make(NodeAccounts, 0)
	nas, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return badActors, err
	}

	if len(nas) == 0 {
		return nil, nil
	}

	// NOTE: Our score gives a numerical representation of the behavior our a
	// node account. The lower the score, the worse behavior. The score is
	// determined by relative to how many slash points they have over how long
	// they have been an active node account.
	type badTracker struct {
		Score       cosmos.Uint
		NodeAccount NodeAccount
	}
	tracker := make([]badTracker, 0, len(nas))
	totalScore := cosmos.ZeroUint()

	// Find bad actor relative to age / slashpoints
	lastChurnHeight := vm.getLastChurnHeight(ctx)
	for _, na := range nas {
		slashPts, err := vm.k.GetNodeAccountSlashPoints(ctx, na.NodeAddress)
		if err != nil {
			ctx.Logger().Error("fail to get node slash points", "error", err)
		}

		if slashPts <= minSlashPointsForBadValidator {
			continue
		}

		score := vm.getScore(ctx, slashPts, lastChurnHeight)
		totalScore = totalScore.Add(score)

		tracker = append(tracker, badTracker{
			Score:       score,
			NodeAccount: na,
		})
	}

	if len(tracker) == 0 {
		// no offenders, exit nicely
		return nil, nil
	}

	sort.SliceStable(tracker, func(i, j int) bool {
		return tracker[i].Score.LT(tracker[j].Score)
	})

	// score lower is worse
	avgScore := totalScore.QuoUint64(uint64(len(nas)))

	// NOTE: our redline is a hard line in the sand to determine if a node
	// account is sufficiently bad that it should just be removed now. This
	// ensures that if we have multiple "really bad" node accounts, they all
	// can get removed in the same churn. It is important to note we shouldn't
	// be able to churn out more than 1/3rd of our node accounts in a single
	// churn, as that could threaten the security of the funds. This logic to
	// protect against this is not inside this function.
	redline := avgScore.QuoUint64(uint64(badValidatorRedline))

	// find any node accounts that have crossed the red line
	for _, track := range tracker {
		if redline.GTE(track.Score) {
			badActors = append(badActors, track.NodeAccount)
		}
	}

	// if no one crossed the redline, lets just grab the worse offender
	if len(badActors) == 0 {
		badActors = NodeAccounts{tracker[0].NodeAccount}
	}

	return badActors, nil
}

// Iterate over active node accounts, finding the one that hasn't been signing blocks
func (vm *ValidatorMgrVCUR) markMissingActors(ctx cosmos.Context) error {
	maxMissingBlocks := vm.k.GetConfigInt64(ctx, constants.MissingBlockChurnOut)
	maxChurnOut := vm.k.GetConfigInt64(ctx, constants.MaxMissingBlockChurnOut)
	if maxMissingBlocks == 0 || maxChurnOut == 0 {
		return nil // skip this mark
	}

	nas, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return err
	}

	counter := int64(0)
	for _, n := range nas {
		// Only mark an old actor not already marked for churn-out.
		if n.LeaveScore > 0 {
			continue
		}

		if maxMissingBlocks < int64(n.MissingBlocks) {
			if err := vm.markActor(ctx, n, "for not signing blocks"); err != nil {
				return err
			}
			counter += 1
			if counter >= maxChurnOut {
				break
			}
		}
	}

	return nil
}

// Iterate over active node accounts, finding the one that has been active longest
func (vm *ValidatorMgrVCUR) findOldActor(ctx cosmos.Context) (NodeAccount, error) {
	na := NodeAccount{}
	nas, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return na, err
	}

	na.StatusSince = ctx.BlockHeight() // set the start status age to "now"
	for _, n := range nas {
		// Only mark an old actor not already marked for churn-out.
		if n.LeaveScore > 0 {
			continue
		}
		if n.StatusSince < na.StatusSince {
			na = n
		}
	}

	return na, nil
}

// Iterate over active node accounts, finding the one that has the lowest bond
func (vm *ValidatorMgrVCUR) findLowBondActor(ctx cosmos.Context) (NodeAccount, error) {
	na := NodeAccount{}
	nas, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return na, err
	}

	if len(nas) > 0 {
		bond := nas[0].Bond
		na = nas[0]
		for _, n := range nas {
			// Only mark a low bond actor not already marked for churn-out.
			if n.LeaveScore > 0 {
				continue
			}
			if n.Bond.LT(bond) {
				bond = n.Bond
				na = n
			}
		}
	}

	return na, nil
}

// Mark an actor to be churned out
func (vm *ValidatorMgrVCUR) markActor(ctx cosmos.Context, na NodeAccount, reason string) error {
	if !na.IsEmpty() && na.LeaveScore == 0 {
		ctx.Logger().Info("marked Validator to be churned out", "node address", na.NodeAddress, "reason", reason)
		slashPts, err := vm.k.GetNodeAccountSlashPoints(ctx, na.NodeAddress)
		if err != nil {
			return fmt.Errorf("fail to get node account(%s) slash points: %w", na.NodeAddress, err)
		}
		na.LeaveScore = vm.getScore(ctx, slashPts, vm.getLastChurnHeight(ctx)).Uint64()
		return vm.k.SetNodeAccount(ctx, na)
	}
	return nil
}

// Mark an old actor to be churned out
func (vm *ValidatorMgrVCUR) markOldActor(ctx cosmos.Context) error {
	na, err := vm.findOldActor(ctx)
	if err != nil {
		return err
	}
	if err := vm.markActor(ctx, na, "for age"); err != nil {
		return err
	}
	return nil
}

// Mark an low bond actor to be churned out
func (vm *ValidatorMgrVCUR) markLowBondActor(ctx cosmos.Context) error {
	na, err := vm.findLowBondActor(ctx)
	if err != nil {
		return err
	}
	if err := vm.markActor(ctx, na, "for low bond"); err != nil {
		return err
	}
	return nil
}

// Mark a bad actor to be churned out
func (vm *ValidatorMgrVCUR) markBadActor(ctx cosmos.Context, minSlashPointsForBadValidator, redline int64) (int64, error) {
	nas, err := vm.findBadActors(ctx, minSlashPointsForBadValidator, redline)
	if err != nil {
		return 0, err
	}
	for _, na := range nas {
		if err := vm.markActor(ctx, na, "for bad behavior"); err != nil {
			return 0, err
		}
	}
	return int64(len(nas)), nil
}

// Mark up to `MaxNodeToChurnOutForLowVersion` nodes as low version
// This will slate them to churn out. `MaxNodeToChurnOutForLowVersion`
// is a Mimir setting that defaults in constants to 1
func (vm *ValidatorMgrVCUR) markLowVersionValidators(ctx cosmos.Context) error {
	// Only mark low version validators later than ChurnOutForLowVersionBlocks since the MinJoinVersion last changed.
	_, minJoinlastHeight := vm.k.GetMinJoinLast(ctx)
	churnOutForLowVersionBlocks := vm.k.GetConfigInt64(ctx, constants.ChurnOutForLowVersionBlocks)
	if ctx.BlockHeight() < minJoinlastHeight+churnOutForLowVersionBlocks {
		return nil
	}

	// Get max number of nodes to mark as low version
	maxNodes := vm.k.GetConfigInt64(ctx, constants.MaxNodeToChurnOutForLowVersion)

	nodeAccs, err := vm.findLowVersionValidators(ctx, maxNodes)
	if err != nil {
		return err
	}
	if len(nodeAccs) > 0 {
		for _, na := range nodeAccs {
			if err := vm.markActor(ctx, na, "for version lower than minimum join version"); err != nil {
				return err
			}
		}
	}
	return nil
}

// Finds up to `maxNodesToFind` active validators with version lower than the most "popular" version
func (vm *ValidatorMgrVCUR) findLowVersionValidators(ctx cosmos.Context, maxNodesToFind int64) (NodeAccounts, error) {
	minimumVersion, _ := vm.k.GetMinJoinLast(ctx)
	activeNodes, err := vm.k.ListValidatorsByStatus(ctx, NodeActive)
	if err != nil {
		return NodeAccounts{}, err
	}
	nodeAccs := NodeAccounts{}
	for _, na := range activeNodes {
		// Only mark low version actors not already marked for churn-out.
		if na.LeaveScore > 0 {
			continue
		}
		if na.GetVersion().LT(minimumVersion) {
			nodeAccs = append(nodeAccs, na)
		}
		if len(nodeAccs) == int(maxNodesToFind) {
			return nodeAccs, nil
		}
	}
	return nodeAccs, nil
}

// clearLeaveScores - clears all leaves scores of active validators except for
// ones that requested to leave
func (vm *ValidatorMgrVCUR) clearLeaveScores(ctx cosmos.Context) error {
	active, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return err
	}

	for _, na := range active {
		if na.RequestedToLeave || na.ForcedToLeave {
			continue
		}
		na.LeaveScore = 0

		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}
	}

	return nil
}

// find any actor that are ready to become "ready" status
func (vm *ValidatorMgrVCUR) markReadyActors(ctx cosmos.Context) error {
	standby, err := vm.k.ListValidatorsByStatus(ctx, NodeStandby)
	if err != nil {
		return err
	}
	ready, err := vm.k.ListValidatorsByStatus(ctx, NodeReady)
	if err != nil {
		return err
	}

	// check all ready and standby nodes are in "ready" state (upgrade/downgrade as needed)
	for _, na := range append(standby, ready...) {
		status, _ := vm.NodeAccountPreflightCheck(ctx, na, vm.k.GetConstants())
		na.UpdateStatus(status, ctx.BlockHeight())

		if err := vm.k.SetNodeAccount(ctx, na); err != nil {
			return err
		}
	}

	return nil
}

// NodeAccountPreflightCheck preflight check to find out what the node account's next status will be
func (vm *ValidatorMgrVCUR) NodeAccountPreflightCheck(ctx cosmos.Context, na NodeAccount, _ constants.ConstantValues) (NodeStatus, error) {
	// ensure banned nodes can't get churned in again
	if na.ForcedToLeave {
		return NodeDisabled, fmt.Errorf("node account has been banned")
	}

	// Check if they've requested to leave
	if na.RequestedToLeave {
		return NodeStandby, fmt.Errorf("node account has requested to leave")
	}

	if na.Maintenance {
		return NodeStandby, fmt.Errorf("node account is in maintenance mode")
	}

	// Check that the node account has an IP address
	if net.ParseIP(na.IPAddress) == nil {
		return NodeStandby, fmt.Errorf("node account has invalid registered IP address")
	}

	// Check that the node account has an pubkey set
	if na.PubKeySet.IsEmpty() {
		return NodeWhiteListed, fmt.Errorf("node account has not registered their pubkey set")
	}

	// ensure we have enough rune
	minBond := vm.k.GetConfigInt64(ctx, constants.MinimumBondInRune)
	if na.Bond.LT(cosmos.NewUint(uint64(minBond))) {
		return NodeStandby, fmt.Errorf("node account does not have minimum bond requirement: %d/%d", na.Bond.Uint64(), minBond)
	}

	minVersion, _ := vm.k.GetMinJoinLast(ctx)
	// Check version number is still supported
	if na.GetVersion().LT(minVersion) {
		return NodeStandby, fmt.Errorf("node account does not meet min version requirement: %s vs %s", na.Version, minVersion)
	}

	jail, err := vm.k.GetNodeAccountJail(ctx, na.NodeAddress)
	if err != nil {
		ctx.Logger().Error("fail to get node account jail", "error", err)
		return NodeStandby, fmt.Errorf("cannot fetch jail status: %w", err)
	}
	if jail.IsJailed(ctx) {
		return NodeStandby, fmt.Errorf("node account is jailed until block %d: %s", jail.ReleaseHeight, jail.Reason)
	}

	if vm.k.RagnarokInProgress(ctx) {
		return NodeStandby, fmt.Errorf("ragnarok is currently in progress: no churning")
	}

	return NodeReady, nil
}

// Returns a list of nodes to include in the next pool
func (vm *ValidatorMgrVCUR) nextVaultNodeAccounts(ctx cosmos.Context, targetCount int) (NodeAccounts, bool, error) {
	rotation := false // track if are making any changes to the current active node accounts

	ready, err := vm.k.ListValidatorsByStatus(ctx, NodeReady)
	if err != nil {
		return nil, false, err
	}

	// sort by bond size, descending
	sort.SliceStable(ready, func(i, j int) bool {
		return ready[i].Bond.GT(ready[j].Bond)
	})

	active, err := vm.k.ListActiveValidators(ctx)
	if err != nil {
		return nil, false, err
	}

	// find out all the nodes that had been marked to leave , and update their score again , because even after a node has been marked
	// to be churn out , they can continue to accumulate slash points, in the scenario that an active node go offline , and consistently fail
	// keygen / keysign for a while , we would like to churn it out first
	lastChurnHeight := vm.getLastChurnHeight(ctx)
	for idx, item := range active {

		if item.LeaveScore == 0 {
			continue
		}
		slashPts, err := vm.k.GetNodeAccountSlashPoints(ctx, item.NodeAddress)
		if err != nil {
			ctx.Logger().Error("fail to get node account slash points", "error", err, "node address", item.NodeAddress.String())
			continue
		}
		newScore := vm.getScore(ctx, slashPts, lastChurnHeight)
		if !newScore.IsZero() {
			active[idx].LeaveScore = newScore.Uint64()
		}
	}

	// sort by LeaveScore ascending
	// giving preferential treatment to people who are forced to leave
	//  and then requested to leave
	sort.SliceStable(active, func(i, j int) bool {
		if active[i].ForcedToLeave != active[j].ForcedToLeave {
			return active[i].ForcedToLeave
		}
		if active[i].RequestedToLeave != active[j].RequestedToLeave {
			return active[i].RequestedToLeave
		}
		// sort by LeaveHeight ascending , but exclude LeaveHeight == 0 , because that's the default value
		if active[i].LeaveScore == 0 && active[j].LeaveScore > 0 {
			return false
		}
		if active[i].LeaveScore > 0 && active[j].LeaveScore == 0 {
			return true
		}
		return active[i].LeaveScore < active[j].LeaveScore
	})

	toRemove := findCountToRemove(active)
	if toRemove > 0 {
		rotation = true
		active = active[toRemove:]
	}
	newNode, err := vm.k.GetMimir(ctx, constants.NumberOfNewNodesPerChurn.String())
	if err != nil || newNode <= 0 {
		newNode = 1
	}

	// add ready nodes to become active
	limit := toRemove + int(newNode) // Max limit of ready nodes to churn in
	minimumNodesForBFT := vm.k.GetConstants().GetInt64Value(constants.MinimumNodesForBFT)
	if len(active)+limit < int(minimumNodesForBFT) {
		limit = int(minimumNodesForBFT) - len(active)
	}

	for i := 1; targetCount > len(active); i++ {
		if len(ready) >= i {
			rotation = true
			active = append(active, ready[i-1])
		}
		if i == limit { // limit adding ready accounts
			break
		}
	}

	return active, rotation, nil
}
