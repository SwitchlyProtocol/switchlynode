package stellar

import (
	"fmt"
	"strings"

	"github.com/stellar/go/txnbuild"
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
)

// StellarAssetMapping represents the mapping between Stellar assets and THORChain assets
type StellarAssetMapping struct {
	StellarAssetType   string // "native", "credit_alphanum4", "credit_alphanum12"
	StellarAssetCode   string // Asset code (e.g., "USDC", empty for native)
	StellarAssetIssuer string // Issuer address (empty for native)
	StellarDecimals    int64  // Decimals used by the asset on Stellar
	THORChainAsset     common.Asset
}

// StellarAssetMappings defines the supported assets on Stellar
// CHANGEME: Add more assets here as needed. This acts as a whitelist.
var StellarAssetMappings = []StellarAssetMapping{
	{
		StellarAssetType:   "native",
		StellarAssetCode:   "",
		StellarAssetIssuer: "",
		StellarDecimals:    7, // XLM uses 7 decimal places (stroops)
		THORChainAsset:     common.XLMAsset,
	},
	// USDC - Circle's official issuer
	{
		StellarAssetType:   "credit_alphanum4",
		StellarAssetCode:   "USDC",
		StellarAssetIssuer: "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN",
		StellarDecimals:    7, // USDC on Stellar uses 7 decimal places
		THORChainAsset:     common.Asset{Chain: common.StellarChain, Symbol: "USDC", Ticker: "USDC"},
	},
}

// GetAssetByStellarAsset finds the asset mapping by Stellar asset details
func GetAssetByStellarAsset(assetType, assetCode, assetIssuer string) (StellarAssetMapping, bool) {
	for _, mapping := range StellarAssetMappings {
		if mapping.StellarAssetType == assetType {
			if assetType == "native" {
				return mapping, true
			}
			// For non-native assets, check code and issuer
			if strings.EqualFold(mapping.StellarAssetCode, assetCode) &&
				strings.EqualFold(mapping.StellarAssetIssuer, assetIssuer) {
				return mapping, true
			}
		}
	}
	return StellarAssetMapping{}, false
}

// GetAssetByTHORChainAsset finds the asset mapping by THORChain asset
func GetAssetByTHORChainAsset(asset common.Asset) (StellarAssetMapping, bool) {
	for _, mapping := range StellarAssetMappings {
		if asset.Equals(mapping.THORChainAsset) {
			return mapping, true
		}
	}
	return StellarAssetMapping{}, false
}

// ToStellarAsset converts a THORChain asset to a Stellar txnbuild.Asset
func (s StellarAssetMapping) ToStellarAsset() txnbuild.Asset {
	if s.StellarAssetType == "native" {
		return txnbuild.NativeAsset{}
	}

	return txnbuild.CreditAsset{
		Code:   s.StellarAssetCode,
		Issuer: s.StellarAssetIssuer,
	}
}

// FromStellarAsset creates a StellarAssetMapping from a Stellar asset
func FromStellarAsset(asset txnbuild.Asset) (StellarAssetMapping, error) {
	switch a := asset.(type) {
	case txnbuild.NativeAsset:
		mapping, found := GetAssetByStellarAsset("native", "", "")
		if !found {
			return StellarAssetMapping{}, fmt.Errorf("native asset not found in mapping")
		}
		return mapping, nil
	case txnbuild.CreditAsset:
		mapping, found := GetAssetByStellarAsset("credit_alphanum4", a.Code, a.Issuer)
		if !found {
			// Try credit_alphanum12 for longer asset codes
			mapping, found = GetAssetByStellarAsset("credit_alphanum12", a.Code, a.Issuer)
		}
		if found {
			return mapping, nil
		}
		return StellarAssetMapping{}, fmt.Errorf("unsupported asset: %s:%s", a.Code, a.Issuer)
	default:
		return StellarAssetMapping{}, fmt.Errorf("unknown asset type: %T", asset)
	}
}

// ConvertToTHORChainAmount converts a Stellar amount to THORChain amount (1e8 decimals)
func (s StellarAssetMapping) ConvertToTHORChainAmount(stellarAmount string) (common.Coin, error) {
	amount := cosmos.NewUintFromString(stellarAmount)

	// Convert from Stellar decimals to THORChain decimals (1e8)
	if s.StellarDecimals != common.THORChainDecimals {
		if s.StellarDecimals > common.THORChainDecimals {
			// Stellar has more decimals, divide
			decimalDiff := s.StellarDecimals - common.THORChainDecimals
			divisor := cosmos.NewUintFromString(fmt.Sprintf("1%s", strings.Repeat("0", int(decimalDiff))))
			amount = amount.Quo(divisor)
		} else {
			// Stellar has fewer decimals, multiply
			decimalDiff := common.THORChainDecimals - s.StellarDecimals
			multiplier := cosmos.NewUintFromString(fmt.Sprintf("1%s", strings.Repeat("0", int(decimalDiff))))
			amount = amount.Mul(multiplier)
		}
	}

	return common.NewCoin(s.THORChainAsset, amount), nil
}

// ConvertFromTHORChainAmount converts a THORChain amount to Stellar amount
func (s StellarAssetMapping) ConvertFromTHORChainAmount(thorchainAmount cosmos.Uint) string {
	amount := thorchainAmount

	// Convert from THORChain decimals (1e8) to Stellar decimals
	if s.StellarDecimals != common.THORChainDecimals {
		if s.StellarDecimals > common.THORChainDecimals {
			// Stellar has more decimals, multiply
			decimalDiff := s.StellarDecimals - common.THORChainDecimals
			multiplier := cosmos.NewUintFromString(fmt.Sprintf("1%s", strings.Repeat("0", int(decimalDiff))))
			amount = amount.Mul(multiplier)
		} else {
			// Stellar has fewer decimals, divide
			decimalDiff := common.THORChainDecimals - s.StellarDecimals
			divisor := cosmos.NewUintFromString(fmt.Sprintf("1%s", strings.Repeat("0", int(decimalDiff))))
			amount = amount.Quo(divisor)
		}
	}

	return amount.String()
}
