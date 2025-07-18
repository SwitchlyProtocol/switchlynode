package keeperv1

import (
	"fmt"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/constants"
)

func (k KVStore) IsTradingHalt(ctx cosmos.Context, msg cosmos.Msg) bool {
	// consider halted if ragnarok in progress for either asset or chain gas asset
	// gather source and target assets
	checkAssets := []common.Asset{}
	switch m := msg.(type) {
	case *MsgSwap:
		// regardless ragnarok, synth to equivalent layer1 asset is allowed
		source := m.Tx.Coins[0].Asset
		if !(source.IsSyntheticAsset() && m.TargetAsset.Equals(source.GetLayer1Asset())) {
			checkAssets = []common.Asset{m.Tx.Coins[0].Asset, m.TargetAsset}
		}

	case *MsgAddLiquidity:
		checkAssets = []common.Asset{m.Asset}
	}

	if k.IsRagnarok(ctx, checkAssets) {
		return true
	}

	switch m := msg.(type) {
	case *MsgSwap:
		source := common.EmptyChain
		sourceAsset := m.Tx.Coins[0].Asset
		if len(m.Tx.Coins) > 0 {
			source = m.Tx.Coins[0].Asset.GetLayer1Asset().Chain
		}
		target := m.TargetAsset.GetLayer1Asset().Chain

		if sourceAsset.IsTCY() {
			return k.IsTCYTradingHalted(ctx) || k.IsChainTradingHalted(ctx, target)
		} else if m.TargetAsset.IsTCY() {
			return k.IsTCYTradingHalted(ctx) || k.IsChainTradingHalted(ctx, source)
		}

		return k.IsChainTradingHalted(ctx, source) || k.IsChainTradingHalted(ctx, target) || k.IsGlobalTradingHalted(ctx)
	case *MsgAddLiquidity:
		if m.Asset.IsTCY() {
			return k.IsTCYTradingHalted(ctx)
		}
		return k.IsChainTradingHalted(ctx, m.Asset.Chain) || k.IsGlobalTradingHalted(ctx)
	default:
		return k.IsGlobalTradingHalted(ctx)
	}
}

func (k KVStore) IsGlobalTradingHalted(ctx cosmos.Context) bool {
	haltTrading, err := k.GetMimir(ctx, "HaltTrading")
	if err == nil && ((haltTrading > 0 && haltTrading < ctx.BlockHeight()) || k.RagnarokInProgress(ctx)) {
		return true
	}
	return false
}

func (k KVStore) IsChainTradingHalted(ctx cosmos.Context, chain common.Chain) bool {
	mimirKey := fmt.Sprintf("Halt%sTrading", chain)
	haltChainTrading, err := k.GetMimir(ctx, mimirKey)
	if err == nil && (haltChainTrading > 0 && haltChainTrading < ctx.BlockHeight()) {
		ctx.Logger().Debug("trading is halt", "chain", chain)
		return true
	}
	// further to check whether the chain is halted
	return k.IsChainHalted(ctx, chain)
}

func (k KVStore) IsChainHalted(ctx cosmos.Context, chain common.Chain) bool {
	haltChain, err := k.GetMimir(ctx, "HaltChainGlobal")
	if err == nil && (haltChain > 0 && haltChain < ctx.BlockHeight()) {
		ctx.Logger().Debug("global is halt")
		return true
	}

	haltChain, err = k.GetMimir(ctx, "NodePauseChainGlobal")
	if err == nil && haltChain > ctx.BlockHeight() {
		ctx.Logger().Debug("node global is halt")
		return true
	}

	haltMimirKey := fmt.Sprintf("Halt%sChain", chain)
	haltChain, err = k.GetMimir(ctx, haltMimirKey)
	if err == nil && (haltChain > 0 && haltChain < ctx.BlockHeight()) {
		ctx.Logger().Debug("chain is halt via admin or double-spend check", "chain", chain)
		return true
	}

	solvencyHaltMimirKey := fmt.Sprintf("SolvencyHalt%sChain", chain)
	haltChain, err = k.GetMimir(ctx, solvencyHaltMimirKey)
	if err == nil && (haltChain > 0 && haltChain < ctx.BlockHeight()) {
		ctx.Logger().Debug("chain is halt via solvency check", "chain", chain)
		return true
	}
	return false
}

func (k KVStore) IsLPPaused(ctx cosmos.Context, chain common.Chain) bool {
	// check if global LP is paused
	pauseLPGlobal, err := k.GetMimir(ctx, "PauseLP")
	if err == nil && pauseLPGlobal > 0 && pauseLPGlobal < ctx.BlockHeight() {
		return true
	}

	pauseLP, err := k.GetMimir(ctx, fmt.Sprintf("PauseLP%s", chain))
	if err == nil && pauseLP > 0 && pauseLP < ctx.BlockHeight() {
		ctx.Logger().Debug("chain has paused LP actions", "chain", chain)
		return true
	}
	return false
}

func (k KVStore) IsPoolDepositPaused(ctx cosmos.Context, asset common.Asset) bool {
	// check if deposits into pool are paused
	v, err := k.GetMimirWithRef(ctx, constants.MimirTemplatePauseLPDeposit, asset.MimirString())
	if err == nil && v > 0 {
		return true
	}
	return false
}

func (k KVStore) IsTCYTradingHalted(ctx cosmos.Context) bool {
	haltTCYTrading, err := k.GetMimir(ctx, "HaltTCYTrading")
	if err == nil && (haltTCYTrading > 0 && haltTCYTrading < ctx.BlockHeight()) {
		ctx.Logger().Debug("TCY trading is halt")
		return true
	}

	return k.IsGlobalTradingHalted(ctx) || k.IsChainHalted(ctx, common.SWITCHLYChain)
}
