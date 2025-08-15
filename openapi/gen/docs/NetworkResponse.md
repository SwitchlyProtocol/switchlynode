# NetworkResponse

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**BondRewardSwitch** | **string** | total amount of SWITCH awarded to node operators | 
**TotalBondUnits** | **string** | total bonded SWITCH | 
**AvailablePoolsSwitch** | **string** | SWITCH in Available pools (equal in value to the Assets in those pools) | 
**VaultsLiquiditySwitch** | **string** | SWITCH value of Layer 1 Assets in vaults | 
**EffectiveSecurityBond** | **string** | effective security bond used to determine maximum pooled SWITCH | 
**TotalReserve** | **string** | total reserve SWITCH | 
**VaultsMigrating** | **bool** | Returns true if there exist RetiringVaults which have not finished migrating funds to new ActiveVaults | 
**GasSpentSwitch** | **string** | Sum of the gas the network has spent to send outbounds | 
**GasWithheldSwitch** | **string** | Sum of the gas withheld from users to cover outbound gas | 
**OutboundFeeMultiplier** | Pointer to **string** | Current outbound fee multiplier, in basis points | [optional] 
**NativeOutboundFeeSwitch** | **string** | the outbound transaction fee in switch, converted from the NativeOutboundFeeUSD mimir (after USD fees are enabled) | 
**NativeTxFeeSwitch** | **string** | the native transaction fee in switch, converted from the NativeTransactionFeeUSD mimir (after USD fees are enabled) | 
**SwitchlynameRegisterFeeSwitch** | **string** | the switchlyname register fee in switch, converted from the TNSRegisterFeeUSD mimir (after USD fees are enabled) | 
**SwitchlynameFeePerBlockSwitch** | **string** | the switchlyname fee per block in switch, converted from the TNSFeePerBlockUSD mimir (after USD fees are enabled) | 
**SwitchPriceInSwitchly** | **string** | the switch price in tor | 
**SwitchlyPriceInSwitch** | **string** | the tor price in switch | 
**SwitchlyPriceHalted** | **bool** | indicates if all anchor chains are halted (true), or at least one anchor chain is available (false) | 

## Methods

### NewNetworkResponse

`func NewNetworkResponse(bondRewardSwitch string, totalBondUnits string, availablePoolsSwitch string, vaultsLiquiditySwitch string, effectiveSecurityBond string, totalReserve string, vaultsMigrating bool, gasSpentSwitch string, gasWithheldSwitch string, nativeOutboundFeeSwitch string, nativeTxFeeSwitch string, switchlynameRegisterFeeSwitch string, switchlynameFeePerBlockSwitch string, switchPriceInSwitchly string, torPriceInSwitch string, torPriceHalted bool, ) *NetworkResponse`

NewNetworkResponse instantiates a new NetworkResponse object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewNetworkResponseWithDefaults

`func NewNetworkResponseWithDefaults() *NetworkResponse`

NewNetworkResponseWithDefaults instantiates a new NetworkResponse object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetBondRewardSwitch

`func (o *NetworkResponse) GetBondRewardSwitch() string`

GetBondRewardSwitch returns the BondRewardSwitch field if non-nil, zero value otherwise.

### GetBondRewardSwitchOk

`func (o *NetworkResponse) GetBondRewardSwitchOk() (*string, bool)`

GetBondRewardSwitchOk returns a tuple with the BondRewardSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetBondRewardSwitch

`func (o *NetworkResponse) SetBondRewardSwitch(v string)`

SetBondRewardSwitch sets BondRewardSwitch field to given value.


### GetTotalBondUnits

`func (o *NetworkResponse) GetTotalBondUnits() string`

GetTotalBondUnits returns the TotalBondUnits field if non-nil, zero value otherwise.

### GetTotalBondUnitsOk

`func (o *NetworkResponse) GetTotalBondUnitsOk() (*string, bool)`

GetTotalBondUnitsOk returns a tuple with the TotalBondUnits field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalBondUnits

`func (o *NetworkResponse) SetTotalBondUnits(v string)`

SetTotalBondUnits sets TotalBondUnits field to given value.


### GetAvailablePoolsSwitch

`func (o *NetworkResponse) GetAvailablePoolsSwitch() string`

GetAvailablePoolsSwitch returns the AvailablePoolsSwitch field if non-nil, zero value otherwise.

### GetAvailablePoolsSwitchOk

`func (o *NetworkResponse) GetAvailablePoolsSwitchOk() (*string, bool)`

GetAvailablePoolsSwitchOk returns a tuple with the AvailablePoolsSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAvailablePoolsSwitch

`func (o *NetworkResponse) SetAvailablePoolsSwitch(v string)`

SetAvailablePoolsSwitch sets AvailablePoolsSwitch field to given value.


### GetVaultsLiquiditySwitch

