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

// UpgradeProposal struct for UpgradeProposal
type UpgradeProposal struct {
	// the name of the upgrade
	Name string `json:"name"`
	// the block height at which the upgrade will occur
	Height int64 `json:"height"`
	// the description of the upgrade, typically json with URLs to binaries for use with automation tools
	Info string `json:"info"`
	// whether the upgrade has been approved by the active validators
	Approved *bool `json:"approved,omitempty"`
	// the percentage of active validators that have approved the upgrade
	ApprovedPercent *string `json:"approved_percent,omitempty"`
	// the amount of additional active validators required to reach quorum for the upgrade
	ValidatorsToQuorum *int64 `json:"validators_to_quorum,omitempty"`
}

// NewUpgradeProposal instantiates a new UpgradeProposal object
// This constructor will assign default values to properties that have it defined,
// and makes sure properties required by API are set, but the set of arguments
// will change when the set of required properties is changed
func NewUpgradeProposal(name string, height int64, info string) *UpgradeProposal {
	this := UpgradeProposal{}
	this.Name = name
	this.Height = height
	this.Info = info
	return &this
}

// NewUpgradeProposalWithDefaults instantiates a new UpgradeProposal object
// This constructor will only assign default values to properties that have it defined,
// but it doesn't guarantee that properties required by API are set
func NewUpgradeProposalWithDefaults() *UpgradeProposal {
	this := UpgradeProposal{}
	return &this
}

// GetName returns the Name field value
func (o *UpgradeProposal) GetName() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Name
}

// GetNameOk returns a tuple with the Name field value
// and a boolean to check if the value has been set.
func (o *UpgradeProposal) GetNameOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Name, true
}

// SetName sets field value
func (o *UpgradeProposal) SetName(v string) {
	o.Name = v
}

// GetHeight returns the Height field value
func (o *UpgradeProposal) GetHeight() int64 {
	if o == nil {
		var ret int64
		return ret
	}

	return o.Height
}

// GetHeightOk returns a tuple with the Height field value
// and a boolean to check if the value has been set.
func (o *UpgradeProposal) GetHeightOk() (*int64, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Height, true
}

// SetHeight sets field value
func (o *UpgradeProposal) SetHeight(v int64) {
	o.Height = v
}

// GetInfo returns the Info field value
func (o *UpgradeProposal) GetInfo() string {
	if o == nil {
		var ret string
		return ret
	}

	return o.Info
}

// GetInfoOk returns a tuple with the Info field value
// and a boolean to check if the value has been set.
func (o *UpgradeProposal) GetInfoOk() (*string, bool) {
	if o == nil {
		return nil, false
	}
	return &o.Info, true
}

// SetInfo sets field value
func (o *UpgradeProposal) SetInfo(v string) {
	o.Info = v
}

// GetApproved returns the Approved field value if set, zero value otherwise.
func (o *UpgradeProposal) GetApproved() bool {
	if o == nil || o.Approved == nil {
		var ret bool
		return ret
	}
	return *o.Approved
}

// GetApprovedOk returns a tuple with the Approved field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UpgradeProposal) GetApprovedOk() (*bool, bool) {
	if o == nil || o.Approved == nil {
		return nil, false
	}
	return o.Approved, true
}

// HasApproved returns a boolean if a field has been set.
func (o *UpgradeProposal) HasApproved() bool {
	if o != nil && o.Approved != nil {
		return true
	}

	return false
}

// SetApproved gets a reference to the given bool and assigns it to the Approved field.
func (o *UpgradeProposal) SetApproved(v bool) {
	o.Approved = &v
}

// GetApprovedPercent returns the ApprovedPercent field value if set, zero value otherwise.
func (o *UpgradeProposal) GetApprovedPercent() string {
	if o == nil || o.ApprovedPercent == nil {
		var ret string
		return ret
	}
	return *o.ApprovedPercent
}

// GetApprovedPercentOk returns a tuple with the ApprovedPercent field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UpgradeProposal) GetApprovedPercentOk() (*string, bool) {
	if o == nil || o.ApprovedPercent == nil {
		return nil, false
	}
	return o.ApprovedPercent, true
}

// HasApprovedPercent returns a boolean if a field has been set.
func (o *UpgradeProposal) HasApprovedPercent() bool {
	if o != nil && o.ApprovedPercent != nil {
		return true
	}

	return false
}

// SetApprovedPercent gets a reference to the given string and assigns it to the ApprovedPercent field.
func (o *UpgradeProposal) SetApprovedPercent(v string) {
	o.ApprovedPercent = &v
}

// GetValidatorsToQuorum returns the ValidatorsToQuorum field value if set, zero value otherwise.
func (o *UpgradeProposal) GetValidatorsToQuorum() int64 {
	if o == nil || o.ValidatorsToQuorum == nil {
		var ret int64
		return ret
	}
	return *o.ValidatorsToQuorum
}

// GetValidatorsToQuorumOk returns a tuple with the ValidatorsToQuorum field value if set, nil otherwise
// and a boolean to check if the value has been set.
func (o *UpgradeProposal) GetValidatorsToQuorumOk() (*int64, bool) {
	if o == nil || o.ValidatorsToQuorum == nil {
		return nil, false
	}
	return o.ValidatorsToQuorum, true
}

// HasValidatorsToQuorum returns a boolean if a field has been set.
func (o *UpgradeProposal) HasValidatorsToQuorum() bool {
	if o != nil && o.ValidatorsToQuorum != nil {
		return true
	}

	return false
}

// SetValidatorsToQuorum gets a reference to the given int64 and assigns it to the ValidatorsToQuorum field.
func (o *UpgradeProposal) SetValidatorsToQuorum(v int64) {
	o.ValidatorsToQuorum = &v
}

func (o UpgradeProposal) MarshalJSON_deprecated() ([]byte, error) {
	toSerialize := map[string]interface{}{}
	if true {
		toSerialize["name"] = o.Name
	}
	if true {
		toSerialize["height"] = o.Height
	}
	if true {
		toSerialize["info"] = o.Info
	}
	if o.Approved != nil {
		toSerialize["approved"] = o.Approved
	}
	if o.ApprovedPercent != nil {
		toSerialize["approved_percent"] = o.ApprovedPercent
	}
	if o.ValidatorsToQuorum != nil {
		toSerialize["validators_to_quorum"] = o.ValidatorsToQuorum
	}
	return json.Marshal(toSerialize)
}

type NullableUpgradeProposal struct {
	value *UpgradeProposal
	isSet bool
}

func (v NullableUpgradeProposal) Get() *UpgradeProposal {
	return v.value
}

func (v *NullableUpgradeProposal) Set(val *UpgradeProposal) {
	v.value = val
	v.isSet = true
}

func (v NullableUpgradeProposal) IsSet() bool {
	return v.isSet
}

func (v *NullableUpgradeProposal) Unset() {
	v.value = nil
	v.isSet = false
}

func NewNullableUpgradeProposal(val *UpgradeProposal) *NullableUpgradeProposal {
	return &NullableUpgradeProposal{value: val, isSet: true}
}

func (v NullableUpgradeProposal) MarshalJSON_deprecated() ([]byte, error) {
	return json.Marshal(v.value)
}

func (v *NullableUpgradeProposal) UnmarshalJSON(src []byte) error {
	v.isSet = true
	return json.Unmarshal(src, &v.value)
}


