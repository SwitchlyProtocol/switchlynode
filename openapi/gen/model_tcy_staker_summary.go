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

// TCYStakerSummary struct for TCYStakerSummary
type TCYStakerSummary struct {
	Amount string `json:"amount"`
}

// NewTCYStakerSummary instantiates a new TCYStakerSummary object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewTCYStakerSummary(amount string) *TCYStakerSummary {
	this := TCYStakerSummary{}
	this.Amount = amount
	return &this
}

// NewTCYStakerSummaryWithDefaults instantiates a new TCYStakerSummary object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewTCYStakerSummaryWithDefaults() *TCYStakerSummary {
	this := TCYStakerSummary{}
	return &this
}

// GetAmount returns the Amount field value
func (o *TCYStakerSummary) GetAmount() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Amount
}

// GetAmountOk returns a tuple with the Amount field value
// and a boolean to check if the value has been set.
func (o *TCYStakerSummary) GetAmountOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Amount, true
}

// SetAmount sets field value
func (o *TCYStakerSummary) SetAmount(v string) {
	o.Amount = v
}

func (o TCYStakerSummary) MarshalJSON_deprecated() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["amount"] = o.Amount
	}
	return json.Marshal(toSerialize)
}

type NullableTCYStakerSummary struct {
	value *TCYStakerSummary
	isSet bool
}

func (v NullableTCYStakerSummary) Get() *TCYStakerSummary {
	return v.value
}

func (v *NullableTCYStakerSummary) Set(val *TCYStakerSummary) {
	v.value = val
	v.isSet = true
}

func (v NullableTCYStakerSummary) IsSet() bool {
	return v.isSet
}

func (v *NullableTCYStakerSummary) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableTCYStakerSummary(val *TCYStakerSummary) *NullableTCYStakerSummary {
	return &NullableTCYStakerSummary{value: val, isSet: true}
}

func (v NullableTCYStakerSummary) MarshalJSON_deprecated() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableTCYStakerSummary) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


