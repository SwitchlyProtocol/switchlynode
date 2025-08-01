package thorchain

import (
	"fmt"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/common/tokenlist"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

// MsgHandler is an interface expect all handler to implement
type MsgHandler interface {
	Run(ctx cosmos.Context, msg cosmos.Msg) (*cosmos.Result, error)
}

// NewInternalHandler returns a handler for "thorchain" internal type messages.
func NewInternalHandler(mgr Manager) cosmos.Handler {
	return func(ctx cosmos.Context, msg cosmos.Msg) (*cosmos.Result, error) {
		handlerMap := getInternalHandlerMapping(mgr)
		h, ok := handlerMap[sdk.MsgTypeURL(msg)]
		if !ok {
			errMsg := fmt.Sprintf("Unrecognized thorchain Msg type: %v", sdk.MsgTypeURL(msg))
			return nil, cosmos.ErrUnknownRequest(errMsg)
		}

		// CacheContext() returns a context which caches all changes and only forwards
		// to the underlying context when commit() is called. Call commit() only when
		// the handler succeeds, otherwise return error and the changes will be discarded.
		// On commit, cached events also have to be explicitly emitted.
		cacheCtx, commit := ctx.CacheContext()
		res, err := h.Run(cacheCtx, msg)
		if err == nil {
			// Success, commit the cached changes and events
			commit()
		}

		return res, err
	}
}

func getInternalHandlerMapping(mgr Manager) map[string]MsgHandler {
	// New arch handlers
	m := make(map[string]MsgHandler)
	m[sdk.MsgTypeURL(&MsgOutboundTx{})] = NewOutboundTxHandler(mgr)
	m[sdk.MsgTypeURL(&MsgSwap{})] = NewSwapHandler(mgr)
	m[sdk.MsgTypeURL(&MsgReserveContributor{})] = NewReserveContributorHandler(mgr)
	m[sdk.MsgTypeURL(&MsgBond{})] = NewBondHandler(mgr)
	m[sdk.MsgTypeURL(&MsgUnBond{})] = NewUnBondHandler(mgr)
	m[sdk.MsgTypeURL(&MsgLeave{})] = NewLeaveHandler(mgr)
	m[sdk.MsgTypeURL(&MsgMaint{})] = NewMaintHandler(mgr)
	m[sdk.MsgTypeURL(&MsgDonate{})] = NewDonateHandler(mgr)
	m[sdk.MsgTypeURL(&MsgWithdrawLiquidity{})] = NewWithdrawLiquidityHandler(mgr)
	m[sdk.MsgTypeURL(&MsgAddLiquidity{})] = NewAddLiquidityHandler(mgr)
	m[sdk.MsgTypeURL(&MsgRefundTx{})] = NewRefundHandler(mgr)
	m[sdk.MsgTypeURL(&MsgMigrate{})] = NewMigrateHandler(mgr)
	m[sdk.MsgTypeURL(&MsgRagnarok{})] = NewRagnarokHandler(mgr)
	m[sdk.MsgTypeURL(&MsgNoOp{})] = NewNoOpHandler(mgr)
	m[sdk.MsgTypeURL(&MsgConsolidate{})] = NewConsolidateHandler(mgr)
	m[sdk.MsgTypeURL(&MsgManageTHORName{})] = NewManageTHORNameHandler(mgr)
	m[sdk.MsgTypeURL(&MsgLoanOpen{})] = NewLoanOpenHandler(mgr)
	m[sdk.MsgTypeURL(&MsgLoanRepayment{})] = NewLoanRepaymentHandler(mgr)
	m[sdk.MsgTypeURL(&MsgTradeAccountDeposit{})] = NewTradeAccountDepositHandler(mgr)
	m[sdk.MsgTypeURL(&MsgTradeAccountWithdrawal{})] = NewTradeAccountWithdrawalHandler(mgr)
	m[sdk.MsgTypeURL(&MsgSecuredAssetDeposit{})] = NewSecuredAssetDepositHandler(mgr)
	m[sdk.MsgTypeURL(&MsgSecuredAssetWithdraw{})] = NewSecuredAssetWithdrawHandler(mgr)
	m[sdk.MsgTypeURL(&MsgRunePoolDeposit{})] = NewRunePoolDepositHandler(mgr)
	m[sdk.MsgTypeURL(&MsgRunePoolWithdraw{})] = NewRunePoolWithdrawHandler(mgr)
	m[sdk.MsgTypeURL(&MsgWasmExec{})] = NewWasmExecHandler(mgr)
	m[sdk.MsgTypeURL(&MsgSwitch{})] = NewSwitchHandler(mgr)
	m[sdk.MsgTypeURL(&MsgTCYClaim{})] = NewTCYClaimHandler(mgr)
	m[sdk.MsgTypeURL(&MsgTCYStake{})] = NewTCYStakeHandler(mgr)
	m[sdk.MsgTypeURL(&MsgTCYUnstake{})] = NewTCYUnstakeHandler(mgr)
	return m
}

func getMsgSwapFromMemo(memo SwapMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	if memo.Destination.IsEmpty() {
		memo.Destination = tx.Tx.FromAddress
	}
	return NewMsgSwap(tx.Tx, memo.GetAsset(), memo.Destination, memo.SlipLimit, memo.AffiliateAddress, memo.AffiliateBasisPoints, memo.GetDexAggregator(), memo.GetDexTargetAddress(), memo.GetDexTargetLimit(), memo.GetSwapType(), memo.GetStreamQuantity(), memo.GetStreamInterval(), signer), nil
}

func getMsgWithdrawFromMemo(memo WithdrawLiquidityMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	withdrawAmount := cosmos.NewUint(MaxWithdrawBasisPoints)
	if !memo.GetAmount().IsZero() {
		withdrawAmount = memo.GetAmount()
	}
	return NewMsgWithdrawLiquidity(tx.Tx, tx.Tx.FromAddress, withdrawAmount, memo.GetAsset(), memo.GetWithdrawalAsset(), signer), nil
}

func getMsgAddLiquidityFromMemo(ctx cosmos.Context, memo AddLiquidityMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	// Extract the Rune amount and the asset amount from the transaction. At least one of them must be
	// nonzero. If THORNode saw two types of coins, one of them must be the asset coin.
	runeCoin := tx.Tx.Coins.GetCoin(common.SwitchNative)
	assetCoin := tx.Tx.Coins.GetCoin(memo.GetAsset())

	var runeAddr common.Address
	var assetAddr common.Address
	if tx.Tx.Chain.Equals(common.SWITCHLYChain) {
		runeAddr = tx.Tx.FromAddress
		assetAddr = memo.GetDestination()
	} else {
		runeAddr = memo.GetDestination()
		assetAddr = tx.Tx.FromAddress
	}
	// in case we are providing native rune and another native asset
	if memo.GetAsset().Chain.Equals(common.SWITCHLYChain) {
		assetAddr = runeAddr
	}

	return NewMsgAddLiquidity(tx.Tx, memo.GetAsset(), runeCoin.Amount, assetCoin.Amount, runeAddr, assetAddr, memo.AffiliateAddress, memo.AffiliateBasisPoints, signer), nil
}

func getMsgDonateFromMemo(memo DonateMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	runeCoin := tx.Tx.Coins.GetCoin(common.SwitchNative)
	assetCoin := tx.Tx.Coins.GetCoin(memo.GetAsset())
	return NewMsgDonate(tx.Tx, memo.GetAsset(), runeCoin.Amount, assetCoin.Amount, signer), nil
}

func getMsgRefundFromMemo(memo RefundMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgRefundTx(tx, memo.GetTxID(), signer), nil
}

func getMsgOutboundFromMemo(memo OutboundMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgOutboundTx(tx, memo.GetTxID(), signer), nil
}

func getMsgMigrateFromMemo(memo MigrateMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgMigrate(tx, memo.GetBlockHeight(), signer), nil
}

func getMsgRagnarokFromMemo(memo RagnarokMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgRagnarok(tx, memo.GetBlockHeight(), signer), nil
}

func getMsgLeaveFromMemo(memo LeaveMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgLeave(tx.Tx, memo.GetAccAddress(), signer), nil
}

func getMsgLoanOpenFromMemo(ctx cosmos.Context, keeper keeper.Keeper, memo LoanOpenMemo, tx ObservedTx, signer cosmos.AccAddress, txid common.TxID) (cosmos.Msg, error) {
	memo.TargetAsset = fuzzyAssetMatch(ctx, keeper, memo.TargetAsset)
	return NewMsgLoanOpen(tx.Tx.FromAddress, tx.Tx.Coins[0].Asset, tx.Tx.Coins[0].Amount, memo.TargetAddress, memo.TargetAsset, memo.GetMinOut(), memo.GetAffiliateAddress(), memo.GetAffiliateBasisPoints(), memo.GetDexAggregator(), memo.GetDexTargetAddress(), memo.DexTargetLimit, signer, txid), nil
}

func getMsgLoanRepaymentFromMemo(memo LoanRepaymentMemo, from common.Address, coin common.Coin, signer cosmos.AccAddress, txid common.TxID) (cosmos.Msg, error) {
	return NewMsgLoanRepayment(memo.Owner, memo.Asset, memo.MinOut, from, coin, signer, txid), nil
}

func getMsgBondFromMemo(memo BondMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	coin := tx.Tx.Coins.GetCoin(common.SwitchNative)
	return NewMsgBond(tx.Tx, memo.GetAccAddress(), coin.Amount, tx.Tx.FromAddress, memo.BondProviderAddress, signer, memo.NodeOperatorFee), nil
}

func getMsgUnbondFromMemo(memo UnbondMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return NewMsgUnBond(tx.Tx, memo.GetAccAddress(), memo.GetAmount(), tx.Tx.FromAddress, memo.BondProviderAddress, signer), nil
}

func getMsgMaintFromMemo(memo MaintMemo, signer cosmos.AccAddress) (cosmos.Msg, error) {
	return types.NewMsgMaint(memo.GetAccAddress(), signer), nil
}

func getMsgManageTHORNameFromMemo(memo ManageTHORNameMemo, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	if len(tx.Tx.Coins) == 0 {
		return nil, fmt.Errorf("transaction must have rune in it")
	}
	return NewMsgManageTHORName(memo.Name, memo.Chain, memo.Address, tx.Tx.Coins[0], memo.Expire, memo.PreferredAsset, memo.Owner, signer), nil
}

func processOneTxIn(ctx cosmos.Context, keeper keeper.Keeper, tx ObservedTx, signer cosmos.AccAddress) (cosmos.Msg, error) {
	if len(tx.Tx.Coins) != 1 {
		return nil, cosmos.ErrInvalidCoins("only send 1 coins per message")
	}

	memo, err := ParseMemoWithTHORNames(ctx, keeper, tx.Tx.Memo)
	if err != nil {
		ctx.Logger().Error("fail to parse memo", "error", err)
		return nil, err
	}

	// THORNode should not have one tx across chain, if it is cross chain it should be separate tx
	var newMsg cosmos.Msg
	// interpret the memo and initialize a corresponding msg event
	switch m := memo.(type) {
	case AddLiquidityMemo:
		m.Asset = fuzzyAssetMatch(ctx, keeper, m.Asset)
		newMsg, err = getMsgAddLiquidityFromMemo(ctx, m, tx, signer)
	case WithdrawLiquidityMemo:
		m.Asset = fuzzyAssetMatch(ctx, keeper, m.Asset)
		if !m.WithdrawalAsset.IsEmpty() {
			m.WithdrawalAsset = fuzzyAssetMatch(ctx, keeper, m.WithdrawalAsset)
		}
		newMsg, err = getMsgWithdrawFromMemo(m, tx, signer)
	case SwapMemo:
		m.Asset = fuzzyAssetMatch(ctx, keeper, m.Asset)
		m.DexTargetAddress = externalAssetMatch(m.Asset.GetChain(), m.DexTargetAddress)
		newMsg, err = getMsgSwapFromMemo(m, tx, signer)
	case DonateMemo:
		m.Asset = fuzzyAssetMatch(ctx, keeper, m.Asset)
		newMsg, err = getMsgDonateFromMemo(m, tx, signer)
	case RefundMemo:
		newMsg, err = getMsgRefundFromMemo(m, tx, signer)
	case OutboundMemo:
		newMsg, err = getMsgOutboundFromMemo(m, tx, signer)
	case MigrateMemo:
		newMsg, err = getMsgMigrateFromMemo(m, tx, signer)
	case BondMemo:
		newMsg, err = getMsgBondFromMemo(m, tx, signer)
	case UnbondMemo:
		newMsg, err = getMsgUnbondFromMemo(m, tx, signer)
	case RagnarokMemo:
		newMsg, err = getMsgRagnarokFromMemo(m, tx, signer)
	case LeaveMemo:
		newMsg, err = getMsgLeaveFromMemo(m, tx, signer)
	case ReserveMemo:
		res := NewReserveContributor(tx.Tx.FromAddress, tx.Tx.Coins.GetCoin(common.SwitchNative).Amount)
		newMsg = NewMsgReserveContributor(tx.Tx, res, signer)
	case NoOpMemo:
		newMsg = NewMsgNoOp(tx, signer, m.Action)
	case ConsolidateMemo:
		newMsg = NewMsgConsolidate(tx, signer)
	case ManageTHORNameMemo:
		newMsg, err = getMsgManageTHORNameFromMemo(m, tx, signer)
	case LoanOpenMemo:
		newMsg, err = getMsgLoanOpenFromMemo(ctx, keeper, m, tx, signer, tx.Tx.ID)
	case LoanRepaymentMemo:
		m.Asset = fuzzyAssetMatch(ctx, keeper, m.Asset)
		from := tx.Tx.FromAddress
		newMsg, err = getMsgLoanRepaymentFromMemo(m, from, tx.Tx.Coins[0], signer, tx.Tx.ID)
	case TradeAccountDepositMemo:
		coin := tx.Tx.Coins[0]
		newMsg = NewMsgTradeAccountDeposit(coin.Asset, coin.Amount, m.GetAccAddress(), signer, tx.Tx)
	case TradeAccountWithdrawalMemo:
		coin := tx.Tx.Coins[0]
		newMsg = NewMsgTradeAccountWithdrawal(coin.Asset, coin.Amount, m.GetAddress(), signer, tx.Tx)
	case SecuredAssetDepositMemo:
		coin := tx.Tx.Coins[0]
		newMsg = NewMsgSecuredAssetDeposit(coin.Asset, coin.Amount, m.GetAccAddress(), signer, tx.Tx)
	case SecuredAssetWithdrawMemo:
		coin := tx.Tx.Coins[0]
		newMsg = NewMsgSecuredAssetWithdraw(coin.Asset, coin.Amount, m.GetAddress(), signer, tx.Tx)
	case RunePoolDepositMemo:
		newMsg = NewMsgRunePoolDeposit(signer, tx.Tx)
	case RunePoolWithdrawMemo:
		newMsg = NewMsgRunePoolWithdraw(signer, tx.Tx, m.GetBasisPts(), m.GetAffiliateAddress(), m.GetAffiliateBasisPoints())
	case ExecMemo:
		coin := tx.Tx.Coins[0]
		sender, err := tx.Tx.FromAddress.MappedAccAddress()
		if err != nil {
			return nil, err
		}
		newMsg = NewMsgWasmExec(coin.Asset, coin.Amount, m.ContractAddress, sender, signer, m.Msg, tx.Tx)
	case SwitchMemo:
		coin := tx.Tx.Coins[0]
		newMsg = NewMsgSwitch(coin.Asset, coin.Amount, m.GetAccAddress(), signer, tx.Tx)
	case TCYClaimMemo:
		newMsg = NewMsgTCYClaim(m.Address, tx.Tx.FromAddress, signer)
	case TCYStakeMemo:
		newMsg = NewMsgTCYStake(tx.Tx, signer)
	case TCYUnstakeMemo:
		newMsg = NewMsgTCYUnstake(tx.Tx, m.BasisPoints, signer)
	case MaintMemo:
		newMsg, err = getMsgMaintFromMemo(m, signer)
	default:
		return nil, errInvalidMemo
	}

	if err != nil {
		return newMsg, err
	}

	newMsgV, ok := newMsg.(sdk.HasValidateBasic)
	if !ok {
		return newMsg, fmt.Errorf("msg does not implement sdk.HasValidateBasic: %T", newMsg)
	}

	return newMsg, newMsgV.ValidateBasic()
}

func fuzzyAssetMatch(ctx cosmos.Context, keeper keeper.Keeper, origAsset common.Asset) common.Asset {
	asset := origAsset.GetLayer1Asset()
	// if it's already an exact match with successfully-added liquidity, return it immediately
	pool, err := keeper.GetPool(ctx, asset)
	if err != nil {
		return origAsset
	}
	// Only check BalanceRune after checking the error so that no panic if there were an error.
	if !pool.BalanceRune.IsZero() {
		return origAsset
	}

	parts := strings.Split(asset.Symbol.String(), "-")
	hasNoSymbol := len(parts) < 2 || len(parts[1]) == 0
	var symbol string
	if !hasNoSymbol {
		symbol = strings.ToLower(parts[1])
	}
	winner := NewPool()
	// if no asset found, return original asset
	winner.Asset = origAsset
	iterator := keeper.GetPoolIterator(ctx)
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		if err = keeper.Cdc().Unmarshal(iterator.Value(), &pool); err != nil {
			ctx.Logger().Error("fail to fetch pool", "asset", asset, "err", err)
			continue
		}

		// check chain match
		if !asset.Chain.Equals(pool.Asset.Chain) {
			continue
		}

		// check ticker match
		if !asset.Ticker.Equals(pool.Asset.Ticker) {
			continue
		}

		// check if no symbol given (ie "USDT" or "USDT-")
		if hasNoSymbol {
			// Use LTE rather than LT so this function can only return origAsset or a match
			if winner.BalanceRune.LTE(pool.BalanceRune) {
				winner = pool
			}
			continue
		}

		if strings.HasSuffix(strings.ToLower(pool.Asset.Symbol.String()), symbol) {
			// Use LTE rather than LT so this function can only return origAsset or a match
			if winner.BalanceRune.LTE(pool.BalanceRune) {
				winner = pool
			}
			continue
		}
	}
	// Since the Chain and Ticker must already match, replace just the Symbol with the winner's,
	// keeping other fields like Synth and Trade the same as the original.
	origAsset.Symbol = winner.Asset.Symbol
	return origAsset
}

func externalAssetMatch(chain common.Chain, hint string) string {
	if len(hint) == 0 {
		return hint
	}
	if chain.IsEVM() {
		// find all potential matches
		firstMatch := ""
		addrHint := strings.ToLower(hint)
		for _, token := range tokenlist.GetEVMTokenList(chain).Tokens {
			if strings.HasSuffix(strings.ToLower(token.Address), addrHint) {
				// store first found address
				if firstMatch == "" {
					firstMatch = token.Address
				} else {
					return hint
				}
			}
		}
		if firstMatch != "" {
			return firstMatch
		}
	}
	return hint
}
