# OutboundFee

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Asset** | **string** | the asset to display the outbound fee for | 
**OutboundFee** | **string** | the asset&#39;s outbound fee, in (1e8-format) units of the asset | 
**FeeWithheldSWITCH** | Pointer to **string** | Total SWITCH the network has withheld as fees to later cover gas costs for this asset&#39;s outbounds | [optional] 
**FeeSpentSWITCH** | Pointer to **string** | Total SWITCH the network has spent to reimburse gas costs for this asset&#39;s outbounds | [optional] 
**SurplusSWITCH** | Pointer to **string** | amount of SWITCH by which the fee_withheld_rune exceeds the fee_spent_rune | [optional] 
**DynamicMultiplierBasisPoints** | Pointer to **string** | dynamic multiplier basis points, based on the surplus_rune, affecting the size of the outbound_fee | [optional] 

## Methods

### NewOutboundFee

`func NewOutboundFee(asset string, outboundFee string, ) *OutboundFee`

NewOutboundFee instantiates a new OutboundFee object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewOutboundFeeWithDefaults

`func NewOutboundFeeWithDefaults() *OutboundFee`

NewOutboundFeeWithDefaults instantiates a new OutboundFee object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAsset

`func (o *OutboundFee) GetAsset() string`

GetAsset returns the Asset field if non-nil, zero value otherwise.

### GetAssetOk

`func (o *OutboundFee) GetAssetOk() (*string, bool)`

GetAssetOk returns a tuple with the Asset field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAsset

`func (o *OutboundFee) SetAsset(v string)`

SetAsset sets Asset field to given value.


### GetOutboundFee

`func (o *OutboundFee) GetOutboundFee() string`

GetOutboundFee returns the OutboundFee field if non-nil, zero value otherwise.

### GetOutboundFeeOk

`func (o *OutboundFee) GetOutboundFeeOk() (*string, bool)`

GetOutboundFeeOk returns a tuple with the OutboundFee field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutboundFee

`func (o *OutboundFee) SetOutboundFee(v string)`

SetOutboundFee sets OutboundFee field to given value.


### GetFeeWithheldSWITCH

`func (o *OutboundFee) GetFeeWithheldSWITCH() string`

GetFeeWithheldSWITCH returns the FeeWithheldSWITCH field if non-nil, zero value otherwise.

### GetFeeWithheldSWITCHOk

`func (o *OutboundFee) GetFeeWithheldSWITCHOk() (*string, bool)`

GetFeeWithheldSWITCHOk returns a tuple with the FeeWithheldSWITCH field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFeeWithheldSWITCH

`func (o *OutboundFee) SetFeeWithheldSWITCH(v string)`

SetFeeWithheldSWITCH sets FeeWithheldSWITCH field to given value.

### HasFeeWithheldSWITCH

`func (o *OutboundFee) HasFeeWithheldSWITCH() bool`

HasFeeWithheldSWITCH returns a boolean if a field has been set.

### GetFeeSpentSWITCH

`func (o *OutboundFee) GetFeeSpentSWITCH() string`

GetFeeSpentSWITCH returns the FeeSpentSWITCH field if non-nil, zero value otherwise.

### GetFeeSpentSWITCHOk

`func (o *OutboundFee) GetFeeSpentSWITCHOk() (*string, bool)`

GetFeeSpentSWITCHOk returns a tuple with the FeeSpentSWITCH field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetFeeSpentSWITCH

`func (o *OutboundFee) SetFeeSpentSWITCH(v string)`

SetFeeSpentSWITCH sets FeeSpentSWITCH field to given value.

### HasFeeSpentSWITCH

`func (o *OutboundFee) HasFeeSpentSWITCH() bool`

HasFeeSpentSWITCH returns a boolean if a field has been set.

### GetSurplusSWITCH

`func (o *OutboundFee) GetSurplusSWITCH() string`

GetSurplusSWITCH returns the SurplusSWITCH field if non-nil, zero value otherwise.

### GetSurplusSWITCHOk

`func (o *OutboundFee) GetSurplusSWITCHOk() (*string, bool)`

GetSurplusSWITCHOk returns a tuple with the SurplusSWITCH field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSurplusSWITCH

`func (o *OutboundFee) SetSurplusSWITCH(v string)`

SetSurplusSWITCH sets SurplusSWITCH field to given value.

### HasSurplusSWITCH

`func (o *OutboundFee) HasSurplusSWITCH() bool`

HasSurplusSWITCH returns a boolean if a field has been set.

### GetDynamicMultiplierBasisPoints

`func (o *OutboundFee) GetDynamicMultiplierBasisPoints() string`

GetDynamicMultiplierBasisPoints returns the DynamicMultiplierBasisPoints field if non-nil, zero value otherwise.

### GetDynamicMultiplierBasisPointsOk

`func (o *OutboundFee) GetDynamicMultiplierBasisPointsOk() (*string, bool)`

GetDynamicMultiplierBasisPointsOk returns a tuple with the DynamicMultiplierBasisPoints field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDynamicMultiplierBasisPoints

`func (o *OutboundFee) SetDynamicMultiplierBasisPoints(v string)`

SetDynamicMultiplierBasisPoints sets DynamicMultiplierBasisPoints field to given value.

### HasDynamicMultiplierBasisPoints

`func (o *OutboundFee) HasDynamicMultiplierBasisPoints() bool`

HasDynamicMultiplierBasisPoints returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


