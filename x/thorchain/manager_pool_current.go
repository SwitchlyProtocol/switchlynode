package thorchain

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

type PoolMgrVCUR struct{}

func newPoolMgrVCUR() *PoolMgrVCUR {
	return &PoolMgrVCUR{}
}

// EndBlock cycle pools if required and if ragnarok is not in progress
func (pm *PoolMgrVCUR) EndBlock(ctx cosmos.Context, mgr Manager) error {
	poolCycle, err := mgr.Keeper().GetMimir(ctx, constants.PoolCycle.String())
	if poolCycle < 0 || err != nil {
		poolCycle = mgr.GetConstants().GetInt64Value(constants.PoolCycle)
	}
	// Enable a pool every poolCycle
	if ctx.BlockHeight()%poolCycle == 0 && !mgr.Keeper().RagnarokInProgress(ctx) {
		maxAvailablePools, err := mgr.Keeper().GetMimir(ctx, constants.MaxAvailablePools.String())
		if maxAvailablePools < 0 || err != nil {
			maxAvailablePools = mgr.GetConstants().GetInt64Value(constants.MaxAvailablePools)
		}
		minRunePoolDepth, err := mgr.Keeper().GetMimir(ctx, constants.MinRunePoolDepth.String())
		if minRunePoolDepth < 0 || err != nil {
			minRunePoolDepth = mgr.GetConstants().GetInt64Value(constants.MinRunePoolDepth)
		}
		stagedPoolCost, err := mgr.Keeper().GetMimir(ctx, constants.StagedPoolCost.String())
		if stagedPoolCost < 0 || err != nil {
			stagedPoolCost = mgr.GetConstants().GetInt64Value(constants.StagedPoolCost)
		}
		if err := pm.cyclePools(ctx, maxAvailablePools, minRunePoolDepth, stagedPoolCost, mgr); err != nil {
			ctx.Logger().Error("Unable to enable a pool", "error", err)
		}
	}
	pm.cleanupPendingLiquidity(ctx, mgr)
	if err = pm.checkSaversUtilization(ctx, mgr); err != nil {
		ctx.Logger().Error("Unable to force withdraw saver position", "error", err)
	}

	return nil
}

