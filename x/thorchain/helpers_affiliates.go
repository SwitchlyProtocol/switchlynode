package thorchain

import (
	"crypto/sha256"
	"fmt"

	"github.com/blang/semver"
	"gitlab.com/thorchain/thornode/v3/common"
	"gitlab.com/thorchain/thornode/v3/common/cosmos"
	"gitlab.com/thorchain/thornode/v3/constants"
	thorchain "gitlab.com/thorchain/thornode/v3/x/thorchain/memo"
)

func triggerPreferredAssetSwap(ctx cosmos.Context, mgr Manager, affiliateAddress common.Address, txID common.TxID, tn THORName, affcol AffiliateFeeCollector, queueIndex int) error {
	version := mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return triggerPreferredAssetSwapV3_0_0(ctx, mgr, tn, affcol, queueIndex)
	default:
		return errBadVersion
	}
}

func triggerPreferredAssetSwapV3_0_0(ctx cosmos.Context, mgr Manager, tn THORName, affcol AffiliateFeeCollector, queueIndex int) error {
	// Check that the THORName has an address alias for the PreferredAsset, if not skip
	// the swap
	alias := tn.GetAlias(tn.PreferredAsset.GetChain())
	if alias.Equals(common.NoAddress) {
		return fmt.Errorf("no alias for preferred asset, skip preferred asset swap: %s", tn.Name)
	}

	// Sanity check: don't swap 0 amount
	if affcol.RuneAmount.IsZero() {
		return fmt.Errorf("can't execute preferred asset swap, accrued RUNE amount is zero")
	}
	// Sanity check: ensure the swap amount isn't more than the entire AffiliateCollector module
	acBalance := mgr.Keeper().GetRuneBalanceOfModule(ctx, AffiliateCollectorName)
	if affcol.RuneAmount.GT(acBalance) {
		return fmt.Errorf("rune amount greater than module balance: (%s/%s)", affcol.RuneAmount.String(), acBalance.String())
	}

	affRune := affcol.RuneAmount
	affCoin := common.NewCoin(common.RuneAsset(), affRune)

	networkMemo := fmt.Sprintf("%s-%s", PreferredAssetSwapMemoPrefix, tn.Name)
	asgardAddress, err := mgr.Keeper().GetModuleAddress(AsgardName)
	if err != nil {
		ctx.Logger().Error("failed to retrieve asgard address", "error", err)
		return err
	}
	affColAddress, err := mgr.Keeper().GetModuleAddress(AffiliateCollectorName)
	if err != nil {
		ctx.Logger().Error("failed to retrieve affiliate collector module address", "error", err)
		return err
	}

	ctx.Logger().Debug("trigger preferred asset swap", "thorname", tn.Name, "amt", affRune.String(), "dest", alias.String(), "asset", tn.PreferredAsset.String())

	// Generate a unique ID for the preferred asset swap, which is a hash of the THORName,
	// affCoin, and BlockHeight This is to prevent the network thinking it's an outbound
	// of the swap that triggered it
	str := fmt.Sprintf("%s|%s|%d", tn.GetName(), affCoin.String(), ctx.BlockHeight())
	hash := fmt.Sprintf("%X", sha256.Sum256([]byte(str)))

	ctx.Logger().Info("preferred asset swap hash", "hash", hash, "thorname", tn.Name)

	paTxID, err := common.NewTxID(hash)
	if err != nil {
		return err
	}

	existingVoter, err := mgr.Keeper().GetObservedTxInVoter(ctx, paTxID)
	if err != nil {
		return fmt.Errorf("fail to get existing voter: %w", err)
	}
	if len(existingVoter.Txs) > 0 {
		return fmt.Errorf("preferred asset tx: %s already exists", str)
	}

	// Construct preferred asset swap tx
	tx := common.NewTx(
		paTxID,
		affColAddress,
		asgardAddress,
		common.NewCoins(affCoin),
		common.Gas{},
		networkMemo,
	)

	preferredAssetSwap := NewMsgSwap(
		tx,
		tn.PreferredAsset,
		alias,
		cosmos.ZeroUint(),
		common.NoAddress,
		cosmos.ZeroUint(),
		"",
		"", nil,
		MarketSwap,
		0, 0,
		tn.Owner,
	)

	// Construct preferred asset swap inbound tx voter
	txIn := ObservedTx{Tx: tx}
	txInVoter := NewObservedTxVoter(txIn.Tx.ID, []common.ObservedTx{txIn})
	txInVoter.Height = ctx.BlockHeight()
	txInVoter.FinalisedHeight = ctx.BlockHeight()
	txInVoter.Tx = txIn
	mgr.Keeper().SetObservedTxInVoter(ctx, txInVoter)

	// Queue the preferred asset swap
	if err = mgr.Keeper().SetSwapQueueItem(ctx, *preferredAssetSwap, queueIndex); err != nil {
		ctx.Logger().Error("fail to add preferred asset swap to queue", "error", err)
		return err
	}

	// Send RUNE from AffiliateCollector to Asgard and update AffiliateCollector
	if err = mgr.Keeper().SendFromModuleToModule(ctx, AffiliateCollectorName, AsgardName, common.NewCoins(affCoin)); err != nil {
		return fmt.Errorf("failed to send rune to asgard: %w", err)
	}

	affcol.RuneAmount = cosmos.ZeroUint()
	mgr.Keeper().SetAffiliateCollector(ctx, affcol)

	return nil
}

