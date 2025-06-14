/*
Thornode API

Thornode REST API.

Contact: devs@thorchain.org
*/

// Code generated by OpenAPI Generator (https://openapi-generator.tech); DO NOT EDIT.

package openapi

import (
	"encoding/json"
)

// InboundAddress struct for InboundAddress
type InboundAddress struct {
	Chain *string `json:"chain,omitempty"`
	PubKey *string `json:"pub_key,omitempty"`
	Address *string `json:"address,omitempty"`
	Router *string `json:"router,omitempty"`
	// Returns true if trading is unavailable for this chain, either because trading is halted globally or specifically for this chain
	Halted bool `json:"halted"`
	// Returns true if trading is paused globally
	GlobalTradingPaused bool `json:"global_trading_paused"`
	// Returns true if trading is paused for this chain
	ChainTradingPaused bool `json:"chain_trading_paused"`
	// Returns true if LP actions are paused for this chain
	ChainLpActionsPaused bool `json:"chain_lp_actions_paused"`
	// The chain's observed fee rate in 1e8 format, before the 1.5x that makes an outbound more likely to have a sufficient gas rate.  Used by validators to check whether they need to report a fee change.
	ObservedFeeRate *string `json:"observed_fee_rate,omitempty"`
	// The minimum fee rate used by vaults to send outbound TXs. The actual fee rate may be higher. For EVM chains this is returned in gwei (1e9).
	GasRate *string `json:"gas_rate,omitempty"`
	// Units of the gas_rate.
	GasRateUnits *string `json:"gas_rate_units,omitempty"`
	// Avg size of outbound TXs on each chain. For UTXO chains it may be larger than average, as it takes into account vault consolidation txs, which can have many vouts
	OutboundTxSize *string `json:"outbound_tx_size,omitempty"`
	// The total outbound fee charged to the user for outbound txs in the gas asset of the chain.  Can be observed_fee_rate * 1.5 * outbound_tx_size or else kept to an equivalent of Mimir key MinimumL1OutboundFeeUSD.
	OutboundFee *string `json:"outbound_fee,omitempty"`
	// Defines the minimum transaction size for the chain in base units (sats, wei, uatom). Transactions with asset amounts lower than the dust_threshold are ignored.
	DustThreshold *string `json:"dust_threshold,omitempty"`
}

// NewInboundAddress instantiates a new InboundAddress object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewInboundAddress(halted bool, globalTradingPaused bool, chainTradingPaused bool, chainLpActionsPaused bool) *InboundAddress {
	this := InboundAddress{}
	this.Halted = halted
	this.GlobalTradingPaused = globalTradingPaused
	this.ChainTradingPaused = chainTradingPaused
	this.ChainLpActionsPaused = chainLpActionsPaused
	return &this
}

// NewInboundAddressWithDefaults instantiates a new InboundAddress object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewInboundAddressWithDefaults() *InboundAddress {
	this := InboundAddress{}
	return &this
}

// GetChain returns the Chain field value if set, zero value otherwise.
func (o *InboundAddress) GetChain() string {
	if o == nil || o.Chain == nil {
		var ret string
		return ret
	}
	return *o.Chain
}

// GetChainOk returns a tuple with the Chain field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetChainOk() (*string, bool) {
	if o == nil || o.Chain == nil {
		return nil, false
	}
	return o.Chain, true
}

// HasChain returns a boolean if a field has been set.
func (o *InboundAddress) HasChain() bool {
	if o != nil && o.Chain != nil {
		return true
	}

	return false
}

// SetChain gets a reference to the given string and assigns it to the Chain field.
func (o *InboundAddress) SetChain(v string) {
	o.Chain = &v
}

// GetPubKey returns the PubKey field value if set, zero value otherwise.
func (o *InboundAddress) GetPubKey() string {
	if o == nil || o.PubKey == nil {
		var ret string
		return ret
	}
	return *o.PubKey
}

