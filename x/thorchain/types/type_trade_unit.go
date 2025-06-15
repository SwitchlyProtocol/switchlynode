package types

import (
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
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