// cyclePools update the set of Available and Staged pools
// Available non-gas pools not meeting the fee quota since last cycle, or not
// meeting availability requirements, are demoted to Staged.
// Staged pools are charged a fee and those with with zero rune depth and
// non-zero asset depth are removed along with their liquidity providers, and
// remaining assets are abandoned.
// The valid Staged pool with the highest rune depth is promoted to Available.
// If there are more than the maximum available pools, the Available pool with
// with the lowest rune depth is demoted to Staged
func (pm *PoolMgrVCUR) cyclePools(ctx cosmos.Context, maxAvailablePools, minRunePoolDepth, stagedPoolCost int64, mgr Manager) error {
	var availblePoolCount int64
	onDeck := NewPool()        // currently staged pool that could get promoted
	choppingBlock := NewPool() // currently available pool that is on the chopping block to being demoted
	minRuneDepth := cosmos.NewUint(uint64(minRunePoolDepth))
	minPoolLiquidityFee := mgr.Keeper().GetConfigInt64(ctx, constants.MinimumPoolLiquidityFee)
	// quick func to check the validity of a pool
	validPool := func(pool Pool) bool {
		if pool.BalanceAsset.IsZero() || pool.BalanceRune.IsZero() || pool.BalanceRune.LT(minRuneDepth) {
			return false
		}
		return true
	}

	// quick func to save a pool status and emit event
	setPool := func(pool Pool) error {
		poolEvt := NewEventPool(pool.Asset, pool.Status)
		if err := mgr.EventMgr().EmitEvent(ctx, poolEvt); err != nil {
			return fmt.Errorf("fail to emit pool event: %w", err)
		}

		switch pool.Status {
		case PoolAvailable:
			ctx.Logger().Info("New available pool", "pool", pool.Asset)
		case PoolStaged:
			ctx.Logger().Info("Pool demoted to staged status", "pool", pool.Asset)
		}
		pool.StatusSince = ctx.BlockHeight()
		return mgr.Keeper().SetPool(ctx, pool)
	}

	iterator := mgr.Keeper().GetPoolIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		if err := mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &pool); err != nil {
			return err
		}
		// skip all cycling on saver pools
		if pool.Asset.IsSyntheticAsset() {
			continue
		}
		if pool.Asset.IsGasAsset() {
			continue
		}
		switch pool.Status {
		case PoolAvailable:
			// any available pools that have no asset, no rune, or less than
			// min rune, moves back to staged status
			if validPool(pool) &&
				pm.poolMeetTradingVolumeCriteria(ctx, mgr, pool, cosmos.NewUint(uint64(minPoolLiquidityFee))) {
				availblePoolCount += 1
			} else {
				pool.Status = PoolStaged
				if err := setPool(pool); err != nil {
					return err
				}
			}
			// reset the pool rolling liquidity fee to zero
			mgr.Keeper().ResetRollingPoolLiquidityFee(ctx, pool.Asset)
			if pool.BalanceRune.LT(choppingBlock.BalanceRune) || choppingBlock.IsEmpty() {
				// omit pools that are gas assets from being on the chopping
				// block, removing these pool requires a chain ragnarok, and
				// cannot be handled individually
				choppingBlock = pool
			}
		case PoolStaged:
			// deduct staged pool rune fee
			fee := cosmos.NewUint(uint64(stagedPoolCost))
			if fee.GT(pool.BalanceRune) {
				fee = pool.BalanceRune
			}
			if !fee.IsZero() {
				pool.BalanceRune = common.SafeSub(pool.BalanceRune, fee)
				if err := mgr.Keeper().SetPool(ctx, pool); err != nil {
					ctx.Logger().Error("fail to save pool", "pool", pool.Asset, "err", err)
				}

				if err := mgr.Keeper().AddPoolFeeToReserve(ctx, fee); err != nil {
					ctx.Logger().Error("fail to add rune to reserve", "from pool", pool.Asset, "err", err)
				}

				emitPoolBalanceChangedEvent(ctx,
					NewPoolMod(pool.Asset, fee, false, cosmos.ZeroUint(), false),
					"pool stage cost",
					mgr)
			}
			// check if the rune balance is zero, and asset balance IS NOT
			// zero. This is because we don't want to abandon a pool that is in
			// the process of being created (race condition). We can safely
			// assume, if a pool has asset, but no rune, it should be
			// abandoned.
			if pool.BalanceRune.IsZero() && !pool.BalanceAsset.IsZero() {
				// the staged pool no longer has any rune, abandon the pool
				// and liquidity provider, and burn the asset (via zero'ing
				// the vaults for the asset, and churning away from the
				// tokens)
				ctx.Logger().Info("burning pool", "pool", pool.Asset)

				// Transfer any pending RUNE to the Reserve so it isn't left in the Pool Module after pool deletion
				if !pool.PendingInboundRune.IsZero() {
					if err := mgr.Keeper().AddPoolFeeToReserve(ctx, pool.PendingInboundRune); err != nil {
						ctx.Logger().Error("fail to transfer pending inbound rune to reserve during pool burning", "from pool", pool.Asset, "err", err)
					}
				}

				// remove LPs
				pm.removeLiquidityProviders(ctx, pool.Asset, mgr)

				// delete the pool
				mgr.Keeper().RemovePool(ctx, pool.Asset)

				poolEvent := NewEventPool(pool.Asset, PoolSuspended)
				if err := mgr.EventMgr().EmitEvent(ctx, poolEvent); err != nil {
					ctx.Logger().Error("fail to emit pool event", "error", err)
				}
				// remove asset from Vault
				pm.removeAssetFromVault(ctx, pool.Asset, mgr)

			} else if validPool(pool) && onDeck.BalanceRune.LT(pool.BalanceRune) {
				onDeck = pool
			}
		}
	}

	if availblePoolCount >= maxAvailablePools {
		// if we've hit our max available pools, and the onDeck pool is less
		// than the chopping block pool, then we do make no changes, by
		// resetting the variables
		if onDeck.BalanceRune.LTE(choppingBlock.BalanceRune) {
			onDeck = NewPool()        // reset
			choppingBlock = NewPool() // reset
		}
	} else {
		// since we haven't hit the max number of available pools, there is no
		// available pool on the chopping block
		choppingBlock = NewPool() // reset
	}

	if !onDeck.IsEmpty() {
		onDeck.Status = PoolAvailable
		if err := setPool(onDeck); err != nil {
			return err
		}
	}

	if !choppingBlock.IsEmpty() {
		choppingBlock.Status = PoolStaged
		if err := setPool(choppingBlock); err != nil {
			return err
		}
	}

	return nil
}

