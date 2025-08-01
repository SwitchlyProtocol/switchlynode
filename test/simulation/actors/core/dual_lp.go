package core

import (
	"fmt"

	"github.com/hashicorp/go-multierror"

	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/thornode"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"

	. "github.com/switchlyprotocol/switchlynode/v1/test/simulation/actors/common"
	. "github.com/switchlyprotocol/switchlynode/v1/test/simulation/pkg/types"
)

////////////////////////////////////////////////////////////////////////////////////////
// DualLPActor
////////////////////////////////////////////////////////////////////////////////////////

type DualLPActor struct {
	Actor

	asset       common.Asset
	account     *User
	thorAddress common.Address
	l1Address   common.Address
	runeAmount  cosmos.Uint
	l1Amount    cosmos.Uint
}

func NewDualLPActor(asset common.Asset) *Actor {
	a := &DualLPActor{
		Actor: *NewActor(fmt.Sprintf("DualLP-%s", asset)),
		asset: asset,
	}

	// lock a user that has L1 and RUNE balance
	a.Ops = append(a.Ops, a.acquireUser)

	// deposit 10% of the user RUNE balance
	a.Ops = append(a.Ops, a.depositRune)

	// deposit 10% of the user L1 balance to match
	if asset.Chain.IsEVM() && !asset.IsGasAsset() {
		a.Ops = append(a.Ops, a.depositL1Token)
	} else {
		a.Ops = append(a.Ops, a.depositL1)
	}

	// ensure the lp is created and release the account
	a.Ops = append(a.Ops, a.verifyLP)

	return &a.Actor
}

////////////////////////////////////////////////////////////////////////////////////////
// Ops
////////////////////////////////////////////////////////////////////////////////////////

func (a *DualLPActor) acquireUser(config *OpConfig) OpResult {
	userMaxRune := cosmos.NewUint(0)

	for _, user := range config.Users {
		a.SetLogger(a.Log().With().Str("user", user.Name()).Logger())

		// skip users already being used
		if !user.Acquire() {
			continue
		}

		// skip users that don't have RUNE balance
		thorAddress, err := user.PubKey().GetAddress(common.THORChain)
		if err != nil {
			a.Log().Error().Err(err).Msg("failed to get thor address")
			user.Release()
			continue
		}
		thorBalances, err := thornode.GetBalances(thorAddress)
		if err != nil {
			a.Log().Error().Err(err).Msg("failed to get thorchain balances")
			user.Release()
			continue
		}
		if thorBalances.GetCoin(common.SwitchNative).Amount.IsZero() {
			a.Log().Error().Msg("user has no RUNE balance")
			user.Release()
			continue
		}

		// skip users that don't have L1 balance
		l1Acct, err := user.ChainClients[a.asset.Chain].GetAccount(nil)
		if err != nil {
			a.Log().Error().Err(err).Msg("failed to get L1 account")
			user.Release()
			continue
		}
		if l1Acct.Coins.GetCoin(a.asset).Amount.IsZero() {
			a.Log().Error().Msg("user has no L1 balance")
			user.Release()
			continue
		}

		// TODO: skip users that already have a position in this pool

		// get l1 address to store in state context
		l1Address, err := user.PubKey().GetAddress(a.asset.Chain)
		if err != nil {
			a.Log().Error().Err(err).Msg("failed to get L1 address")
			user.Release()
			continue
		}

		// find the user with the most RUNE balance
		if thorBalances.GetCoin(common.SwitchNative).Amount.LTE(userMaxRune) {
			user.Release()
			continue
		}
		userMaxRune = thorBalances.GetCoin(common.SwitchNative).Amount

		// set acquired account and amounts in state context
		a.Log().Info().
			Stringer("address", thorAddress).
			Stringer("l1Address", l1Address).
			Msg("acquired user")
		a.thorAddress = thorAddress
		a.l1Address = l1Address
		a.runeAmount = thorBalances.GetCoin(common.SwitchNative).Amount.QuoUint64(5)
		a.l1Amount = l1Acct.Coins.GetCoin(a.asset).Amount.QuoUint64(5)
		a.account = user
	}

	// remain pending if no user is available
	return OpResult{
		Continue: a.account != nil,
	}
}

