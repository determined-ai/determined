# DispatchMetadata

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Owner** | Pointer to **string** |  | [optional] 
**Dispatcher** | Pointer to **string** |  | [optional] 
**Carriers** | Pointer to **map[string]string** |  | [optional] 
**Launched** | Pointer to **string** |  | [optional] 
**Terminated** | Pointer to **string** |  | [optional] 
**UserInterfaces** | Pointer to [**[]UserInterface**](UserInterface.md) |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewDispatchMetadata

`func NewDispatchMetadata() *DispatchMetadata`

NewDispatchMetadata instantiates a new DispatchMetadata object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewDispatchMetadataWithDefaults

`func NewDispatchMetadataWithDefaults() *DispatchMetadata`

NewDispatchMetadataWithDefaults instantiates a new DispatchMetadata object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetOwner

`func (o *DispatchMetadata) GetOwner() string`

GetOwner returns the Owner field if non-nil, zero value otherwise.

### GetOwnerOk

`func (o *DispatchMetadata) GetOwnerOk() (*string, bool)`

GetOwnerOk returns a tuple with the Owner field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetOwner

`func (o *DispatchMetadata) SetOwner(v string)`

SetOwner sets Owner field to given value.

### HasOwner

`func (o *DispatchMetadata) HasOwner() bool`

HasOwner returns a boolean if a field has been set.

### GetDispatcher

`func (o *DispatchMetadata) GetDispatcher() string`

GetDispatcher returns the Dispatcher field if non-nil, zero value otherwise.

### GetDispatcherOk

`func (o *DispatchMetadata) GetDispatcherOk() (*string, bool)`

GetDispatcherOk returns a tuple with the Dispatcher field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDispatcher

`func (o *DispatchMetadata) SetDispatcher(v string)`

SetDispatcher sets Dispatcher field to given value.

### HasDispatcher

`func (o *DispatchMetadata) HasDispatcher() bool`

HasDispatcher returns a boolean if a field has been set.

### GetCarriers

`func (o *DispatchMetadata) GetCarriers() map[string]string`

GetCarriers returns the Carriers field if non-nil, zero value otherwise.

### GetCarriersOk

`func (o *DispatchMetadata) GetCarriersOk() (*map[string]string, bool)`

GetCarriersOk returns a tuple with the Carriers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCarriers

`func (o *DispatchMetadata) SetCarriers(v map[string]string)`

SetCarriers sets Carriers field to given value.

### HasCarriers

`func (o *DispatchMetadata) HasCarriers() bool`

HasCarriers returns a boolean if a field has been set.

### GetLaunched

`func (o *DispatchMetadata) GetLaunched() string`

GetLaunched returns the Launched field if non-nil, zero value otherwise.

### GetLaunchedOk

`func (o *DispatchMetadata) GetLaunchedOk() (*string, bool)`

GetLaunchedOk returns a tuple with the Launched field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLaunched

`func (o *DispatchMetadata) SetLaunched(v string)`

SetLaunched sets Launched field to given value.

### HasLaunched

`func (o *DispatchMetadata) HasLaunched() bool`

HasLaunched returns a boolean if a field has been set.

### GetTerminated

`func (o *DispatchMetadata) GetTerminated() string`

GetTerminated returns the Terminated field if non-nil, zero value otherwise.

### GetTerminatedOk

`func (o *DispatchMetadata) GetTerminatedOk() (*string, bool)`

GetTerminatedOk returns a tuple with the Terminated field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTerminated

`func (o *DispatchMetadata) SetTerminated(v string)`

SetTerminated sets Terminated field to given value.

### HasTerminated

`func (o *DispatchMetadata) HasTerminated() bool`

HasTerminated returns a boolean if a field has been set.

### GetUserInterfaces

`func (o *DispatchMetadata) GetUserInterfaces() []UserInterface`

GetUserInterfaces returns the UserInterfaces field if non-nil, zero value otherwise.

### GetUserInterfacesOk

`func (o *DispatchMetadata) GetUserInterfacesOk() (*[]UserInterface, bool)`

GetUserInterfacesOk returns a tuple with the UserInterfaces field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetUserInterfaces

`func (o *DispatchMetadata) SetUserInterfaces(v []UserInterface)`

SetUserInterfaces sets UserInterfaces field to given value.

### HasUserInterfaces

`func (o *DispatchMetadata) HasUserInterfaces() bool`

HasUserInterfaces returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *DispatchMetadata) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *DispatchMetadata) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *DispatchMetadata) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *DispatchMetadata) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


