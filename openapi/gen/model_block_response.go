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

// BlockResponse struct for BlockResponse
type BlockResponse struct {
	Id BlockResponseId `json:"id"`
	Header BlockResponseHeader `json:"header"`
	FinalizeBlockEvents []map[string]string `json:"finalize_block_events"`
	BeginBlockEvents []map[string]string `json:"begin_block_events"`
	EndBlockEvents []map[string]string `json:"end_block_events"`
	Txs []BlockTx `json:"txs"`
}

// NewBlockResponse instantiates a new BlockResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewBlockResponse(id BlockResponseId, header BlockResponseHeader, finalizeBlockEvents []map[string]string, beginBlockEvents []map[string]string, endBlockEvents []map[string]string, txs []BlockTx) *BlockResponse {
	this := BlockResponse{}
	this.Id = id
	this.Header = header
	this.FinalizeBlockEvents = finalizeBlockEvents
	this.BeginBlockEvents = beginBlockEvents
	this.EndBlockEvents = endBlockEvents
	this.Txs = txs
	return &this
}

// NewBlockResponseWithDefaults instantiates a new BlockResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewBlockResponseWithDefaults() *BlockResponse {
	this := BlockResponse{}
	return &this
}

// GetId returns the Id field value
func (o *BlockResponse) GetId() BlockResponseId {
	if o == nil {
		var ret BlockResponseId
		return ret
	}

	return o.Id
}

// GetIdOk returns a tuple with the Id field value
// and a boolean to check if the value has been set.
func (o *BlockResponse) GetIdOk() (*BlockResponseId, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Id, true
}

// SetId sets field value
func (o *BlockResponse) SetId(v BlockResponseId) {
	o.Id = v
}

// GetHeader returns the Header field value
func (o *BlockResponse) GetHeader() BlockResponseHeader {
	if o == nil {
		var ret BlockResponseHeader
		return ret
	}

	return o.Header
}

// GetHeaderOk returns a tuple with the Header field value
// and a boolean to check if the value has been set.
func (o *BlockResponse) GetHeaderOk() (*BlockResponseHeader, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Header, true
}

// SetHeader sets field value
func (o *BlockResponse) SetHeader(v BlockResponseHeader) {
	o.Header = v
}

// GetFinalizeBlockEvents returns the FinalizeBlockEvents field value
func (o *BlockResponse) GetFinalizeBlockEvents() []map[string]string {
	if o == nil {
		var ret []map[string]string
		return ret
	}

	return o.FinalizeBlockEvents
}

// GetFinalizeBlockEventsOk returns a tuple with the FinalizeBlockEvents field value
// and a boolean to check if the value has been set.
func (o *BlockResponse) GetFinalizeBlockEventsOk() ([]map[string]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.FinalizeBlockEvents, true
}

// SetFinalizeBlockEvents sets field value
func (o *BlockResponse) SetFinalizeBlockEvents(v []map[string]string) {
	o.FinalizeBlockEvents = v
}

// GetBeginBlockEvents returns the BeginBlockEvents field value
func (o *BlockResponse) GetBeginBlockEvents() []map[string]string {
	if o == nil {
		var ret []map[string]string
		return ret
	}

	return o.BeginBlockEvents
}

// GetBeginBlockEventsOk returns a tuple with the BeginBlockEvents field value
// and a boolean to check if the value has been set.
func (o *BlockResponse) GetBeginBlockEventsOk() ([]map[string]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.BeginBlockEvents, true
}

// SetBeginBlockEvents sets field value
func (o *BlockResponse) SetBeginBlockEvents(v []map[string]string) {
	o.BeginBlockEvents = v
}

// GetEndBlockEvents returns the EndBlockEvents field value
func (o *BlockResponse) GetEndBlockEvents() []map[string]string {
	if o == nil {
		var ret []map[string]string
		return ret
	}

	return o.EndBlockEvents
}

// GetEndBlockEventsOk returns a tuple with the EndBlockEvents field value
// and a boolean to check if the value has been set.
func (o *BlockResponse) GetEndBlockEventsOk() ([]map[string]string, bool) {
	if o == nil {
		return nil, false
	}
	return o.EndBlockEvents, true
}

// SetEndBlockEvents sets field value
func (o *BlockResponse) SetEndBlockEvents(v []map[string]string) {
	o.EndBlockEvents = v
}

// GetTxs returns the Txs field value
// If the value is explicit nil, the zero value for []BlockTx will be returned
func (o *BlockResponse) GetTxs() []BlockTx {
	if o == nil {
		var ret []BlockTx
		return ret
	}

	return o.Txs
}

// GetTxsOk returns a tuple with the Txs field value
// and a boolean to check if the value has been set.
// NOTE: If the value is an explicit nil, `nil, true` will be returned
func (o *BlockResponse) GetTxsOk() ([]BlockTx, bool) {
	if o == nil || o.Txs == nil {
		return nil, false
	}
	return o.Txs, true
}

// SetTxs sets field value
func (o *BlockResponse) SetTxs(v []BlockTx) {
	o.Txs = v
}

func (o BlockResponse) MarshalJSON_deprecated() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["id"] = o.Id
	}
	if true {
		toSerialize["header"] = o.Header
	}
	if true {
		toSerialize["finalize_block_events"] = o.FinalizeBlockEvents
	}
	if true {
		toSerialize["begin_block_events"] = o.BeginBlockEvents
	}
	if true {
		toSerialize["end_block_events"] = o.EndBlockEvents
	}
	if o.Txs != nil {
		toSerialize["txs"] = o.Txs
	}
	return json.Marshal(toSerialize)
}

type NullableBlockResponse struct {
	value *BlockResponse
	isSet bool
}

func (v NullableBlockResponse) Get() *BlockResponse {
	return v.value
}

func (v *NullableBlockResponse) Set(val *BlockResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableBlockResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableBlockResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableBlockResponse(val *BlockResponse) *NullableBlockResponse {
	return &NullableBlockResponse{value: val, isSet: true}
}

func (v NullableBlockResponse) MarshalJSON_deprecated() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableBlockResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


