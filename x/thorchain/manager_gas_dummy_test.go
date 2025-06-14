package thorchain

import (
	"github.com/switchlyprotocol/switchlynode/v1/common"
	"github.com/switchlyprotocol/switchlynode/v1/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/keeper"
	"github.com/switchlyprotocol/switchlynode/v1/x/thorchain/types"
)

type DummyGasManager struct{}

func NewDummyGasManager() *DummyGasManager {
	return &DummyGasManager{}
}

func (m *DummyGasManager) BeginBlock() {}
func (m *DummyGasManager) EndBlock(ctx cosmos.Context, keeper keeper.Keeper, eventManager EventManager) {
}
func (m *DummyGasManager) AddGasAsset(_ common.Asset, gas common.Gas, increaseTxCount bool) {}
func (m *DummyGasManager) AddGas(gas common.Gas)                                            {}
func (m *DummyGasManager) GetGas() common.Gas                                               { return nil }
func (m *DummyGasManager) ProcessGas(ctx cosmos.Context, keeper keeper.Keeper)              {}
func (m *DummyGasManager) GetAssetOutboundFee(ctx cosmos.Context, asset common.Asset, inRune bool) (cosmos.Uint, error) {
	return cosmos.ZeroUint(), nil
}

func (m *DummyGasManager) GetFee(ctx cosmos.Context, chain common.Chain, _ common.Asset) cosmos.Uint {
	return cosmos.ZeroUint()
}

func (m *DummyGasManager) CalcOutboundFeeMultiplier(ctx cosmos.Context, targetSurplusRune, gasSpentRune, gasWithheldRune, maxMultiplier, minMultiplier cosmos.Uint) cosmos.Uint {
	return cosmos.ZeroUint()
}

func (m *DummyGasManager) GetMaxGas(ctx cosmos.Context, chain common.Chain) (common.Coin, error) {
	if chain.Equals(common.BTCChain) {
		return common.NewCoin(common.BTCAsset, cosmos.NewUint(1000)), nil
	} else if chain.Equals(common.ETHChain) {
		return common.NewCoin(common.ETHAsset, cosmos.NewUint(37500)), nil
	}
	return common.NoCoin, errKaboom
}

func (m *DummyGasManager) GetGasRate(ctx cosmos.Context, chain common.Chain) cosmos.Uint {
	return cosmos.OneUint()
}

func (m *DummyGasManager) GetNetworkFee(ctx cosmos.Context, chain common.Chain) (types.NetworkFee, error) {
	return types.NetworkFee{}, nil
}
