package gaia

import (
	"strings"
)

type CosmosAssetMapping struct {
	CosmosDenom            string
	CosmosDecimals         int
	SwitchlyProtocolSymbol string
}

func (c *CosmosBlockScanner) GetAssetByCosmosDenom(denom string) (CosmosAssetMapping, bool) {
	for _, asset := range c.cfg.WhitelistCosmosAssets {
		if strings.EqualFold(asset.Denom, denom) {
			return CosmosAssetMapping{
				CosmosDenom:            asset.Denom,
				CosmosDecimals:         asset.Decimals,
				SwitchlyProtocolSymbol: asset.SwitchlySymbol,
			}, true
		}
	}
	return CosmosAssetMapping{}, false
}

func (c *CosmosBlockScanner) GetAssetBySwitchlySymbol(symbol string) (CosmosAssetMapping, bool) {
	for _, asset := range c.cfg.WhitelistCosmosAssets {
		if strings.EqualFold(asset.SwitchlySymbol, symbol) {
			return CosmosAssetMapping{
				CosmosDenom:            asset.Denom,
				CosmosDecimals:         asset.Decimals,
				SwitchlyProtocolSymbol: asset.SwitchlySymbol,
			}, true
		}
	}
	return CosmosAssetMapping{}, false
}
