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

// BalancesResponse struct for BalancesResponse
type BalancesResponse struct {
	Result []Amount `json:"result,omitempty"`
}

// NewBalancesResponse instantiates a new BalancesResponse object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewBalancesResponse() *BalancesResponse {
	this := BalancesResponse{}
	return &this
}

// NewBalancesResponseWithDefaults instantiates a new BalancesResponse object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewBalancesResponseWithDefaults() *BalancesResponse {
	this := BalancesResponse{}
	return &this
}

// GetResult returns the Result field value if set, zero value otherwise.
func (o *BalancesResponse) GetResult() []Amount {
	if o == nil || o.Result == nil {
		var ret []Amount
		return ret
	}
	return o.Result
}

// GetResultOk returns a tuple with the Result field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *BalancesResponse) GetResultOk() ([]Amount, bool) {
	if o == nil || o.Result == nil {
		return nil, false
	}
	return o.Result, true
}

// HasResult returns a boolean if a field has been set.
func (o *BalancesResponse) HasResult() bool {
	if o != nil && o.Result != nil {
		return true
	}

	return false
}

// SetResult gets a reference to the given []Amount and assigns it to the Result field.
func (o *BalancesResponse) SetResult(v []Amount) {
	o.Result = v
}

func (o BalancesResponse) MarshalJSON_deprecated() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if o.Result != nil {
		toSerialize["result"] = o.Result
	}
	return json.Marshal(toSerialize)
}

type NullableBalancesResponse struct {
	value *BalancesResponse
	isSet bool
}

func (v NullableBalancesResponse) Get() *BalancesResponse {
	return v.value
}

func (v *NullableBalancesResponse) Set(val *BalancesResponse) {
	v.value = val
	v.isSet = true
}

func (v NullableBalancesResponse) IsSet() bool {
	return v.isSet
}

func (v *NullableBalancesResponse) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableBalancesResponse(val *BalancesResponse) *NullableBalancesResponse {
	return &NullableBalancesResponse{value: val, isSet: true}
}

func (v NullableBalancesResponse) MarshalJSON_deprecated() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableBalancesResponse) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


