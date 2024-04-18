# DispatchManagementStatus

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Info** | Pointer to [**DispatchInfo**](DispatchInfo.md) |  | [optional] 
**Dispatcher** | Pointer to **string** |  | [optional] 
**Reason** | Pointer to **string** |  | [optional] 
**CanManage** | Pointer to **bool** |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewDispatchManagementStatus

`func NewDispatchManagementStatus() *DispatchManagementStatus`

NewDispatchManagementStatus instantiates a new DispatchManagementStatus object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDispatchManagementStatusWithDefaults

`func NewDispatchManagementStatusWithDefaults() *DispatchManagementStatus`

NewDispatchManagementStatusWithDefaults instantiates a new DispatchManagementStatus object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetInfo

`func (o *DispatchManagementStatus) GetInfo() DispatchInfo`

GetInfo returns the Info field if non-nil, zero value otherwise.

### GetInfoOk

`func (o *DispatchManagementStatus) GetInfoOk() (*DispatchInfo, bool)`

GetInfoOk returns a tuple with the Info field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInfo

`func (o *DispatchManagementStatus) SetInfo(v DispatchInfo)`

SetInfo sets Info field to given value.

### HasInfo

`func (o *DispatchManagementStatus) HasInfo() bool`

HasInfo returns a boolean if a field has been set.

### GetDispatcher

`func (o *DispatchManagementStatus) GetDispatcher() string`

GetDispatcher returns the Dispatcher field if non-nil, zero value otherwise.

### GetDispatcherOk

`func (o *DispatchManagementStatus) GetDispatcherOk() (*string, bool)`

GetDispatcherOk returns a tuple with the Dispatcher field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDispatcher

`func (o *DispatchManagementStatus) SetDispatcher(v string)`

SetDispatcher sets Dispatcher field to given value.

### HasDispatcher

`func (o *DispatchManagementStatus) HasDispatcher() bool`

HasDispatcher returns a boolean if a field has been set.

### GetReason

`func (o *DispatchManagementStatus) GetReason() string`

GetReason returns the Reason field if non-nil, zero value otherwise.

### GetReasonOk

`func (o *DispatchManagementStatus) GetReasonOk() (*string, bool)`

GetReasonOk returns a tuple with the Reason field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReason

`func (o *DispatchManagementStatus) SetReason(v string)`

SetReason sets Reason field to given value.

### HasReason

`func (o *DispatchManagementStatus) HasReason() bool`

HasReason returns a boolean if a field has been set.

### GetCanManage

`func (o *DispatchManagementStatus) GetCanManage() bool`

GetCanManage returns the CanManage field if non-nil, zero value otherwise.

### GetCanManageOk

`func (o *DispatchManagementStatus) GetCanManageOk() (*bool, bool)`

GetCanManageOk returns a tuple with the CanManage field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCanManage

`func (o *DispatchManagementStatus) SetCanManage(v bool)`

SetCanManage sets CanManage field to given value.

### HasCanManage

`func (o *DispatchManagementStatus) HasCanManage() bool`

HasCanManage returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *DispatchManagementStatus) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *DispatchManagementStatus) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *DispatchManagementStatus) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *DispatchManagementStatus) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


