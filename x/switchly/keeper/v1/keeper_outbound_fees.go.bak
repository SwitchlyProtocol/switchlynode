package keeperv1

import (
	"github.com/switchlyprotocol/switchlynode/v3/common"
	"github.com/switchlyprotocol/switchlynode/v3/common/cosmos"
	"github.com/switchlyprotocol/switchlynode/v3/constants"
)

func (k KVStore) GetOutboundTxFee(ctx cosmos.Context) cosmos.Uint {
	if k.usdFeesEnabled(ctx) {
		return k.DollarConfigInSWITCH(ctx, constants.NativeOutboundFeeUSD)
	}
	fee := k.GetConfigInt64(ctx, constants.OutboundTransactionFee)
	return cosmos.NewUint(uint64(fee))
}

// GetOutboundFeeWithheldSwitch - record of SWITCH collected by the Reserve for an Asset's outbound fees
func (k KVStore) GetOutboundFeeWithheldSwitch(ctx cosmos.Context, outAsset common.Asset) (cosmos.Uint, error) {
	var record uint64
	_, err := k.getUint64(ctx, k.GetKey(prefixOutboundFeeWithheldSwitch, outAsset.String()), &record)
	return cosmos.NewUint(record), err
}

// AddToOutboundFeeWithheldSwitch - add to record of SWITCH collected by the Reserve for an Asset's outbound fees
func (k KVStore) AddToOutboundFeeWithheldSwitch(ctx cosmos.Context, outAsset common.Asset, withheld cosmos.Uint) error {
	outboundFeeWithheldSwitch, err := k.GetOutboundFeeWithheldSwitch(ctx, outAsset)
	if err != nil {
		return err
	}

	outboundFeeWithheldSwitch = outboundFeeWithheldSwitch.Add(withheld)
	k.setUint64(ctx, k.GetKey(prefixOutboundFeeWithheldSwitch, outAsset.String()), outboundFeeWithheldSwitch.Uint64())
	return nil
}

// GetOutboundFeeWithheldSwitchIterator to iterate through all Assets' OutboundFeeWithheldSwitch
// (e.g. for hard-fork GenesisState export)
func (k KVStore) GetOutboundFeeWithheldSwitchIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixOutboundFeeWithheldSwitch)
}

// GetOutboundFeeSpentSwitch - record of SWITCH spent by the Reserve for an Asset's outbounds' gas costs
func (k KVStore) GetOutboundFeeSpentSwitch(ctx cosmos.Context, outAsset common.Asset) (cosmos.Uint, error) {
	var record uint64
	_, err := k.getUint64(ctx, k.GetKey(prefixOutboundFeeSpentSwitch, outAsset.String()), &record)
	return cosmos.NewUint(record), err
}

// AddToOutboundFeeSpentSwitch - add to record of SWITCH spent by the Reserve for an Asset's outbounds' gas costs
func (k KVStore) AddToOutboundFeeSpentSwitch(ctx cosmos.Context, outAsset common.Asset, spent cosmos.Uint) error {
	outboundFeeSpentSwitch, err := k.GetOutboundFeeSpentSwitch(ctx, outAsset)
	if err != nil {
		return err
	}

	outboundFeeSpentSwitch = outboundFeeSpentSwitch.Add(spent)
	k.setUint64(ctx, k.GetKey(prefixOutboundFeeSpentSwitch, outAsset.String()), outboundFeeSpentSwitch.Uint64())
	return nil
}

// GetOutboundFeeSpentSwitchIterator to iterate through all Assets' OutboundFeeSpentSwitch
// (e.g. for hard-fork GenesisState export)
func (k KVStore) GetOutboundFeeSpentSwitchIterator(ctx cosmos.Context) cosmos.Iterator {
	return k.getIterator(ctx, prefixOutboundFeeSpentSwitch)
}
