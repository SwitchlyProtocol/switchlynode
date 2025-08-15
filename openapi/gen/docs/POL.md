# POL

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**SWITCHDeposited** | **string** | total amount of SWITCH deposited into the pools | 
**SWITCHWithdrawn** | **string** | total amount of SWITCH withdrawn from the pools | 
**Value** | **string** | total value of protocol&#39;s LP position in SWITCH value | 
**Pnl** | **string** | profit and loss of protocol owned liquidity | 
**CurrentDeposit** | **string** | current amount of rune deposited | 

## Methods

### NewPOL

`func NewPOL(runeDeposited string, runeWithdrawn string, value string, pnl string, currentDeposit string, ) *POL`

NewPOL instantiates a new POL object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPOLWithDefaults

`func NewPOLWithDefaults() *POL`

NewPOLWithDefaults instantiates a new POL object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSWITCHDeposited

`func (o *POL) GetSWITCHDeposited() string`

GetSWITCHDeposited returns the SWITCHDeposited field if non-nil, zero value otherwise.

### GetSWITCHDepositedOk

`func (o *POL) GetSWITCHDepositedOk() (*string, bool)`

GetSWITCHDepositedOk returns a tuple with the SWITCHDeposited field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSWITCHDeposited

`func (o *POL) SetSWITCHDeposited(v string)`

SetSWITCHDeposited sets SWITCHDeposited field to given value.


### GetSWITCHWithdrawn

`func (o *POL) GetSWITCHWithdrawn() string`

GetSWITCHWithdrawn returns the SWITCHWithdrawn field if non-nil, zero value otherwise.

### GetSWITCHWithdrawnOk

`func (o *POL) GetSWITCHWithdrawnOk() (*string, bool)`

GetSWITCHWithdrawnOk returns a tuple with the SWITCHWithdrawn field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSWITCHWithdrawn

`func (o *POL) SetSWITCHWithdrawn(v string)`

SetSWITCHWithdrawn sets SWITCHWithdrawn field to given value.


### GetValue

`func (o *POL) GetValue() string`

GetValue returns the Value field if non-nil, zero value otherwise.

### GetValueOk

`func (o *POL) GetValueOk() (*string, bool)`

GetValueOk returns a tuple with the Value field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetValue

`func (o *POL) SetValue(v string)`

SetValue sets Value field to given value.


### GetPnl

`func (o *POL) GetPnl() string`

GetPnl returns the Pnl field if non-nil, zero value otherwise.

### GetPnlOk

`func (o *POL) GetPnlOk() (*string, bool)`

GetPnlOk returns a tuple with the Pnl field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPnl

`func (o *POL) SetPnl(v string)`

SetPnl sets Pnl field to given value.


### GetCurrentDeposit

`func (o *POL) GetCurrentDeposit() string`

GetCurrentDeposit returns the CurrentDeposit field if non-nil, zero value otherwise.

### GetCurrentDepositOk

`func (o *POL) GetCurrentDepositOk() (*string, bool)`

GetCurrentDepositOk returns a tuple with the CurrentDeposit field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCurrentDeposit

`func (o *POL) SetCurrentDeposit(v string)`

SetCurrentDeposit sets CurrentDeposit field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