// GetPubKeyOk returns a tuple with the PubKey field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetPubKeyOk() (*string, bool) {
	if o == nil || o.PubKey == nil {
		return nil, false
	}
	return o.PubKey, true
}

// HasPubKey returns a boolean if a field has been set.
func (o *InboundAddress) HasPubKey() bool {
	if o != nil && o.PubKey != nil {
		return true
	}

	return false
}

// SetPubKey gets a reference to the given string and assigns it to the PubKey field.
func (o *InboundAddress) SetPubKey(v string) {
	o.PubKey = &v
}

// GetAddress returns the Address field value if set, zero value otherwise.
func (o *InboundAddress) GetAddress() string {
	if o == nil || o.Address == nil {
		var ret string
		return ret
	}
	return *o.Address
}

// GetAddressOk returns a tuple with the Address field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetAddressOk() (*string, bool) {
	if o == nil || o.Address == nil {
		return nil, false
	}
	return o.Address, true
}

// HasAddress returns a boolean if a field has been set.
func (o *InboundAddress) HasAddress() bool {
	if o != nil && o.Address != nil {
		return true
	}

	return false
}

// SetAddress gets a reference to the given string and assigns it to the Address field.
func (o *InboundAddress) SetAddress(v string) {
	o.Address = &v
}

// GetRouter returns the Router field value if set, zero value otherwise.
func (o *InboundAddress) GetRouter() string {
	if o == nil || o.Router == nil {
		var ret string
		return ret
	}
	return *o.Router
}

// GetRouterOk returns a tuple with the Router field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetRouterOk() (*string, bool) {
	if o == nil || o.Router == nil {
		return nil, false
	}
	return o.Router, true
}

// HasRouter returns a boolean if a field has been set.
func (o *InboundAddress) HasRouter() bool {
	if o != nil && o.Router != nil {
		return true
	}

	return false
}

// SetRouter gets a reference to the given string and assigns it to the Router field.
func (o *InboundAddress) SetRouter(v string) {
	o.Router = &v
}

// GetHalted returns the Halted field value
func (o *InboundAddress) GetHalted() bool {
	if o == nil {
		var ret bool
		return ret
	}

	return o.Halted
}

// GetHaltedOk returns a tuple with the Halted field value
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetHaltedOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Halted, true
}

// SetHalted sets field value
func (o *InboundAddress) SetHalted(v bool) {
	o.Halted = v
}

// GetGlobalTradingPaused returns the GlobalTradingPaused field value
func (o *InboundAddress) GetGlobalTradingPaused() bool {
	if o == nil {
		var ret bool
		return ret
	}

	return o.GlobalTradingPaused
}

// GetGlobalTradingPausedOk returns a tuple with the GlobalTradingPaused field value
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetGlobalTradingPausedOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.GlobalTradingPaused, true
}

// SetGlobalTradingPaused sets field value
func (o *InboundAddress) SetGlobalTradingPaused(v bool) {
	o.GlobalTradingPaused = v
}

// GetChainTradingPaused returns the ChainTradingPaused field value
func (o *InboundAddress) GetChainTradingPaused() bool {
	if o == nil {
		var ret bool
		return ret
	}

	return o.ChainTradingPaused
}

// GetChainTradingPausedOk returns a tuple with the ChainTradingPaused field value
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetChainTradingPausedOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ChainTradingPaused, true
}

// SetChainTradingPaused sets field value
func (o *InboundAddress) SetChainTradingPaused(v bool) {
	o.ChainTradingPaused = v
}

// GetChainLpActionsPaused returns the ChainLpActionsPaused field value
func (o *InboundAddress) GetChainLpActionsPaused() bool {
	if o == nil {
		var ret bool
		return ret
	}

	return o.ChainLpActionsPaused
}

// GetChainLpActionsPausedOk returns a tuple with the ChainLpActionsPaused field value
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetChainLpActionsPausedOk() (*bool, bool) {
	if o == nil {
		return nil, false
	}
	return &o.ChainLpActionsPaused, true
}

