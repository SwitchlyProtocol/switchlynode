package xrp

import (
	"fmt"
	"math/big"
	"strconv"

	sdkmath "cosmossdk.io/math"

	"github.com/switchlyprotocol/switchlynode/v3/common"

	txtypes "github.com/Peersyst/xrpl-go/xrpl/transaction/types"
)

func parseCurrencyAmount(coin txtypes.CurrencyAmount) (*big.Int, error) {
	if xrpAmount, ok := coin.(txtypes.XRPCurrencyAmount); ok {
		return big.NewInt(int64(xrpAmount.Uint64())), nil
	}
	if issuedAmount, ok := coin.(txtypes.IssuedCurrencyAmount); ok {
		amount, err := strconv.ParseInt(issuedAmount.Value, 10, 64)
		if err != nil {
			return nil, err
		}
		return big.NewInt(amount), nil
	}
	return nil, fmt.Errorf("invalid xrp currency type")
}

func fromXrpToSwitchly(coin txtypes.CurrencyAmount) (common.Coin, error) {
	asset, exists := GetAssetByXrpCurrency(coin)
	if !exists {
		return common.NoCoin, fmt.Errorf("asset does not exist / not whitelisted by client")
	}

	decimals := asset.XrpDecimals
	amount, err := parseCurrencyAmount(coin)
	if err != nil {
		return common.NoCoin, err
	}
	var exp big.Int
	// Decimals are more than native SWITCHLYChain, so divide...
	if decimals > common.SwitchlyDecimals {
		decimalDiff := decimals - common.SwitchlyDecimals
		amount.Quo(amount, exp.Exp(big.NewInt(10), big.NewInt(decimalDiff), nil))
	} else if decimals < common.SwitchlyDecimals {
		// Decimals are less than native SWITCHLYChain, so multiply...
		decimalDiff := common.SwitchlyDecimals - decimals
		amount.Mul(amount, exp.Exp(big.NewInt(10), big.NewInt(decimalDiff), nil))
	}
	return common.Coin{
		Asset:    asset.SwitchlyAsset,
		Amount:   sdkmath.NewUintFromBigInt(amount),
		Decimals: decimals,
	}, nil
}

func fromSwitchlyToXrp(coin common.Coin) (txtypes.CurrencyAmount, error) {
	asset, exists := GetAssetBySwitchlyAsset(coin.Asset)
	if !exists {
		return nil, fmt.Errorf("asset (%s) does not exist / not whitelisted by client", coin.Asset)
	}

	decimals := asset.XrpDecimals
	amount := coin.Amount.BigInt()
	var exp big.Int
	if decimals > common.SwitchlyDecimals {
		// Decimals are more than native SWITCHLYChain, so multiply...
		decimalDiff := decimals - common.SwitchlyDecimals
		amount.Mul(amount, exp.Exp(big.NewInt(10), big.NewInt(decimalDiff), nil))
	} else if decimals < common.SwitchlyDecimals {
		// Decimals are less than native SWITCHLYChain, so divide...
		decimalDiff := common.SwitchlyDecimals - decimals
		amount.Quo(amount, exp.Exp(big.NewInt(10), big.NewInt(decimalDiff), nil))
	}

	if asset.XrpKind == txtypes.ISSUED {
		return txtypes.IssuedCurrencyAmount{
			Issuer:   txtypes.Address(asset.XrpIssuer),
			Currency: asset.XrpCurrency,
			Value:    amount.String(),
		}, nil
	}

	return txtypes.XRPCurrencyAmount(amount.Uint64()), nil
}