`func (o *NetworkResponse) GetVaultsLiquiditySwitch() string`

GetVaultsLiquiditySwitch returns the VaultsLiquiditySwitch field if non-nil, zero value otherwise.

### GetVaultsLiquiditySwitchOk

`func (o *NetworkResponse) GetVaultsLiquiditySwitchOk() (*string, bool)`

GetVaultsLiquiditySwitchOk returns a tuple with the VaultsLiquiditySwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVaultsLiquiditySwitch

`func (o *NetworkResponse) SetVaultsLiquiditySwitch(v string)`

SetVaultsLiquiditySwitch sets VaultsLiquiditySwitch field to given value.


### GetEffectiveSecurityBond

`func (o *NetworkResponse) GetEffectiveSecurityBond() string`

GetEffectiveSecurityBond returns the EffectiveSecurityBond field if non-nil, zero value otherwise.

### GetEffectiveSecurityBondOk

`func (o *NetworkResponse) GetEffectiveSecurityBondOk() (*string, bool)`

GetEffectiveSecurityBondOk returns a tuple with the EffectiveSecurityBond field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEffectiveSecurityBond

`func (o *NetworkResponse) SetEffectiveSecurityBond(v string)`

SetEffectiveSecurityBond sets EffectiveSecurityBond field to given value.


### GetTotalReserve

`func (o *NetworkResponse) GetTotalReserve() string`

GetTotalReserve returns the TotalReserve field if non-nil, zero value otherwise.

### GetTotalReserveOk

`func (o *NetworkResponse) GetTotalReserveOk() (*string, bool)`

GetTotalReserveOk returns a tuple with the TotalReserve field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTotalReserve

`func (o *NetworkResponse) SetTotalReserve(v string)`

SetTotalReserve sets TotalReserve field to given value.


### GetVaultsMigrating

`func (o *NetworkResponse) GetVaultsMigrating() bool`

GetVaultsMigrating returns the VaultsMigrating field if non-nil, zero value otherwise.

### GetVaultsMigratingOk

`func (o *NetworkResponse) GetVaultsMigratingOk() (*bool, bool)`

GetVaultsMigratingOk returns a tuple with the VaultsMigrating field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVaultsMigrating

`func (o *NetworkResponse) SetVaultsMigrating(v bool)`

SetVaultsMigrating sets VaultsMigrating field to given value.


### GetGasSpentSwitch

`func (o *NetworkResponse) GetGasSpentSwitch() string`

GetGasSpentSwitch returns the GasSpentSwitch field if non-nil, zero value otherwise.

### GetGasSpentSwitchOk

`func (o *NetworkResponse) GetGasSpentSwitchOk() (*string, bool)`

GetGasSpentSwitchOk returns a tuple with the GasSpentSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGasSpentSwitch

`func (o *NetworkResponse) SetGasSpentSwitch(v string)`

SetGasSpentSwitch sets GasSpentSwitch field to given value.


### GetGasWithheldSwitch

`func (o *NetworkResponse) GetGasWithheldSwitch() string`

GetGasWithheldSwitch returns the GasWithheldSwitch field if non-nil, zero value otherwise.

### GetGasWithheldSwitchOk

`func (o *NetworkResponse) GetGasWithheldSwitchOk() (*string, bool)`

GetGasWithheldSwitchOk returns a tuple with the GasWithheldSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGasWithheldSwitch

`func (o *NetworkResponse) SetGasWithheldSwitch(v string)`

SetGasWithheldSwitch sets GasWithheldSwitch field to given value.


### GetOutboundFeeMultiplier

`func (o *NetworkResponse) GetOutboundFeeMultiplier() string`

GetOutboundFeeMultiplier returns the OutboundFeeMultiplier field if non-nil, zero value otherwise.

### GetOutboundFeeMultiplierOk

`func (o *NetworkResponse) GetOutboundFeeMultiplierOk() (*string, bool)`

GetOutboundFeeMultiplierOk returns a tuple with the OutboundFeeMultiplier field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOutboundFeeMultiplier

`func (o *NetworkResponse) SetOutboundFeeMultiplier(v string)`

SetOutboundFeeMultiplier sets OutboundFeeMultiplier field to given value.

### HasOutboundFeeMultiplier

`func (o *NetworkResponse) HasOutboundFeeMultiplier() bool`

HasOutboundFeeMultiplier returns a boolean if a field has been set.

### GetNativeOutboundFeeSwitch

`func (o *NetworkResponse) GetNativeOutboundFeeSwitch() string`

GetNativeOutboundFeeSwitch returns the NativeOutboundFeeSwitch field if non-nil, zero value otherwise.

### GetNativeOutboundFeeSwitchOk

`func (o *NetworkResponse) GetNativeOutboundFeeSwitchOk() (*string, bool)`

