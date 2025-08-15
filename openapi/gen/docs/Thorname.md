# Thorname

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**ExpireBlockHeight** | Pointer to **int64** |  | [optional] 
**Owner** | Pointer to **string** |  | [optional] 
**PreferredAsset** | **string** |  | 
**PreferredAssetSwapThresholdSWITCH** | Pointer to **string** | Amount of SWITCH currently needed to trigger a preferred asset swap. | [optional] 
**AffiliateCollectorSWITCH** | Pointer to **string** | Amount of SWITCH currently accrued by this switchlyname in affiliate fees waiting to be swapped to preferred asset. | [optional] 
**Aliases** | [**[]ThornameAlias**](ThornameAlias.md) |  | 

## Methods

### NewThorname

`func NewThorname(preferredAsset string, aliases []ThornameAlias, ) *Thorname`

NewThorname instantiates a new Thorname object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewThornameWithDefaults

`func NewThornameWithDefaults() *Thorname`

NewThornameWithDefaults instantiates a new Thorname object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *Thorname) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Thorname) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Thorname) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *Thorname) HasName() bool`

HasName returns a boolean if a field has been set.

### GetExpireBlockHeight

`func (o *Thorname) GetExpireBlockHeight() int64`

GetExpireBlockHeight returns the ExpireBlockHeight field if non-nil, zero value otherwise.

### GetExpireBlockHeightOk

`func (o *Thorname) GetExpireBlockHeightOk() (*int64, bool)`

GetExpireBlockHeightOk returns a tuple with the ExpireBlockHeight field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExpireBlockHeight

`func (o *Thorname) SetExpireBlockHeight(v int64)`

SetExpireBlockHeight sets ExpireBlockHeight field to given value.

### HasExpireBlockHeight

`func (o *Thorname) HasExpireBlockHeight() bool`

HasExpireBlockHeight returns a boolean if a field has been set.

### GetOwner

`func (o *Thorname) GetOwner() string`

GetOwner returns the Owner field if non-nil, zero value otherwise.

### GetOwnerOk

`func (o *Thorname) GetOwnerOk() (*string, bool)`

GetOwnerOk returns a tuple with the Owner field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwner

`func (o *Thorname) SetOwner(v string)`

SetOwner sets Owner field to given value.

### HasOwner

`func (o *Thorname) HasOwner() bool`

HasOwner returns a boolean if a field has been set.

### GetPreferredAsset

`func (o *Thorname) GetPreferredAsset() string`

GetPreferredAsset returns the PreferredAsset field if non-nil, zero value otherwise.

### GetPreferredAssetOk

`func (o *Thorname) GetPreferredAssetOk() (*string, bool)`

GetPreferredAssetOk returns a tuple with the PreferredAsset field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreferredAsset

`func (o *Thorname) SetPreferredAsset(v string)`

SetPreferredAsset sets PreferredAsset field to given value.


### GetPreferredAssetSwapThresholdSWITCH

`func (o *Thorname) GetPreferredAssetSwapThresholdSWITCH() string`

GetPreferredAssetSwapThresholdSWITCH returns the PreferredAssetSwapThresholdSWITCH field if non-nil, zero value otherwise.

### GetPreferredAssetSwapThresholdSWITCHOk

`func (o *Thorname) GetPreferredAssetSwapThresholdSWITCHOk() (*string, bool)`

GetPreferredAssetSwapThresholdSWITCHOk returns a tuple with the PreferredAssetSwapThresholdSWITCH field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPreferredAssetSwapThresholdSWITCH

`func (o *Thorname) SetPreferredAssetSwapThresholdSWITCH(v string)`

SetPreferredAssetSwapThresholdSWITCH sets PreferredAssetSwapThresholdSWITCH field to given value.

### HasPreferredAssetSwapThresholdSWITCH

`func (o *Thorname) HasPreferredAssetSwapThresholdSWITCH() bool`

HasPreferredAssetSwapThresholdSWITCH returns a boolean if a field has been set.

### GetAffiliateCollectorSWITCH

`func (o *Thorname) GetAffiliateCollectorSWITCH() string`

GetAffiliateCollectorSWITCH returns the AffiliateCollectorSWITCH field if non-nil, zero value otherwise.

### GetAffiliateCollectorSWITCHOk

`func (o *Thorname) GetAffiliateCollectorSWITCHOk() (*string, bool)`

GetAffiliateCollectorSWITCHOk returns a tuple with the AffiliateCollectorSWITCH field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAffiliateCollectorSWITCH

`func (o *Thorname) SetAffiliateCollectorSWITCH(v string)`

SetAffiliateCollectorSWITCH sets AffiliateCollectorSWITCH field to given value.

### HasAffiliateCollectorSWITCH

`func (o *Thorname) HasAffiliateCollectorSWITCH() bool`

HasAffiliateCollectorSWITCH returns a boolean if a field has been set.

### GetAliases

`func (o *Thorname) GetAliases() []ThornameAlias`

GetAliases returns the Aliases field if non-nil, zero value otherwise.

### GetAliasesOk

`func (o *Thorname) GetAliasesOk() (*[]ThornameAlias, bool)`

GetAliasesOk returns a tuple with the Aliases field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAliases

`func (o *Thorname) SetAliases(v []ThornameAlias)`

SetAliases sets Aliases field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