// skimAffiliateFee - attempts to distribute a fee to each affiliate in the memo,
// skimmed from coin. Returns the total fee distributed priced in coin.Asset.
// Logic:
//  1. Parse the memo to get the affiliate fee and the memo
//  2. For each affiliate
//     - If coin is RUNE transfer to the affiliate
//     - If coin is not RUNE, swap the coin to RUNE and transfer to the affiliate
//     - If affiliate is a thorname and has a preferred asset, send RUNE to the affiliate collector
func skimAffiliateFees(ctx cosmos.Context, mgr Manager, mainTx common.Tx, signer cosmos.AccAddress, coin common.Coin, memoStr string) (cosmos.Uint, error) {
	version := mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return skimAffiliateFeesV3_0_0(ctx, mgr, mainTx, signer, coin, memoStr)
	default:
		return cosmos.ZeroUint(), errBadVersion
	}
}

func skimAffiliateFeesV3_0_0(ctx cosmos.Context, mgr Manager, mainTx common.Tx, signer cosmos.AccAddress, coin common.Coin, memoStr string) (cosmos.Uint, error) {
	// sanity checks
	if mainTx.IsEmpty() {
		return cosmos.ZeroUint(), fmt.Errorf("main tx is empty")
	}
	if coin.IsEmpty() {
		return cosmos.ZeroUint(), fmt.Errorf("coin is empty")
	}

	// Parse memo
	memo, err := ParseMemoWithTHORNames(ctx, mgr.Keeper(), memoStr)
	if err != nil {
		ctx.Logger().Error("fail to parse swap memo", "memo", memoStr, "error", err)
		return cosmos.ZeroUint(), err
	}
	affiliates := memo.GetAffiliates()
	affiliatesBps := memo.GetAffiliatesBasisPoints()
	if len(affiliates) == 0 || len(affiliatesBps) == 0 {
		return cosmos.ZeroUint(), nil
	}

	var feeEvents []*EventAffiliateFee // fee events to emit
	totalFee := cosmos.ZeroUint()      // total fee distributed
	swapIndex := 1                     // swap index, start at 1 to account for the main user swap

	// Iterate through each affiliate and attempt to distribute the fee
	for i, affiliate := range affiliates {
		ctx.Logger().Info("distributing affiliate fee", "txid", mainTx.ID.String(), "affiliate", affiliate, "fee", affiliatesBps[i].String(), "asset", coin.Asset, "amount", coin.Amount)
		// Determine if affiliate is address or thorname. If it's an address it must be a RUNE address.
		var runeAddr cosmos.AccAddress
		var thorname *THORName
		tnString := ""

		// Fetch thorname + RUNE alias for THORChain
		if mgr.Keeper().THORNameExists(ctx, affiliate) {
			tn, errTn := mgr.Keeper().GetTHORName(ctx, affiliate)
			if errTn != nil {
				ctx.Logger().Error("fail to get thorname, skipping fee", "err", err)
				continue
			}
			thorname = &tn

			// If affiliate is thorname, check if it can receive an affiliate fee. If not, skip.
			if !thorname.CanReceiveAffiliateFee() {
				ctx.Logger().Info("affiliate cannot receive affiliate fee", "affiliate", affiliate)
				continue
			}
			addr := thorname.GetAlias(common.THORChain)
			if !addr.IsEmpty() {
				runeAddr, err = addr.AccAddress()
				if err != nil {
					ctx.Logger().Error("fail to convert address into AccAddress, skipping fee", "msg", addr, "error", err)
					continue
				}
			} else {
				// If this is reached, the thorname has a preferred asset + no alias for RUNE,
				// so set swap destination to thorname owner
				runeAddr = thorname.Owner
			}
		} else {
			addr, errAddr := common.NewAddress(affiliate)
			if errAddr != nil {
				ctx.Logger().Error("fail to parse affiliate address, skipping fee", "msg", affiliate, "error", err)
				continue
			}
			if !addr.GetChain().IsTHORChain() {
				ctx.Logger().Error("affiliate address is not THORChain, skipping fee", "msg", affiliate)
				continue
			}
			runeAddr, err = addr.AccAddress()
			if err != nil {
				ctx.Logger().Error("fail to convert address into AccAddress, skipping fee", "msg", addr, "error", err)
				continue
			}
		}

		feeBps := affiliatesBps[i]
		if !feeBps.IsZero() {
			affAmt := common.GetSafeShare(
				feeBps,
				cosmos.NewUint(constants.MaxBasisPts),
				coin.Amount,
			)
			affCoin := common.NewCoin(coin.Asset, affAmt)

			// Distribute fee to affiliate
			if coin.Asset.IsRune() {
				// Transfer to RUNE address or affiliate collector module
				if thorname != nil && !thorname.PreferredAsset.IsEmpty() {
					// Send RUNE to the affiliate collector and update the account
					err = addRuneToAffiliateCollector(ctx, mgr, affCoin, thorname, &swapIndex)
					if err != nil {
						ctx.Logger().Error("fail to update affiliate collector", "error", err)
						continue
					}
				} else {
					// Send RUNE to the affiliate address
					err = mgr.Keeper().SendFromModuleToAccount(ctx, AsgardName, runeAddr, common.NewCoins(affCoin))
					if err != nil {
						ctx.Logger().Error("fail to send rune to affiliate", "affiliate", affiliate, "error", err)
						continue
					}
				}
			} else {
				// Swap to RUNE and transfer to RUNE address or affiliate collector module
				err := affiliateSwapToRune(ctx, mgr, mainTx, signer, affAmt, runeAddr, memo, thorname, &swapIndex)
				if err != nil {
					ctx.Logger().Error("fail to swap to rune", "affiliate", affiliate, "error", err)
					continue
				}
			}
			if thorname != nil && thorname.Name != "" {
				tnString = thorname.Name
			}

			// add event
			feeEvent := NewEventAffiliateFee(
				mainTx.ID,
				mainTx.Memo,
				tnString,
				common.Address(runeAddr.String()),
				coin.Asset,
				feeBps.Uint64(),
				coin.Amount,
				affCoin.Amount)

			feeEvents = append(feeEvents, feeEvent)
			totalFee = totalFee.Add(affAmt)
		}
	}

	// Emit affiliate fee events
	for _, event := range feeEvents {
		if err := mgr.EventMgr().EmitEvent(ctx, event); err != nil {
			ctx.Logger().Error("fail to emit affiliate fee event", "error", err)
		}
	}

	return totalFee, nil
}