// poolMeetTradingVolumeCriteria check if pool generated the minimum amount of fees since last cycle
func (pm *PoolMgrVCUR) poolMeetTradingVolumeCriteria(ctx cosmos.Context, mgr Manager, pool Pool, minPoolLiquidityFee cosmos.Uint) bool {
	if minPoolLiquidityFee.IsZero() {
		return true
	}
	blockPoolLiquidityFee, err := mgr.Keeper().GetRollingPoolLiquidityFee(ctx, pool.Asset)
	if err != nil {
		ctx.Logger().Error("fail to get rolling pool liquidity from key value store", "error", err)
		// when we failed to get rolling liquidity fee from key value store for some reason, return true here
		// thus the pool will not be demoted
		return true
	}
	return cosmos.NewUint(blockPoolLiquidityFee).GTE(minPoolLiquidityFee)
}

// removeAssetFromVault set asset balance to zero for all vaults holding the asset
func (pm *PoolMgrVCUR) removeAssetFromVault(ctx cosmos.Context, asset common.Asset, mgr Manager) {
	// zero vaults with the pool asset
	vaultIter := mgr.Keeper().GetVaultIterator(ctx)
	defer vaultIter.Close()
	for ; vaultIter.Valid(); vaultIter.Next() {
		var vault Vault
		if err := mgr.Keeper().Cdc().Unmarshal(vaultIter.Value(), &vault); err != nil {
			ctx.Logger().Error("fail to unmarshal vault", "error", err)
			continue
		}
		if vault.HasAsset(asset) {
			for i, coin := range vault.Coins {
				if asset.Equals(coin.Asset) {
					vault.Coins[i].Amount = cosmos.ZeroUint()
					if err := mgr.Keeper().SetVault(ctx, vault); err != nil {
						ctx.Logger().Error("fail to save vault", "error", err)
					}
					break
				}
			}
		}
	}
}

// removeLiquidityProviders remove all lps for the given asset pool
func (pm *PoolMgrVCUR) removeLiquidityProviders(ctx cosmos.Context, asset common.Asset, mgr Manager) {
	iterator := mgr.Keeper().GetLiquidityProviderIterator(ctx, asset)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var lp LiquidityProvider
		if err := mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &lp); err != nil {
			ctx.Logger().Error("fail to unmarshal liquidity provider", "error", err)
			continue
		}

		// fields must be populated with empty values or midgard will not process
		withdrawEvt := NewEventWithdraw(
			asset,
			lp.Units,
			int64(10000),
			cosmos.ZeroDec(),
			common.Tx{
				ID:          common.BlankTxID,
				FromAddress: lp.GetAddress(),
				ToAddress:   common.NoAddress,
				Coins:       common.NewCoins(common.NewCoin(common.SwitchNative, cosmos.ZeroUint())),
				Chain:       common.SWITCHLYChain,
			},
			cosmos.ZeroUint(),
			cosmos.ZeroUint(),
		)
		if err := mgr.EventMgr().EmitEvent(ctx, withdrawEvt); err != nil {
			ctx.Logger().Error("fail to emit pool withdraw event", "error", err)
		}

		mgr.Keeper().RemoveLiquidityProvider(ctx, lp)
	}
}

