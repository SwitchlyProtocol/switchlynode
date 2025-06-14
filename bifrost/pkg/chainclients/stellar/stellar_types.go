package stellar

import (
	"gitlab.com/thorchain/thornode/common"
)

const (
	maxGasAmount       = 100000
	maxMemoLength      = 28
	maxRetries         = 3
	defaultTimeoutSecs = 300
	minTxValue         = 1000000 // 0.1 XLM in stroops
)

var (
	stellarAsset      = common.XLMAsset
	stellarUSDC       = common.Asset{Chain: common.STELLARChain, Symbol: "USDC", Ticker: "USDC", Synth: false}
	stellarUSDCIssuer = "GA5ZSEJYB37JRC5AVCIA5MOP4RHTM335X2KGX3IHOJAPP5RE34K4KZVN" // Stellar USDC issuer
)

// StellarAsset represents a Stellar asset
type StellarAsset struct {
	Type   string `json:"type"`
	Code   string `json:"code,omitempty"`
	Issuer string `json:"issuer,omitempty"`
}

// NewStellarAsset creates a new Stellar asset
func NewStellarAsset(asset common.Asset) StellarAsset {
	if asset.Equals(stellarAsset) {
		return StellarAsset{Type: "native"}
	}
	if asset.Equals(stellarUSDC) {
		return StellarAsset{
			Type:   "credit_alphanum4",
			Code:   "USDC",
			Issuer: stellarUSDCIssuer,
		}
	}
	return StellarAsset{}
}