func affiliateSwapToRune(ctx cosmos.Context, mgr Manager, mainTx common.Tx, signer cosmos.AccAddress, affAmt cosmos.Uint, affAcc cosmos.AccAddress, memo thorchain.Memo, tn *THORName, swapIndex *int) error {
	version := mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return affiliateSwapToRuneV3_0_0(ctx, mgr, mainTx, signer, affAmt, affAcc, memo, tn, swapIndex)
	default:
		return errBadVersion
	}
}

func affiliateSwapToRuneV3_0_0(ctx cosmos.Context, mgr Manager, mainTx common.Tx, signer cosmos.AccAddress, affAmt cosmos.Uint, affAcc cosmos.AccAddress, memo thorchain.Memo, tn *THORName, swapIndex *int) error {
	affAddr, err := common.NewAddress(affAcc.String())
	if err != nil {
		return fmt.Errorf("fail to parse affiliate address: %w", err)
	}

	// Copy mainTx coins so as not to modify the original
	mainTx.Coins = mainTx.Coins.Copy()

	// Update memo to include only this affiliate
	tnMemo := affAddr.String()
	if tn != nil {
		tnMemo = tn.Name
	}
	memoStr := NewSwapMemo(ctx, mgr, common.RuneAsset(), affAddr, cosmos.ZeroUint(), tnMemo, cosmos.ZeroUint())
	mainTx.Memo = memoStr

	affiliateSwap := NewMsgSwap(
		mainTx,
		common.RuneAsset(),
		affAddr,
		cosmos.ZeroUint(),
		common.NoAddress,
		cosmos.ZeroUint(),
		"",
		"", nil,
		MarketSwap,
		0, 0,
		signer,
	)

	// check if swap will succeed, if not, skip
	willSucceed := willSwapOutputExceedLimitAndFees(ctx, mgr, *affiliateSwap)
	if !willSucceed {
		return fmt.Errorf("swap will not succeed, skipping affiliate swap")
	}

	// PreferredAsset set, swap to the AffiliateCollector Module + check if the
	// preferred asset swap should be triggered
	if tn != nil && !tn.Owner.Empty() && !tn.PreferredAsset.IsEmpty() {
		var affCol AffiliateFeeCollector
		affCol, err = mgr.Keeper().GetAffiliateCollector(ctx, tn.Owner)
		if err != nil {
			return fmt.Errorf("failed to get affiliate collector for thorname: %w", err)
		}

		var affColAddress common.Address
		affColAddress, err = mgr.Keeper().GetModuleAddress(AffiliateCollectorName)
		if err != nil {
			return fmt.Errorf("failed to retrieve the affiliate collector module address: %w", err)
		}

		// Set AffiliateCollector Module as destination and populate the AffiliateAddress
		// so that the swap handler can increment the emitted RUNE for the affiliate in
		// the AffiliateCollector KVStore.
		affiliateSwap.Destination = affColAddress
		affiliateSwap.AffiliateAddress = affAddr

		// Check if accrued RUNE is 100x current outbound fee of preferred asset chain, if
		// so trigger the preferred asset swap
		ofRune, err := mgr.GasMgr().GetAssetOutboundFee(ctx, tn.PreferredAsset, true)
		if err != nil {
			ctx.Logger().Error("failed to get outbound fee for preferred asset, skipping preferred asset swap", "name", tn.Name, "asset", tn.PreferredAsset, "error", err)
		}
		multiplier := mgr.Keeper().GetConfigInt64(ctx, constants.PreferredAssetOutboundFeeMultiplier)
		threshold := ofRune.Mul(cosmos.NewUint(uint64(multiplier)))

		ctx.Logger().Info("check preferred asset threshold", "threshold", threshold.String(), "accrued", affCol.RuneAmount.String())

		if err == nil && affCol.RuneAmount.GT(threshold) {
			*swapIndex++
			if err = triggerPreferredAssetSwap(ctx, mgr, common.NoAddress, "", *tn, affCol, *swapIndex); err != nil {
				ctx.Logger().Error("fail to swap to preferred asset", "thorname", tn.Name, "err", err)
			}
		}
	}

	if affiliateSwap.Tx.Coins[0].Amount.GTE(affAmt) {
		affiliateSwap.Tx.Coins[0].Amount = affAmt
	}

	*swapIndex++
	if err = mgr.Keeper().SetSwapQueueItem(ctx, *affiliateSwap, *swapIndex); err != nil {
		return fmt.Errorf("fail to add swap to queue: %w", err)
	}

	return nil
}