// SetChainLpActionsPaused sets field value
func (o *InboundAddress) SetChainLpActionsPaused(v bool) {
	o.ChainLpActionsPaused = v
}

// GetObservedFeeRate returns the ObservedFeeRate field value if set, zero value otherwise.
func (o *InboundAddress) GetObservedFeeRate() string {
	if o == nil || o.ObservedFeeRate == nil {
		var ret string
		return ret
	}
	return *o.ObservedFeeRate
}

// GetObservedFeeRateOk returns a tuple with the ObservedFeeRate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetObservedFeeRateOk() (*string, bool) {
	if o == nil || o.ObservedFeeRate == nil {
		return nil, false
	}
	return o.ObservedFeeRate, true
}

// HasObservedFeeRate returns a boolean if a field has been set.
func (o *InboundAddress) HasObservedFeeRate() bool {
	if o != nil && o.ObservedFeeRate != nil {
		return true
	}

	return false
}

// SetObservedFeeRate gets a reference to the given string and assigns it to the ObservedFeeRate field.
func (o *InboundAddress) SetObservedFeeRate(v string) {
	o.ObservedFeeRate = &v
}

// GetGasRate returns the GasRate field value if set, zero value otherwise.
func (o *InboundAddress) GetGasRate() string {
	if o == nil || o.GasRate == nil {
		var ret string
		return ret
	}
	return *o.GasRate
}

// GetGasRateOk returns a tuple with the GasRate field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetGasRateOk() (*string, bool) {
	if o == nil || o.GasRate == nil {
		return nil, false
	}
	return o.GasRate, true
}

// HasGasRate returns a boolean if a field has been set.
func (o *InboundAddress) HasGasRate() bool {
	if o != nil && o.GasRate != nil {
		return true
	}

	return false
}

// SetGasRate gets a reference to the given string and assigns it to the GasRate field.
func (o *InboundAddress) SetGasRate(v string) {
	o.GasRate = &v
}

// GetGasRateUnits returns the GasRateUnits field value if set, zero value otherwise.
func (o *InboundAddress) GetGasRateUnits() string {
	if o == nil || o.GasRateUnits == nil {
		var ret string
		return ret
	}
	return *o.GasRateUnits
}

// GetGasRateUnitsOk returns a tuple with the GasRateUnits field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetGasRateUnitsOk() (*string, bool) {
	if o == nil || o.GasRateUnits == nil {
		return nil, false
	}
	return o.GasRateUnits, true
}

// HasGasRateUnits returns a boolean if a field has been set.
func (o *InboundAddress) HasGasRateUnits() bool {
	if o != nil && o.GasRateUnits != nil {
		return true
	}

	return false
}

// SetGasRateUnits gets a reference to the given string and assigns it to the GasRateUnits field.
func (o *InboundAddress) SetGasRateUnits(v string) {
	o.GasRateUnits = &v
}

// GetOutboundTxSize returns the OutboundTxSize field value if set, zero value otherwise.
func (o *InboundAddress) GetOutboundTxSize() string {
	if o == nil || o.OutboundTxSize == nil {
		var ret string
		return ret
	}
	return *o.OutboundTxSize
}

// GetOutboundTxSizeOk returns a tuple with the OutboundTxSize field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetOutboundTxSizeOk() (*string, bool) {
	if o == nil || o.OutboundTxSize == nil {
		return nil, false
	}
	return o.OutboundTxSize, true
}

// HasOutboundTxSize returns a boolean if a field has been set.
func (o *InboundAddress) HasOutboundTxSize() bool {
	if o != nil && o.OutboundTxSize != nil {
		return true
	}

	return false
}

// SetOutboundTxSize gets a reference to the given string and assigns it to the OutboundTxSize field.
func (o *InboundAddress) SetOutboundTxSize(v string) {
	o.OutboundTxSize = &v
}

