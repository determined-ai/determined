# DispatchInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**LaunchedCapsuleReference** | Pointer to [**OwnedResourceReference**](OwnedResourceReference.md) |  | [optional] 
**LaunchingUser** | Pointer to **string** |  | [optional] 
**DispatchId** | Pointer to **string** |  | [optional] 
**State** | Pointer to [**DispatchState**](DispatchState.md) |  | [optional] 
**Events** | Pointer to [**[]Event**](Event.md) |  | [optional] 
**LastUpdated** | Pointer to **string** |  | [optional] 
**PayloadStates** | Pointer to [**map[string]DispatchState**](DispatchState.md) |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewDispatchInfo

`func NewDispatchInfo() *DispatchInfo`

NewDispatchInfo instantiates a new DispatchInfo object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDispatchInfoWithDefaults

`func NewDispatchInfoWithDefaults() *DispatchInfo`

NewDispatchInfoWithDefaults instantiates a new DispatchInfo object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLaunchedCapsuleReference

`func (o *DispatchInfo) GetLaunchedCapsuleReference() OwnedResourceReference`

GetLaunchedCapsuleReference returns the LaunchedCapsuleReference field if non-nil, zero value otherwise.

### GetLaunchedCapsuleReferenceOk

`func (o *DispatchInfo) GetLaunchedCapsuleReferenceOk() (*OwnedResourceReference, bool)`

GetLaunchedCapsuleReferenceOk returns a tuple with the LaunchedCapsuleReference field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLaunchedCapsuleReference

`func (o *DispatchInfo) SetLaunchedCapsuleReference(v OwnedResourceReference)`

SetLaunchedCapsuleReference sets LaunchedCapsuleReference field to given value.

### HasLaunchedCapsuleReference

`func (o *DispatchInfo) HasLaunchedCapsuleReference() bool`

HasLaunchedCapsuleReference returns a boolean if a field has been set.

### GetLaunchingUser

`func (o *DispatchInfo) GetLaunchingUser() string`

GetLaunchingUser returns the LaunchingUser field if non-nil, zero value otherwise.

### GetLaunchingUserOk

`func (o *DispatchInfo) GetLaunchingUserOk() (*string, bool)`

GetLaunchingUserOk returns a tuple with the LaunchingUser field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLaunchingUser

`func (o *DispatchInfo) SetLaunchingUser(v string)`

SetLaunchingUser sets LaunchingUser field to given value.

### HasLaunchingUser

`func (o *DispatchInfo) HasLaunchingUser() bool`

HasLaunchingUser returns a boolean if a field has been set.

### GetDispatchId

`func (o *DispatchInfo) GetDispatchId() string`

GetDispatchId returns the DispatchId field if non-nil, zero value otherwise.

### GetDispatchIdOk

`func (o *DispatchInfo) GetDispatchIdOk() (*string, bool)`

GetDispatchIdOk returns a tuple with the DispatchId field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDispatchId

`func (o *DispatchInfo) SetDispatchId(v string)`

SetDispatchId sets DispatchId field to given value.

### HasDispatchId

`func (o *DispatchInfo) HasDispatchId() bool`

HasDispatchId returns a boolean if a field has been set.

### GetState

`func (o *DispatchInfo) GetState() DispatchState`

GetState returns the State field if non-nil, zero value otherwise.

### GetStateOk

`func (o *DispatchInfo) GetStateOk() (*DispatchState, bool)`

GetStateOk returns a tuple with the State field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetState

`func (o *DispatchInfo) SetState(v DispatchState)`

SetState sets State field to given value.

### HasState

`func (o *DispatchInfo) HasState() bool`

HasState returns a boolean if a field has been set.

### GetEvents

`func (o *DispatchInfo) GetEvents() []Event`

GetEvents returns the Events field if non-nil, zero value otherwise.

### GetEventsOk

`func (o *DispatchInfo) GetEventsOk() (*[]Event, bool)`

GetEventsOk returns a tuple with the Events field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEvents

`func (o *DispatchInfo) SetEvents(v []Event)`

SetEvents sets Events field to given value.

### HasEvents

`func (o *DispatchInfo) HasEvents() bool`

HasEvents returns a boolean if a field has been set.

### GetLastUpdated

`func (o *DispatchInfo) GetLastUpdated() string`

GetLastUpdated returns the LastUpdated field if non-nil, zero value otherwise.

### GetLastUpdatedOk

`func (o *DispatchInfo) GetLastUpdatedOk() (*string, bool)`

GetLastUpdatedOk returns a tuple with the LastUpdated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLastUpdated

`func (o *DispatchInfo) SetLastUpdated(v string)`

SetLastUpdated sets LastUpdated field to given value.

### HasLastUpdated

`func (o *DispatchInfo) HasLastUpdated() bool`

HasLastUpdated returns a boolean if a field has been set.

### GetPayloadStates

`func (o *DispatchInfo) GetPayloadStates() map[string]DispatchState`

GetPayloadStates returns the PayloadStates field if non-nil, zero value otherwise.

### GetPayloadStatesOk

`func (o *DispatchInfo) GetPayloadStatesOk() (*map[string]DispatchState, bool)`

GetPayloadStatesOk returns a tuple with the PayloadStates field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayloadStates

`func (o *DispatchInfo) SetPayloadStates(v map[string]DispatchState)`

SetPayloadStates sets PayloadStates field to given value.

### HasPayloadStates

`func (o *DispatchInfo) HasPayloadStates() bool`

HasPayloadStates returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *DispatchInfo) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *DispatchInfo) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *DispatchInfo) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *DispatchInfo) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