func (pm *PoolMgrVCUR) cleanupPendingLiquidity(ctx cosmos.Context, mgr Manager) {
	if atTVLCap(ctx, nil, mgr) {
		ctx.Logger().Info("cleaning pending liquidity skipped due to TVL cap")
		return
	}

	pendingLiquidityAgeLimit := mgr.Keeper().GetConfigInt64(ctx, constants.PendingLiquidityAgeLimit)
	if pendingLiquidityAgeLimit <= 0 {
		return
	}

	var pools Pools
	iterator := mgr.Keeper().GetPoolIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		err := mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &pool)
		if err != nil {
			ctx.Logger().Error("fail to unmarshal pool for cleanup pending liquidity", "error", err)
			continue
		}
		if pool.Asset.IsDerivedAsset() || pool.Asset.IsSyntheticAsset() {
			continue
		}
		if !pool.IsAvailable() && !pool.IsStaged() {
			continue
		}
		// no need to commit pending liquidity when there is none, quick exit
		if pool.PendingInboundRune.IsZero() && pool.PendingInboundAsset.IsZero() {
			continue
		}
		if mgr.Keeper().IsChainHalted(ctx, pool.Asset.GetChain()) || mgr.Keeper().IsLPPaused(ctx, pool.Asset.GetChain()) {
			continue
		}
		pools = append(pools, pool)
	}

	if len(pools) == 0 {
		return
	}

	// process each pool within ageLimit evenly (in terms of blocks between
	// each pool). For example, if ageLimit is 100 blocks, and we have 5 pools,
	// we want to clean a pool every ~20 blocks, but each pool is only cleaned
	// once every 100 blocks (just a different 100 blocks)
	separator := pendingLiquidityAgeLimit / int64(len(pools))
	if separator == 0 {
		// If PendingLiquidityAgeLimit is smaller than the number of pools,
		// still spread them out over the available blocks (and wrap around).
		separator = 1
	}
	cleanupTarget := ctx.BlockHeight() % pendingLiquidityAgeLimit
	for i, pool := range pools {
		if cleanupTarget != (separator*int64(i))%pendingLiquidityAgeLimit {
			continue
		}
		if err := pm.commitPendingLiquidity(ctx, pool, mgr); err != nil {
			ctx.Logger().Error("fail to clean pending liquidity", "pool", pool.Asset, "error", err)
		}
	}
}