GetNativeOutboundFeeSwitchOk returns a tuple with the NativeOutboundFeeSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNativeOutboundFeeSwitch

`func (o *NetworkResponse) SetNativeOutboundFeeSwitch(v string)`

SetNativeOutboundFeeSwitch sets NativeOutboundFeeSwitch field to given value.


### GetNativeTxFeeSwitch

`func (o *NetworkResponse) GetNativeTxFeeSwitch() string`

GetNativeTxFeeSwitch returns the NativeTxFeeSwitch field if non-nil, zero value otherwise.

### GetNativeTxFeeSwitchOk

`func (o *NetworkResponse) GetNativeTxFeeSwitchOk() (*string, bool)`

GetNativeTxFeeSwitchOk returns a tuple with the NativeTxFeeSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetNativeTxFeeSwitch

`func (o *NetworkResponse) SetNativeTxFeeSwitch(v string)`

SetNativeTxFeeSwitch sets NativeTxFeeSwitch field to given value.


### GetSwitchlynameRegisterFeeSwitch

`func (o *NetworkResponse) GetSwitchlynameRegisterFeeSwitch() string`

GetSwitchlynameRegisterFeeSwitch returns the SwitchlynameRegisterFeeSwitch field if non-nil, zero value otherwise.

### GetSwitchlynameRegisterFeeSwitchOk

`func (o *NetworkResponse) GetSwitchlynameRegisterFeeSwitchOk() (*string, bool)`

GetSwitchlynameRegisterFeeSwitchOk returns a tuple with the SwitchlynameRegisterFeeSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwitchlynameRegisterFeeSwitch

`func (o *NetworkResponse) SetSwitchlynameRegisterFeeSwitch(v string)`

SetSwitchlynameRegisterFeeSwitch sets SwitchlynameRegisterFeeSwitch field to given value.


### GetSwitchlynameFeePerBlockSwitch

`func (o *NetworkResponse) GetSwitchlynameFeePerBlockSwitch() string`

GetSwitchlynameFeePerBlockSwitch returns the SwitchlynameFeePerBlockSwitch field if non-nil, zero value otherwise.

### GetSwitchlynameFeePerBlockSwitchOk

`func (o *NetworkResponse) GetSwitchlynameFeePerBlockSwitchOk() (*string, bool)`

GetSwitchlynameFeePerBlockSwitchOk returns a tuple with the SwitchlynameFeePerBlockSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwitchlynameFeePerBlockSwitch

`func (o *NetworkResponse) SetSwitchlynameFeePerBlockSwitch(v string)`

SetSwitchlynameFeePerBlockSwitch sets SwitchlynameFeePerBlockSwitch field to given value.


### GetSwitchPriceInSwitchly

`func (o *NetworkResponse) GetSwitchPriceInSwitchly() string`

GetSwitchPriceInSwitchly returns the SwitchPriceInSwitchly field if non-nil, zero value otherwise.

### GetSwitchPriceInSwitchlyOk

`func (o *NetworkResponse) GetSwitchPriceInSwitchlyOk() (*string, bool)`

GetSwitchPriceInSwitchlyOk returns a tuple with the SwitchPriceInSwitchly field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwitchPriceInSwitchly

`func (o *NetworkResponse) SetSwitchPriceInSwitchly(v string)`

SetSwitchPriceInSwitchly sets SwitchPriceInSwitchly field to given value.


### GetSwitchlyPriceInSwitch

`func (o *NetworkResponse) GetSwitchlyPriceInSwitch() string`

GetSwitchlyPriceInSwitch returns the SwitchlyPriceInSwitch field if non-nil, zero value otherwise.

### GetSwitchlyPriceInSwitchOk

`func (o *NetworkResponse) GetSwitchlyPriceInSwitchOk() (*string, bool)`

GetSwitchlyPriceInSwitchOk returns a tuple with the SwitchlyPriceInSwitch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwitchlyPriceInSwitch

`func (o *NetworkResponse) SetSwitchlyPriceInSwitch(v string)`

SetSwitchlyPriceInSwitch sets SwitchlyPriceInSwitch field to given value.


### GetSwitchlyPriceHalted

`func (o *NetworkResponse) GetSwitchlyPriceHalted() bool`

GetSwitchlyPriceHalted returns the SwitchlyPriceHalted field if non-nil, zero value otherwise.

### GetSwitchlyPriceHaltedOk

`func (o *NetworkResponse) GetSwitchlyPriceHaltedOk() (*bool, bool)`

GetSwitchlyPriceHaltedOk returns a tuple with the SwitchlyPriceHalted field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSwitchlyPriceHalted

`func (o *NetworkResponse) SetSwitchlyPriceHalted(v bool)`

SetSwitchlyPriceHalted sets SwitchlyPriceHalted field to given value.



[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


