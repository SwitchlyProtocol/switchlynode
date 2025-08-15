# SWITCHPoolResponseProviders

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Units** | **string** | the units of SWITCHPool owned by providers (including pending) | 
**PendingUnits** | **string** | the units of SWITCHPool owned by providers that remain pending | 
**PendingSWITCH** | **string** | the amount of SWITCH pending | 
**Value** | **string** | the value of the provider share of the SWITCHPool (includes pending SWITCH) | 
**Pnl** | **string** | the profit and loss of the provider share of the SWITCHPool | 
**CurrentDeposit** | **string** | the current SWITCH deposited by providers | 

## Methods

### NewSWITCHPoolResponseProviders

`func NewSWITCHPoolResponseProviders(units string, pendingUnits string, pendingSWITCH string, value string, pnl string, currentDeposit string, ) *SWITCHPoolResponseProviders`

NewSWITCHPoolResponseProviders instantiates a new SWITCHPoolResponseProviders object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewSWITCHPoolResponseProvidersWithDefaults

`func NewSWITCHPoolResponseProvidersWithDefaults() *SWITCHPoolResponseProviders`

NewSWITCHPoolResponseProvidersWithDefaults instantiates a new SWITCHPoolResponseProviders object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetUnits

`func (o *SWITCHPoolResponseProviders) GetUnits() string`

GetUnits returns the Units field if non-nil, zero value otherwise.

### GetUnitsOk

`func (o *SWITCHPoolResponseProviders) GetUnitsOk() (*string, bool)`

GetUnitsOk returns a tuple with the Units field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUnits

`func (o *SWITCHPoolResponseProviders) SetUnits(v string)`

SetUnits sets Units field to given value.


### GetPendingUnits

`func (o *SWITCHPoolResponseProviders) GetPendingUnits() string`

GetPendingUnits returns the PendingUnits field if non-nil, zero value otherwise.

### GetPendingUnitsOk

`func (o *SWITCHPoolResponseProviders) GetPendingUnitsOk() (*string, bool)`

GetPendingUnitsOk returns a tuple with the PendingUnits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPendingUnits

`func (o *SWITCHPoolResponseProviders) SetPendingUnits(v string)`

SetPendingUnits sets PendingUnits field to given value.


### GetPendingSWITCH

`func (o *SWITCHPoolResponseProviders) GetPendingSWITCH() string`

GetPendingSWITCH returns the PendingSWITCH field if non-nil, zero value otherwise.

### GetPendingSWITCHOk

`func (o *SWITCHPoolResponseProviders) GetPendingSWITCHOk() (*string, bool)`

GetPendingSWITCHOk returns a tuple with the PendingSWITCH field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPendingSWITCH

`func (o *SWITCHPoolResponseProviders) SetPendingSWITCH(v string)`

SetPendingSWITCH sets PendingSWITCH field to given value.


### GetValue

`func (o *SWITCHPoolResponseProviders) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *SWITCHPoolResponseProviders) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *SWITCHPoolResponseProviders) SetValue(v string)`

SetValue sets Value field to given value.


### GetPnl

`func (o *SWITCHPoolResponseProviders) GetPnl() string`

GetPnl returns the Pnl field if non-nil, zero value otherwise.

### GetPnlOk

`func (o *SWITCHPoolResponseProviders) GetPnlOk() (*string, bool)`

GetPnlOk returns a tuple with the Pnl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPnl

`func (o *SWITCHPoolResponseProviders) SetPnl(v string)`

SetPnl sets Pnl field to given value.


### GetCurrentDeposit

`func (o *SWITCHPoolResponseProviders) GetCurrentDeposit() string`

GetCurrentDeposit returns the CurrentDeposit field if non-nil, zero value otherwise.

### GetCurrentDepositOk

`func (o *SWITCHPoolResponseProviders) GetCurrentDepositOk() (*string, bool)`

GetCurrentDepositOk returns a tuple with the CurrentDeposit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentDeposit

`func (o *SWITCHPoolResponseProviders) SetCurrentDeposit(v string)`

SetCurrentDeposit sets CurrentDeposit field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