func (pm *PoolMgrVCUR) checkSaversUtilization(ctx cosmos.Context, mgr Manager) error {
	if mgr.Keeper().IsGlobalTradingHalted(ctx) {
		return nil
	}
	saversFreq := mgr.Keeper().GetConfigInt64(ctx, constants.SaversEjectInterval)
	if saversFreq <= 0 || ctx.BlockHeight()%saversFreq != 0 {
		return nil
	}

	nodeAccounts, err := mgr.Keeper().ListActiveValidators(ctx)
	if err != nil {
		return err
	}

	if len(nodeAccounts) == 0 {
		return fmt.Errorf("dev err: no active node accounts")
	}

	var (
		synthPool           Pool
		maxUtilizationRatio = sdkmath.ZeroUint()
		handler             = NewInternalHandler(mgr)
		signer              = nodeAccounts[0].NodeAddress
		synthCap            = mgr.Keeper().GetConfigInt64(ctx, constants.MaxSynthPerPoolDepth)
	)
	iterator := mgr.Keeper().GetPoolIterator(ctx)
	defer iterator.Close()

	// Find the pool with the highest utilization ratio above the cap
	for ; iterator.Valid(); iterator.Next() {
		var pool Pool
		err = mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &pool)
		if err != nil {
			ctx.Logger().Error("fail to unmarshal pool for synth utilization check", "error", err)
			continue
		}

		// skip non synth asset pool
		if !pool.Asset.IsSyntheticAsset() {
			continue
		}

		if !pool.IsAvailable() && !pool.IsStaged() {
			continue
		}

		l1Chain := pool.Asset.GetLayer1Asset().GetChain()
		if mgr.Keeper().IsChainTradingHalted(ctx, l1Chain) || mgr.Keeper().IsLPPaused(ctx, l1Chain) {
			continue
		}

		if mgr.Keeper().IsRagnarok(ctx, []common.Asset{pool.Asset.GetLayer1Asset()}) {
			continue
		}

		synthSupply := mgr.Keeper().GetTotalSupply(ctx, pool.Asset.GetSyntheticAsset())
		l1Pool, err := mgr.Keeper().GetPool(ctx, pool.Asset.GetLayer1Asset())
		if err != nil {
			continue
		}

		maxSynth := common.GetUncappedShare(cosmos.NewUint(uint64(synthCap)), cosmos.NewUint(constants.MaxBasisPts), l1Pool.BalanceAsset.MulUint64(2))
		if synthSupply.GT(maxSynth) {
			utilizationRatio := common.GetUncappedShare(maxSynth, synthSupply, sdkmath.NewUint(constants.MaxBasisPts))
			if synthPool.IsEmpty() || utilizationRatio.GT(maxUtilizationRatio) {
				synthPool = pool
				maxUtilizationRatio = utilizationRatio
			}
		}
	}
	if synthPool.IsEmpty() {
		// No pool found with synth utilization above the cap
		return nil
	}

	lpIterator := mgr.Keeper().GetLiquidityProviderIterator(ctx, synthPool.Asset)
	defer lpIterator.Close()
	var latestLp LiquidityProvider
	for ; lpIterator.Valid(); lpIterator.Next() {
		var lp LiquidityProvider
		if err := mgr.Keeper().Cdc().Unmarshal(lpIterator.Value(), &lp); err != nil {
			ctx.Logger().Error("fail to unmarshal liquidity provider", "error", err)
			continue
		}
		// get the last added lp
		if lp.LastAddHeight > latestLp.LastAddHeight {
			latestLp = lp
		} else if lp.LastAddHeight == latestLp.LastAddHeight {
			// withdraw smaller position
			if lp.Units.LT(latestLp.Units) {
				latestLp = lp
			}
		}
	}

	// sanity check
	if latestLp.LastAddHeight == 0 || latestLp.AssetAddress.IsEmpty() {
		ctx.Logger().Error("saver utilization exceeded without any LP position", "asset", synthPool.Asset)
		return nil
	}

	coins := common.NewCoins(common.NewCoin(synthPool.Asset.GetLayer1Asset(), cosmos.ZeroUint())) // used in tx hash
	tx := common.NewTx(
		common.BlankTxID, latestLp.AssetAddress, latestLp.AssetAddress, coins, nil, "THOR-SAVER-EJECT",
	)
	tx.Chain = synthPool.Asset.GetChain()
	tx.ID, err = common.NewTxID(tx.Hash(ctx.BlockHeight()))
	if err != nil {
		ctx.Logger().Error("fail to create tx id", "error", err, "tx", tx)
		return nil
	}

	withdrawMsg := NewMsgWithdrawLiquidity(
		tx,
		latestLp.AssetAddress,
		cosmos.NewUint(constants.MaxBasisPts),
		synthPool.Asset.GetSyntheticAsset(),
		common.EmptyAsset,
		signer,
	)
	ctx.Logger().Info("closing saver LP position", "tx_id", tx.ID, "pool", synthPool.Asset, "units", latestLp.Units, "asset_address", latestLp.AssetAddress)
	_, err = handler(ctx, withdrawMsg)
	if err != nil {
		ctx.Logger().Error("fail to eject saver position", "asset", synthPool.Asset, "address", latestLp.AssetAddress, "error", err)
	}

	return nil
}