func (a *DualLPActor) depositL1Token(config *OpConfig) OpResult {
	memo := fmt.Sprintf("+:%s:%s", a.asset, a.thorAddress)
	client := a.account.ChainClients[a.asset.Chain]
	txid, err := DepositL1Token(a.Log(), client, a.asset, memo, a.l1Amount)
	if err != nil {
		a.Log().Error().Err(err).Msg("failed to deposit L1 token")
		return OpResult{
			Continue: false,
		}
	}

	a.Log().Info().Str("txid", txid).Msg("broadcasted token add liquidity tx")
	return OpResult{
		Continue: true,
	}
}

func (a *DualLPActor) depositL1(config *OpConfig) OpResult {
	memo := fmt.Sprintf("+:%s:%s", a.asset, a.thorAddress)
	client := a.account.ChainClients[a.asset.Chain]
	txid, err := DepositL1(a.Log(), client, a.asset, memo, a.l1Amount)
	if err != nil {
		a.Log().Error().Err(err).Msg("failed to deposit L1")
		return OpResult{
			Continue: false,
		}
	}

	a.Log().Info().Str("txid", txid).Msg("broadcasted L1 add liquidity tx")
	return OpResult{
		Continue: true,
	}
}

func (a *DualLPActor) depositRune(config *OpConfig) OpResult {
	memo := fmt.Sprintf("+:%s:%s", a.asset, a.l1Address)
	accAddr, err := a.account.PubKey().GetThorAddress()
	if err != nil {
		a.Log().Error().Err(err).Msg("failed to get thor address")
		return OpResult{
			Continue: false,
		}
	}
	deposit := types.NewMsgDeposit(
		common.NewCoins(common.NewCoin(common.SwitchNative, a.runeAmount)),
		memo,
		accAddr,
	)
	txid, err := a.account.Thorchain.Broadcast(deposit)
	if err != nil {
		a.Log().Error().Err(err).Msg("failed to broadcast tx")
		return OpResult{
			Continue: false,
		}
	}

	a.Log().Info().Stringer("txid", txid).Msg("broadcasted RUNE add liquidity tx")
	return OpResult{
		Continue: true,
	}
}

func (a *DualLPActor) verifyLP(config *OpConfig) OpResult {
	lps, err := thornode.GetLiquidityProviders(a.asset)
	if err != nil {
		a.Log().Error().Err(err).Msg("failed to get liquidity providers")
		return OpResult{
			Continue: false,
		}
	}

	for _, lp := range lps {
		// skip pending lps
		if lp.PendingAsset != "0" || lp.PendingRune != "0" {
			continue
		}

		// find the matching lp record
		if lp.RuneAddress == nil || lp.AssetAddress == nil {
			continue
		}

		if common.Address(*lp.RuneAddress).Equals(a.thorAddress) &&
			common.Address(*lp.AssetAddress).Equals(a.l1Address) {

			// found the matching lp record
			res := OpResult{
				Finish: true,
			}

			// verify the amounts
			if lp.RuneDepositValue != a.runeAmount.String() {
				err = fmt.Errorf("mismatch RUNE amount: %s != %s", lp.RuneDepositValue, a.runeAmount)
				res.Error = multierror.Append(res.Error, err)
			}
			if lp.AssetDepositValue != a.l1Amount.String() {
				err = fmt.Errorf("mismatch L1 amount: %s != %s", lp.AssetDepositValue, a.l1Amount)
				res.Error = multierror.Append(res.Error, err)
			}
			if res.Error != nil {
				a.Log().Error().Err(res.Error).Msg("invalid liquidity provider")
			}

			a.account.Release() // release the user on success
			return res
		}
	}

	// remain pending if no lp is available
	return OpResult{
		Continue: false,
	}
}
