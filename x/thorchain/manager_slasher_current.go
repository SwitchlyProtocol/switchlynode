package thorchain

import (
	"context"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"cosmossdk.io/core/comet"
	"github.com/cometbft/cometbft/crypto"
	"github.com/cosmos/cosmos-sdk/telemetry"
	"github.com/hashicorp/go-metrics"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

// SlasherVCUR is VCUR implementation of slasher
type SlasherVCUR struct {
	keeper   keeper.Keeper
	eventMgr EventManager
}

// newSlasherVCUR create a new instance of Slasher
func newSlasherVCUR(keeper keeper.Keeper, eventMgr EventManager) *SlasherVCUR {
	return &SlasherVCUR{keeper: keeper, eventMgr: eventMgr}
}

// BeginBlock called when a new block get proposed to detect whether there are duplicate vote
func (s *SlasherVCUR) BeginBlock(ctx cosmos.Context, constAccessor constants.ConstantValues) {
	var doubleSignEvidence []comet.Evidence
	// Iterate through any newly discovered evidence of infraction
	// Slash any validators (and since-unbonded liquidity within the unbonding period)
	// who contributed to valid infractions
	for i := range ctx.CometInfo().GetEvidence().Len() {
		evidence := ctx.CometInfo().GetEvidence().Get(i)
		switch evidence.Type() {
		case comet.DuplicateVote:
			doubleSignEvidence = append(doubleSignEvidence, evidence)
		default:
			ctx.Logger().Error("ignored unknown evidence type", "type", evidence.Type)
		}
	}

	// Identify validators which didn't sign the previous block
	var missingSignAddresses []crypto.Address
	var successfulSignAddresses []crypto.Address
	for i := range ctx.CometInfo().GetLastCommit().Votes().Len() {
		voteInfo := ctx.CometInfo().GetLastCommit().Votes().Get(i)
		if voteInfo.GetBlockIDFlag() != comet.BlockIDFlagAbsent {
			successfulSignAddresses = append(successfulSignAddresses, voteInfo.Validator().Address())
		} else {
			missingSignAddresses = append(missingSignAddresses, voteInfo.Validator().Address())
		}
	}

	// Do not continue if there is no action to take.
	if len(doubleSignEvidence)+len(missingSignAddresses) == 0 {
		return
	}

	// Derive Active node validator addresses once.
	nas, err := s.keeper.ListActiveValidators(ctx)
	if err != nil {
		ctx.Logger().Error("fail to list active validators", "error", err)
		return
	}
	var validatorAddresses []nodeAddressValidatorAddressPair
	for _, na := range nas {
		pk, err := cosmos.GetPubKeyFromBech32(cosmos.Bech32PubKeyTypeConsPub, na.ValidatorConsPubKey)
		if err != nil {
			ctx.Logger().Error("fail to derive validator address", "error", err)
			continue
		}
		var pair nodeAddressValidatorAddressPair
		pair.nodeAddress = na.NodeAddress
		pair.validatorAddress = pk.Address()
		validatorAddresses = append(validatorAddresses, pair)
	}

	// Act on double signs.
	for _, evidence := range doubleSignEvidence {
		if err := s.HandleDoubleSign(ctx, evidence.Validator().Address(), evidence.Height(), constAccessor, validatorAddresses); err != nil {
			ctx.Logger().Error("fail to slash for double signing a block", "error", err)
		}
	}

	// Act on missing signs.
	for _, missingSignAddress := range missingSignAddresses {
		if err := s.HandleMissingSign(ctx, missingSignAddress, constAccessor, validatorAddresses); err != nil {
			ctx.Logger().Error("fail to slash for missing signing a block", "error", err)
		}
	}

	// Act on successful signs.
	for _, successfulSignAddress := range successfulSignAddresses {
		if err := s.HandleSuccessfulSign(ctx, successfulSignAddress, constAccessor, validatorAddresses); err != nil {
			ctx.Logger().Error("fail to mark for successfully signing a block", "error", err)
		}
	}
}

// HandleDoubleSign - slashes a validator for signing two blocks at the same
// block height
// https://blog.cosmos.network/consensus-compare-casper-vs-tendermint-6df154ad56ae
func (s *SlasherVCUR) HandleDoubleSign(ctx cosmos.Context, addr crypto.Address, infractionHeight int64, constAccessor constants.ConstantValues, validatorAddresses []nodeAddressValidatorAddressPair) error {
	// check if we're recent enough to slash for this behavior
	maxAge := constAccessor.GetInt64Value(constants.DoubleSignMaxAge)
	if (ctx.BlockHeight() - infractionHeight) > maxAge {
		ctx.Logger().Info("double sign detected but too old to be slashed", "infraction height", fmt.Sprintf("%d", infractionHeight), "address", addr.String())
		return nil
	}

	doubleBlockSignSlashPoints := s.keeper.GetConfigInt64(ctx, constants.DoubleBlockSignSlashPoints)
	for _, pair := range validatorAddresses {
		if addr.String() != pair.validatorAddress.String() {
			continue
		}

		na, err := s.keeper.GetNodeAccount(ctx, pair.nodeAddress)
		if err != nil {
			return err
		}

		slashCtx := ctx.WithContext(context.WithValue(ctx.Context(), constants.CtxMetricLabels, []metrics.Label{
			telemetry.NewLabel("address", na.NodeAddress.String()),
			telemetry.NewLabel("reason", "double_block_sign"),
		}))
		if err := s.keeper.IncNodeAccountSlashPoints(slashCtx, na.NodeAddress, doubleBlockSignSlashPoints); err != nil {
			ctx.Logger().Error("fail to increase node account slash point", "error", err, "address", na.NodeAddress.String())
		}

		return s.keeper.SetNodeAccount(ctx, na)
	}

	return fmt.Errorf("could not find active node account with validator address: %s", addr)
}

// HandleSuccessfulSign - decrement missing blocks from a validator for signing a block
func (s *SlasherVCUR) HandleSuccessfulSign(ctx cosmos.Context, addr crypto.Address, constAccessor constants.ConstantValues, validatorAddresses []nodeAddressValidatorAddressPair) error {
	for _, pair := range validatorAddresses {
		if addr.String() != pair.validatorAddress.String() {
			continue
		}

		na, err := s.keeper.GetNodeAccount(ctx, pair.nodeAddress)
		if err != nil {
			return err
		}

		if na.MissingBlocks == 0 {
			return nil
		}

		// decrement the number of blocks that weren't signed
		na.MissingBlocks -= 1

		return s.keeper.SetNodeAccount(ctx, na)
	}

	return fmt.Errorf("could not find active node account with validator address: %s", addr)
}

// HandleMissingSign - slashes a validator for not signing a block
func (s *SlasherVCUR) HandleMissingSign(ctx cosmos.Context, addr crypto.Address, constAccessor constants.ConstantValues, validatorAddresses []nodeAddressValidatorAddressPair) error {
	missBlockSignSlashPoints := s.keeper.GetConfigInt64(ctx, constants.MissBlockSignSlashPoints)
	maxTrack := s.keeper.GetConfigInt64(ctx, constants.MaxTrackMissingBlock)

	for _, pair := range validatorAddresses {
		if addr.String() != pair.validatorAddress.String() {
			continue
		}

		na, err := s.keeper.GetNodeAccount(ctx, pair.nodeAddress)
		if err != nil {
			return err
		}

		slashCtx := ctx.WithContext(context.WithValue(ctx.Context(), constants.CtxMetricLabels, []metrics.Label{
			telemetry.NewLabel("address", na.NodeAddress.String()),
			telemetry.NewLabel("reason", "miss_block_sign"),
		}))
		if err := s.keeper.IncNodeAccountSlashPoints(slashCtx, na.NodeAddress, missBlockSignSlashPoints); err != nil {
			ctx.Logger().Error("fail to increase node account slash points", "error", err, "address", na.NodeAddress.String())
		}

		// increment the number of blocks that weren't signed
		na.MissingBlocks += 1
		if na.MissingBlocks > uint64(maxTrack) {
			na.MissingBlocks = uint64(maxTrack)
		}

		return s.keeper.SetNodeAccount(ctx, na)
	}

	return fmt.Errorf("could not find active node account with validator address: %s", addr)
}

// LackSigning slash account that fail to sign tx
func (s *SlasherVCUR) LackSigning(ctx cosmos.Context, mgr Manager) error {
	var resultErr error
	maxOutboundAttempts := mgr.Keeper().GetConfigInt64(ctx, constants.MaxOutboundAttempts)
	signingTransPeriod := mgr.GetConstants().GetInt64Value(constants.SigningTransactionPeriod)
	if ctx.BlockHeight() < signingTransPeriod {
		return nil
	}
	height := ctx.BlockHeight() - signingTransPeriod
	txs, err := s.keeper.GetTxOut(ctx, height)
	if err != nil {
		return fmt.Errorf("fail to get txout from block height(%d): %w", height, err)
	}

	// round up to nearest multiple of RescheduleCoalesceBlocks
	rescheduleHeight := ctx.BlockHeight()
	rescheduleCoalesceBlocks := mgr.Keeper().GetConfigInt64(ctx, constants.RescheduleCoalesceBlocks)
	if rescheduleCoalesceBlocks > 1 {
		overBlocks := rescheduleHeight % rescheduleCoalesceBlocks
		if overBlocks != 0 {
			rescheduleHeight += rescheduleCoalesceBlocks - overBlocks
		}
	}

	for i, toi := range txs.TxArray {
		if !common.CurrentChainNetwork.SoftEquals(toi.ToAddress.GetNetwork(toi.Chain)) {
			continue // skip this transaction
		}
		if toi.OutHash.IsEmpty() {
			// Slash node account for not sending funds
			vault, err := s.keeper.GetVault(ctx, toi.VaultPubKey)
			if err != nil {
				// in some edge cases the vault may no longer exists, in which
				// case log and continue with rescheduling
				ctx.Logger().Error("Unable to get vault", "error", err, "vault pub key", toi.VaultPubKey.String())
			}

			// if the vault is frozen, reschedule to same vault with no changes
			frozen := false
			if len(vault.Frozen) > 0 {
				chains, err := common.NewChains(vault.Frozen)
				if err != nil {
					ctx.Logger().Error("failed to convert chains", "error", err)
				}
				if chains.Has(toi.Coin.Asset.GetChain()) {
					etx := common.Tx{
						ID:        toi.InHash,
						Chain:     toi.Chain,
						ToAddress: toi.ToAddress,
						Coins:     []common.Coin{toi.Coin},
						Gas:       toi.MaxGas,
						Memo:      toi.Memo,
					}
					eve := NewEventSecurity(etx, "frozen vault reschedule")
					if err := mgr.EventMgr().EmitEvent(ctx, eve); err != nil {
						ctx.Logger().Error("fail to emit security event", "error", err)
					}
					frozen = true
				}
			}

			memo, _ := ParseMemoWithTHORNames(ctx, s.keeper, toi.Memo) // ignore err
			if memo.IsInternal() {
				// there is a different mechanism for rescheduling outbound
				// transactions for migration transactions
				continue
			}
			var voter ObservedTxVoter
			if memo.IsType(TxRagnarok) {
				// A Ragnarok outbound has no ObservedTxInVoter,
				// so check MaxOutboundAttempts against the memo height.
				ragnarokHeight := memo.GetBlockHeight()
				// Though a negative is not expected to cause problems,
				// nevertheless keep the default 0 as the minimum.
				if ragnarokHeight > 0 {
					voter.FinalisedHeight = ragnarokHeight
				}
			} else {
				voter, err = s.keeper.GetObservedTxInVoter(ctx, toi.InHash)
				if err != nil {
					ctx.Logger().Error("fail to get observed tx voter", "error", err)
					resultErr = fmt.Errorf("failed to get observed tx voter: %w", err)
					continue
				}
			}

			if maxOutboundAttempts > 0 {
				age := ctx.BlockHeight() - voter.FinalisedHeight
				attempts := age / signingTransPeriod
				if attempts >= maxOutboundAttempts {
					ctx.Logger().Info("txn dropped, too many attempts", "hash", toi.InHash)
					continue
				}
			}

			// if vault is inactive, do not reassign the outbound txn to
			// another vault
			if vault.Status == InactiveVault {
				ctx.Logger().Info("cannot reassign outbound from inactive vault", "hash", toi.InHash)
				continue
			}

			if !frozen && s.needsNewVault(ctx, mgr, vault, signingTransPeriod, voter.FinalisedHeight, toi) {
				active, err := s.keeper.GetAsgardVaultsByStatus(ctx, ActiveVault)
				if err != nil {
					return fmt.Errorf("fail to get active asgard vaults: %w", err)
				}
				// Deduct the asset's pending outbound funds to represent only the available funds.
				pendingOutbounds := mgr.Keeper().GetPendingOutbounds(ctx, toi.Coin.Asset)
				for i := range active {
					active[i].DeductVaultPendingOutbounds(pendingOutbounds)

					// If the currently-assigned vault is an ActiveVault and the only one with enough funds for the outbound,
					// the item should be reassigned to the same vault rather than assigned to another vault without enough funds;
					// this is especially important for GAIA outbounds, for which insufficient-funds failures
					// have SwitchlyProtocol-unobserved gas costs (causing churn-migration-jamming vault insolvency).
					// As such, re-add the (now free) funds of the outbound being replaced.
					if active[i].PubKey.Equals(toi.VaultPubKey) {
						active[i].Coins = active[i].Coins.Add(toi.Coin)
						active[i].Coins = active[i].Coins.Add(toi.MaxGas...)
					}
				}

				available := active
				mainCoin := toi.Coin
				maxGasCoin, err := mgr.GasMgr().GetMaxGas(ctx, toi.Chain)
				if err != nil {
					ctx.Logger().Error("fail to get max gas", "error", err)
				}
				if mainCoin.Asset.Equals(maxGasCoin.Asset) {
					// If the main coin and the gas coin are the same asset,
					// ensure the assigned vault has enough available for both.
					mainCoin.Amount = mainCoin.Amount.Add(maxGasCoin.Amount)
				} else {
					// If the gas coin isn't the main asset,
					// directly ensure the assigned vault has enough available for it.
					available = active.Has(maxGasCoin)
				}
				available = available.Has(mainCoin)

				if len(available) == 0 {
					// we need to give it somewhere to send from, even if that
					// asgard doesn't have enough funds. This is because if we
					// don't the transaction will just be dropped on the floor,
					// which is bad. Instead it may try to send from an asgard that
					// doesn't have enough funds, fail, and then get rescheduled
					// again later. Maybe by then the network will have enough
					// funds to satisfy.
					// TODO add split logic to send it out from multiple asgards in
					// this edge case.
					ctx.Logger().Error("unable to determine asgard vault to send funds, trying first asgard")
					if len(active) > 0 {
						// Fall back on the vault with the most available funds.
						vault = active.SortBy(mainCoin.Asset)[0]
					}
				} else {
					rep := int(toi.InHash.Int64() + ctx.BlockHeight()/signingTransPeriod)
					if vault.PubKey.Equals(available[rep%len(available)].PubKey) {
						// looks like the new vault is going to be the same as the
						// old vault, increment rep to ensure a differ asgard is
						// chosen (if there is more than one option)
						rep++
					}
					vault = available[rep%len(available)]
				}
				if !memo.IsType(TxRagnarok) {
					// update original toi action in observed tx
					// check observedTx has done status. Skip if it does already.
					voterTx := voter.GetTx(NodeAccounts{})
					if voterTx.IsDone(len(voter.Actions)) {
						if len(voterTx.OutHashes) > 0 && len(voterTx.GetOutHashes()) > 0 {
							txs.TxArray[i].OutHash = voterTx.GetOutHashes()[0]
						}
						continue
					}

					// update the actions in the voter with the new vault pubkey
					for i, action := range voter.Actions {
						if action.Equals(toi) {
							voter.Actions[i].VaultPubKey = vault.PubKey

							if toi.Aggregator != "" || toi.AggregatorTargetAsset != "" || toi.AggregatorTargetLimit != nil {
								ctx.Logger().Info("clearing aggregator fields on outbound reassignment", "hash", toi.InHash)

								// Here, simultaneously clear the Aggregator information for a reassigned TxOutItem and its Actions item
								// so that a SwapOut will send the SwitchlyProtocol output asset instead of cycling and swallowingif
								// (and maybe failing with slashes) if something goes wrong.
								toi.Aggregator = ""
								toi.AggregatorTargetAsset = ""
								toi.AggregatorTargetLimit = nil
								voter.Actions[i].Aggregator = ""
								voter.Actions[i].AggregatorTargetAsset = ""
								voter.Actions[i].AggregatorTargetLimit = nil
							}
						}
					}
					s.keeper.SetObservedTxInVoter(ctx, voter)

				}
				// Save the toi to as a new toi, select Asgard to send it this time.
				toi.VaultPubKey = vault.PubKey

				// update max gas
				if !maxGasCoin.IsEmpty() {
					toi.MaxGas = common.Gas{maxGasCoin}
					// Update MaxGas in ObservedTxVoter action as well
					if err := updateTxOutGas(ctx, s.keeper, toi, common.Gas{maxGasCoin}); err != nil {
						ctx.Logger().Error("Failed to update MaxGas of action in ObservedTxVoter", "hash", toi.InHash, "error", err)
					}
				}
				// Equals checks GasRate so update actions GasRate too (before updating in the queue item)
				// for future updates of MaxGas, which must match for matchActionItem in AddOutTx.
				gasRate := int64(mgr.GasMgr().GetGasRate(ctx, toi.Chain).Uint64())
				if err := updateTxOutGasRate(ctx, s.keeper, toi, gasRate); err != nil {
					ctx.Logger().Error("Failed to update GasRate of action in ObservedTxVoter", "hash", toi.InHash, "error", err)
				}
				toi.GasRate = gasRate
			}

			// if a pool with the asset name doesn't exist, skip rescheduling
			if !toi.Coin.IsSwitch() && !s.keeper.PoolExist(ctx, toi.Coin.Asset) {
				ctx.Logger().Error("fail to add outbound to queue", "error", "coin is not rune and does not have an associated pool")
				continue
			}

			err = mgr.TxOutStore().UnSafeAddTxOutItem(ctx, mgr, toi, rescheduleHeight)
			if err != nil {
				ctx.Logger().Error("fail to add outbound to queue", "error", err)
				resultErr = fmt.Errorf("failed to add outbound to queue: %w", err)
				continue
			}
			// because the txout item has been rescheduled, thus mark the replaced tx out item as already send out, even it is not
			// in this way bifrost will not send it out again cause node to be slashed
			txs.TxArray[i].OutHash = common.BlankTxID
		}
	}
	if !txs.IsEmpty() {
		if err := s.keeper.SetTxOut(ctx, txs); err != nil {
			return fmt.Errorf("fail to save tx out : %w", err)
		}
	}

	return resultErr
}

// SlashVault thorchain keep monitoring the outbound tx from asgard pool
// usually the txout is triggered by thorchain itself by
// adding an item into the txout array, refer to TxOutItem for the detail, the
// TxOutItem contains a specific coin and amount.  if somehow thorchain
// discover signer send out fund more than the amount specified in TxOutItem,
// it will slash the node account who does that by taking 1.5 * extra fund from
// node account's bond and subsidise the pool that actually lost it.
func (s *SlasherVCUR) SlashVault(ctx cosmos.Context, vaultPK common.PubKey, coins common.Coins, mgr Manager) error {
	if coins.IsEmpty() {
		return nil
	}

	vault, err := s.keeper.GetVault(ctx, vaultPK)
	if err != nil {
		return fmt.Errorf("fail to get slash vault (pubkey %s), %w", vaultPK, err)
	}
	membership := vault.GetMembership()

	// sum the total bond of membership of the vault
	totalBond := cosmos.ZeroUint()
	for _, member := range membership {
		na, err := s.keeper.GetNodeAccountByPubKey(ctx, member)
		if err != nil {
			ctx.Logger().Error("fail to get node account bond", "pk", member, "error", err)
			continue
		}
		totalBond = totalBond.Add(na.Bond)
	}

	for _, coin := range coins {
		if coin.IsEmpty() {
			continue
		}
		pool, err := s.keeper.GetPool(ctx, coin.Asset)
		if err != nil {
			ctx.Logger().Error("fail to get pool for slash", "asset", coin.Asset, "error", err)
			continue
		}
		// SwitchlyProtocol doesn't even have a pool for the asset
		if pool.IsEmpty() {
			ctx.Logger().Error("cannot slash for an empty pool", "asset", coin.Asset)
			continue
		}

		stolenAssetValue := coin.Amount
		vaultAmount := vault.GetCoin(coin.Asset).Amount
		if stolenAssetValue.GT(vaultAmount) {
			stolenAssetValue = vaultAmount
		}
		if stolenAssetValue.GT(pool.BalanceAsset) {
			stolenAssetValue = pool.BalanceAsset
		}

		// stolenRuneValue is the value in RUNE of the missing funds
		stolenRuneValue := pool.AssetValueInRune(stolenAssetValue)

		if stolenRuneValue.IsZero() {
			continue
		}

		penaltyPts := mgr.Keeper().GetConfigInt64(ctx, constants.SlashPenalty)
		// total slash amount is penaltyPts the RUNE value of the missing funds
		totalRuneToSlash := common.GetUncappedShare(cosmos.NewUint(uint64(penaltyPts)), cosmos.NewUint(10_000), stolenRuneValue)
		totalRuneSlashed := cosmos.ZeroUint()
		pauseOnSlashThreshold := mgr.Keeper().GetConfigInt64(ctx, constants.PauseOnSlashThreshold)
		if pauseOnSlashThreshold > 0 && totalRuneToSlash.GTE(cosmos.NewUint(uint64(pauseOnSlashThreshold))) {
			// set mimirs to pause the chain
			key := fmt.Sprintf("Halt%sChain", coin.Asset.Chain)
			s.keeper.SetMimir(ctx, key, ctx.BlockHeight())
			mimirEvent := NewEventSetMimir(strings.ToUpper(key), strconv.FormatInt(ctx.BlockHeight(), 10))
			if err := mgr.EventMgr().EmitEvent(ctx, mimirEvent); err != nil {
				ctx.Logger().Error("fail to emit set_mimir event", "error", err)
			}
		}
		for _, member := range membership {
			na, err := s.keeper.GetNodeAccountByPubKey(ctx, member)
			if err != nil {
				ctx.Logger().Error("fail to get node account for slash", "pk", member, "error", err)
				continue
			}
			if na.Bond.IsZero() {
				ctx.Logger().Info("validator's bond is zero, can't be slashed", "node address", na.NodeAddress.String())
				continue
			}
			runeSlashed := s.slashAndUpdateNodeAccount(ctx, na, coin, vault, totalBond, totalRuneToSlash)
			totalRuneSlashed = totalRuneSlashed.Add(runeSlashed)
		}

		//  2/3 of the total slashed RUNE value to asgard
		//  1/3 of the total slashed RUNE value to reserve
		runeToAsgard := stolenRuneValue

		// stolenRuneValue is the total value in RUNE of stolen coins, but totalRuneSlashed is
		// the total amount able to be slashed from Nodes, credit the pool only totalRuneSlashed
		if totalRuneSlashed.LT(stolenRuneValue) {
			// this should theoretically never happen
			ctx.Logger().Info("total slashed bond amount is less than RUNE value", "slashed_bond", totalRuneSlashed.String(), "rune_value", stolenRuneValue.String())
			runeToAsgard = totalRuneSlashed
		}
		runeToReserve := common.SafeSub(totalRuneSlashed, runeToAsgard)

		if !runeToReserve.IsZero() {
			if err := s.keeper.SendFromModuleToModule(ctx, BondName, ReserveName, common.NewCoins(common.NewCoin(common.SwitchNative, runeToReserve))); err != nil {
				ctx.Logger().Error("fail to send slash funds to reserve module", "pk", vaultPK, "error", err)
			}
		}
		if !runeToAsgard.IsZero() {
			if err := s.keeper.SendFromModuleToModule(ctx, BondName, AsgardName, common.NewCoins(common.NewCoin(common.SwitchNative, runeToAsgard))); err != nil {
				ctx.Logger().Error("fail to send slash fund to asgard module", "pk", vaultPK, "error", err)
			}
			s.updatePoolFromSlash(ctx, pool, common.NewCoin(coin.Asset, stolenAssetValue), runeToAsgard, mgr)
		}
	}

	return nil
}

// slashAndUpdateNodeAccount slashes a NodeAccount a portion of the value of coin based on their
// portion of the total bond of the offending Vault's membership. Return the amount of RUNE slashed
func (s SlasherVCUR) slashAndUpdateNodeAccount(ctx cosmos.Context, na types.NodeAccount, coin common.Coin, vault types.Vault, totalBond, totalSlashAmountInRune cosmos.Uint) cosmos.Uint {
	slashAmountRune := common.GetSafeShare(na.Bond, totalBond, totalSlashAmountInRune)
	if slashAmountRune.GT(na.Bond) {
		ctx.Logger().Info("slash amount is larger than bond", "slash amount", slashAmountRune, "bond", na.Bond)
		slashAmountRune = na.Bond
	}

	ctx.Logger().Info("slash node account", "node address", na.NodeAddress.String(), "amount", slashAmountRune.String(), "total slash amount", totalSlashAmountInRune)
	na.Bond = common.SafeSub(na.Bond, slashAmountRune)

	bondEvent := NewEventBond(slashAmountRune, BondCost, common.Tx{}, &na, nil)
	if err := s.eventMgr.EmitEvent(ctx, bondEvent); err != nil {
		ctx.Logger().Error("fail to emit bond event", "error", err)
	}

	metricLabels, _ := ctx.Context().Value(constants.CtxMetricLabels).([]metrics.Label)
	slashAmountRuneFloat, _ := new(big.Float).SetInt(slashAmountRune.BigInt()).Float32()
	telemetry.IncrCounterWithLabels(
		[]string{"thornode", "bond_slash"},
		slashAmountRuneFloat,
		append(
			metricLabels,
			telemetry.NewLabel("address", na.NodeAddress.String()),
			telemetry.NewLabel("coin_symbol", coin.Asset.Symbol.String()),
			telemetry.NewLabel("coin_chain", string(coin.Asset.Chain)),
			telemetry.NewLabel("vault_type", vault.Type.String()),
			telemetry.NewLabel("vault_status", vault.Status.String()),
		),
	)

	if err := s.keeper.SetNodeAccount(ctx, na); err != nil {
		ctx.Logger().Error("fail to save node account for slash", "error", err)
	}

	return slashAmountRune
}

// IncSlashPoints will increase the given account's slash points
func (s *SlasherVCUR) IncSlashPoints(ctx cosmos.Context, point int64, addresses ...cosmos.AccAddress) {
	for _, addr := range addresses {
		if err := s.keeper.IncNodeAccountSlashPoints(ctx, addr, point); err != nil {
			ctx.Logger().Error("fail to increase node account slash point", "error", err, "address", addr.String())
		}
	}
}

// DecSlashPoints will decrease the given account's slash points
func (s *SlasherVCUR) DecSlashPoints(ctx cosmos.Context, point int64, addresses ...cosmos.AccAddress) {
	for _, addr := range addresses {
		if err := s.keeper.DecNodeAccountSlashPoints(ctx, addr, point); err != nil {
			ctx.Logger().Error("fail to decrease node account slash point", "error", err, "address", addr.String())
		}
	}
}

// updatePoolFromSlash updates a pool's depths and emits appropriate events after a slash
func (s *SlasherVCUR) updatePoolFromSlash(ctx cosmos.Context, pool types.Pool, stolenAsset common.Coin, runeCreditAmt cosmos.Uint, mgr Manager) {
	pool.BalanceAsset = common.SafeSub(pool.BalanceAsset, stolenAsset.Amount)
	pool.BalanceRune = pool.BalanceRune.Add(runeCreditAmt)
	if err := s.keeper.SetPool(ctx, pool); err != nil {
		ctx.Logger().Error("fail to save pool for slash", "asset", pool.Asset, "error", err)
	}
	poolSlashAmt := []PoolAmt{
		{
			Asset:  pool.Asset,
			Amount: 0 - int64(stolenAsset.Amount.Uint64()),
		},
		{
			Asset:  common.SwitchNative,
			Amount: int64(runeCreditAmt.Uint64()),
		},
	}
	eventSlash := NewEventSlash(pool.Asset, poolSlashAmt)
	if err := mgr.EventMgr().EmitEvent(ctx, eventSlash); err != nil {
		ctx.Logger().Error("fail to emit slash event", "error", err)
	}
}

func (s *SlasherVCUR) needsNewVault(ctx cosmos.Context, mgr Manager, vault Vault, signingTransPeriod, startHeight int64, toi TxOutItem) bool {
	outhashes := mgr.Keeper().GetObservedLink(ctx, toi.InHash)
	if len(outhashes) == 0 {
		return true
	}

	for _, hash := range outhashes {
		voter, err := mgr.Keeper().GetObservedTxOutVoter(ctx, hash)
		if err != nil {
			ctx.Logger().Error("fail to get txout voter", "hash", hash, "error", err)
			continue
		}
		if voter.FinalisedHeight > 0 {
			// Finalised observed txouts should have nothing to do with unfulfilled TxOutItems.
			// This finalised txout might for instance be from an output
			// that was split into multiple outbounds from initially-different vaults.
			continue
		}
		// in the event there are multiple observed txouts for a given inhash, we
		// focus on the matching pubkey and asset
		signers := make(map[string]bool)
		for _, tx1 := range voter.Txs {
			if !tx1.ObservedPubKey.Equals(toi.VaultPubKey) ||
				len(tx1.Tx.Coins) != 1 ||
				!tx1.Tx.Coins[0].Asset.Equals(toi.Coin.Asset) {
				continue
			}

			for _, tx := range voter.Txs {
				if !tx.Tx.ID.Equals(hash) {
					continue
				}
				for _, signer := range tx.Signers {
					// Uniquely record each signer for this outbound hash.
					signers[signer] = true
				}
			}
		}
		if len(signers) > 0 {
			var count int // count the number of signers from the assigned vault
			for _, member := range vault.Membership {
				pk, err := common.NewPubKey(member)
				if err != nil {
					continue
				}
				addr, err := pk.GetAddress(common.SWITCHLYChain)
				if err != nil {
					continue
				}
				if _, ok := signers[addr.String()]; ok {
					count++
				}
			}
			// if a super majority of vault members have observed the outbound,
			// then we should not reschedule. If a vault says it sent it, it
			// sent it and shouldn't get another vault to send it (potentially
			// a second time)
			if count > 0 && HasSuperMajority(count, len(vault.Membership)) {
				return false
			}
			maxHeight := startHeight + ((int64(len(signers)) + 1) * signingTransPeriod)
			return maxHeight < ctx.BlockHeight()
		}

	}

	return true
}