// GetOutboundFee returns the OutboundFee field value if set, zero value otherwise.
func (o *InboundAddress) GetOutboundFee() string {
	if o == nil || o.OutboundFee == nil {
		var ret string
		return ret
	}
	return *o.OutboundFee
}

// GetOutboundFeeOk returns a tuple with the OutboundFee field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetOutboundFeeOk() (*string, bool) {
	if o == nil || o.OutboundFee == nil {
		return nil, false
	}
	return o.OutboundFee, true
}

// HasOutboundFee returns a boolean if a field has been set.
func (o *InboundAddress) HasOutboundFee() bool {
	if o != nil && o.OutboundFee != nil {
		return true
	}

	return false
}

// SetOutboundFee gets a reference to the given string and assigns it to the OutboundFee field.
func (o *InboundAddress) SetOutboundFee(v string) {
	o.OutboundFee = &v
}

// GetDustThreshold returns the DustThreshold field value if set, zero value otherwise.
func (o *InboundAddress) GetDustThreshold() string {
	if o == nil || o.DustThreshold == nil {
		var ret string
		return ret
	}
	return *o.DustThreshold
}

// GetDustThresholdOk returns a tuple with the DustThreshold field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *InboundAddress) GetDustThresholdOk() (*string, bool) {
	if o == nil || o.DustThreshold == nil {
		return nil, false
	}
	return o.DustThreshold, true
}

// HasDustThreshold returns a boolean if a field has been set.
func (o *InboundAddress) HasDustThreshold() bool {
	if o != nil && o.DustThreshold != nil {
		return true
	}

	return false
}

// SetDustThreshold gets a reference to the given string and assigns it to the DustThreshold field.
func (o *InboundAddress) SetDustThreshold(v string) {
	o.DustThreshold = &v
}

func (o InboundAddress) MarshalJSON_deprecated() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Chain != nil {
		toSerialize["chain"] = o.Chain
	}
	if o.PubKey != nil {
		toSerialize["pub_key"] = o.PubKey
	}
	if o.Address != nil {
		toSerialize["address"] = o.Address
	}
	if o.Router != nil {
		toSerialize["router"] = o.Router
	}
	if true {
		toSerialize["halted"] = o.Halted
	}
	if true {
		toSerialize["global_trading_paused"] = o.GlobalTradingPaused
	}
	if true {
		toSerialize["chain_trading_paused"] = o.ChainTradingPaused
	}
	if true {
		toSerialize["chain_lp_actions_paused"] = o.ChainLpActionsPaused
	}
	if o.ObservedFeeRate != nil {
		toSerialize["observed_fee_rate"] = o.ObservedFeeRate
	}
	if o.GasRate != nil {
		toSerialize["gas_rate"] = o.GasRate
	}
	if o.GasRateUnits != nil {
		toSerialize["gas_rate_units"] = o.GasRateUnits
	}
	if o.OutboundTxSize != nil {
		toSerialize["outbound_tx_size"] = o.OutboundTxSize
	}
	if o.OutboundFee != nil {
		toSerialize["outbound_fee"] = o.OutboundFee
	}
	if o.DustThreshold != nil {
		toSerialize["dust_threshold"] = o.DustThreshold
	}
	return json.Marshal(toSerialize)
}

type NullableInboundAddress struct {
	value *InboundAddress
	isSet bool
}

func (v NullableInboundAddress) Get() *InboundAddress {
	return v.value
}

func (v *NullableInboundAddress) Set(val *InboundAddress) {
	v.value = val
	v.isSet = true
}

func (v NullableInboundAddress) IsSet() bool {
	return v.isSet
}

func (v *NullableInboundAddress) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableInboundAddress(val *InboundAddress) *NullableInboundAddress {
	return &NullableInboundAddress{value: val, isSet: true}
}

func (v NullableInboundAddress) MarshalJSON_deprecated() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableInboundAddress) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


