package types

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
)

func NewTradeUnit(asset common.Asset) TradeUnit {
	return TradeUnit{
		Asset: asset,
		Units: cosmos.ZeroUint(),
		Depth: cosmos.ZeroUint(),
	}
}

func (tu TradeUnit) Key() string {
	return tu.Asset.String()
}