// commitPendingLiquidity - for aged pending liquidity, commit it to the pool
func (pm *PoolMgrVCUR) commitPendingLiquidity(ctx cosmos.Context, pool Pool, mgr Manager) error {
	ctx.Logger().Info("cleaning pending liquidity in pool", "pool", pool.Asset)
	// track stats
	var count int
	cleanedRune := cosmos.ZeroUint()
	cleanedAsset := cosmos.ZeroUint()

	// no need to commit pending liquidity when there is none, quick exit
	if pool.PendingInboundRune.IsZero() && pool.PendingInboundAsset.IsZero() {
		return nil
	}

	// get a signer of the txn
	nodeAccounts, err := mgr.Keeper().ListActiveValidators(ctx)
	if err != nil {
		return err
	}
	if len(nodeAccounts) == 0 {
		return fmt.Errorf("dev err: no active node accounts")
	}
	signer := nodeAccounts[0].NodeAddress

	handler := NewInternalHandler(mgr)
	pendingLiquidityAgeLimit := mgr.Keeper().GetConfigInt64(ctx, constants.PendingLiquidityAgeLimit)

	iterator := mgr.Keeper().GetLiquidityProviderIterator(ctx, pool.Asset)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		var lp LiquidityProvider
		if err := mgr.Keeper().Cdc().Unmarshal(iterator.Value(), &lp); err != nil {
			ctx.Logger().Error("fail to unmarshal liquidity provider", "error", err)
			continue
		}

		// check if this LP has any pending liquidity, quick exit if it doesn't
		if lp.PendingRune.IsZero() && lp.PendingAsset.IsZero() {
			continue
		}

		if lp.AssetAddress.IsEmpty() || lp.RuneAddress.IsEmpty() {
			continue
		}

		// check if last add height is within our pendingLiquidityAgeLimit
		if ctx.BlockHeight()-lp.LastAddHeight <= pendingLiquidityAgeLimit {
			continue
		}

		runeAdd := cosmos.ZeroUint()
		assetAdd := cosmos.ZeroUint()
		tx := common.Tx{
			ID:        common.BlankTxID,
			ToAddress: common.NoAddress,
			Memo:      "THOR-PENDING-LIQUIDITY-ADD",
		}

		if !lp.PendingRune.IsZero() {
			tx.FromAddress = lp.AssetAddress
			tx.Chain = pool.Asset.GetChain()
			tx.Coins = common.NewCoins(common.NewCoin(pool.Asset, assetAdd))
		}
		if !lp.PendingAsset.IsZero() {
			tx.FromAddress = lp.RuneAddress
			tx.Chain = common.SWITCHLYChain
			tx.Coins = common.NewCoins(common.NewCoin(common.SwitchNative, runeAdd))
		}

		msg := NewMsgAddLiquidity(tx, lp.Asset, runeAdd, assetAdd, lp.RuneAddress, lp.AssetAddress, common.NoAddress, cosmos.ZeroUint(), signer)
		_, err := handler(ctx, msg)
		if err != nil {
			ctx.Logger().Error("failed to commit pending liquidity", "asset", lp.Asset, "thor address", lp.RuneAddress, "asset address", lp.AssetAddress, "pending rune", lp.PendingRune, "pending asset", lp.PendingAsset, "error", err)
			// since we failed to clear pending liquidity, lets add it to the
			// running total for the pool
			continue
		}

		count += 1
		cleanedRune = cleanedRune.Add(lp.PendingRune)
		cleanedAsset = cleanedRune.Add(lp.PendingAsset)
	}
	ctx.Logger().Info("cleaned pending liquidity", "pool", pool.Asset, "count", count, "cleared rune", cleanedRune.String(), "cleared asset", cleanedAsset.String())

	// add telemetry
	telemetry.IncrCounterWithLabels(
		[]string{"thornode", "pool", "pending", "clear", "count"},
		float32(count),
		[]metrics.Label{telemetry.NewLabel("pool", pool.Asset.String())},
	)
	telemetry.IncrCounterWithLabels(
		[]string{"thornode", "pool", "pending", "clear", "rune"},
		telem(cleanedRune),
		[]metrics.Label{telemetry.NewLabel("pool", pool.Asset.String())},
	)
	telemetry.IncrCounterWithLabels(
		[]string{"thornode", "pool", "pending", "clear", "asset"},
		telem(cleanedAsset),
		[]metrics.Label{telemetry.NewLabel("pool", pool.Asset.String())},
	)

	return nil
}