// addRuneToAffiliateCollector - accrue RUNE in the AffiliateCollector module and check if
// a PreferredAsset swap should be triggered. Returns an error if the fee distribution fails.
func addRuneToAffiliateCollector(ctx cosmos.Context, mgr Manager, coin common.Coin, thorname *THORName, swapIndex *int) error {
	version := mgr.GetVersion()
	switch {
	case version.GTE(semver.MustParse("3.0.0")):
		return updateAffiliateCollectorV3_0_0(ctx, mgr, coin, thorname, swapIndex)
	default:
		return fmt.Errorf("unsupported version: %s", version)
	}
}

// updateAffiliateCollector - accrue RUNE in the AffiliateCollector module and check if
// a PreferredAsset swap should be triggered. Returns an error if the fee distribution fails.
func updateAffiliateCollectorV3_0_0(ctx cosmos.Context, mgr Manager, coin common.Coin, thorname *THORName, swapIndex *int) error {
	affcol, err := mgr.Keeper().GetAffiliateCollector(ctx, thorname.Owner)
	if err != nil {
		return fmt.Errorf("failed to get affiliate collector: %w", err)
	} else {
		if err = mgr.Keeper().SendFromModuleToModule(ctx, AsgardName, AffiliateCollectorName, common.NewCoins(coin)); err != nil {
			return fmt.Errorf("failed to send coin to affiliate collector: %w", err)
		} else {
			affcol.RuneAmount = affcol.RuneAmount.Add(coin.Amount)
			mgr.Keeper().SetAffiliateCollector(ctx, affcol)
		}
	}

	// Check if balance exceeds threshold for preferred asset swap. Don't return error if preferred asset swap fails.
	ofRune, err := mgr.GasMgr().GetAssetOutboundFee(ctx, thorname.PreferredAsset, true)
	if err != nil {
		ctx.Logger().Error("failed to get outbound fee for preferred asset, skipping preferred asset swap", "name", thorname.Name, "asset", thorname.PreferredAsset, "error", err)
		return nil
	}

	multiplier := mgr.Keeper().GetConfigInt64(ctx, constants.PreferredAssetOutboundFeeMultiplier)
	threshold := ofRune.Mul(cosmos.NewUint(uint64(multiplier)))
	if affcol.RuneAmount.GT(threshold) {
		*swapIndex++
		if err = triggerPreferredAssetSwap(ctx, mgr, common.NoAddress, "", *thorname, affcol, *swapIndex); err != nil {
			ctx.Logger().Error("fail to swap to preferred asset", "thorname", thorname.Name, "err", err)
		}
	}
	return nil
}
