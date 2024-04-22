# Payload

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | Pointer to **string** |  | [optional] 
**Id** | Pointer to **string** |  | [optional] 
**Version** | Pointer to **string** |  | [optional] 
**Carriers** | Pointer to **[]string** |  | [optional] 
**LaunchParameters** | Pointer to [**LaunchParameters**](LaunchParameters.md) |  | [optional] 
**ResourceRequirements** | Pointer to [**ResourceRequirements**](ResourceRequirements.md) |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewPayload

`func NewPayload() *Payload`

NewPayload instantiates a new Payload object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewPayloadWithDefaults

`func NewPayloadWithDefaults() *Payload`

NewPayloadWithDefaults instantiates a new Payload object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *Payload) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *Payload) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *Payload) SetName(v string)`

SetName sets Name field to given value.

### HasName

`func (o *Payload) HasName() bool`

HasName returns a boolean if a field has been set.

### GetId

`func (o *Payload) GetId() string`

GetId returns the Id field if non-nil, zero value otherwise.

### GetIdOk

`func (o *Payload) GetIdOk() (*string, bool)`

GetIdOk returns a tuple with the Id field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetId

`func (o *Payload) SetId(v string)`

SetId sets Id field to given value.

### HasId

`func (o *Payload) HasId() bool`

HasId returns a boolean if a field has been set.

### GetVersion

`func (o *Payload) GetVersion() string`

GetVersion returns the Version field if non-nil, zero value otherwise.

### GetVersionOk

`func (o *Payload) GetVersionOk() (*string, bool)`

GetVersionOk returns a tuple with the Version field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetVersion

`func (o *Payload) SetVersion(v string)`

SetVersion sets Version field to given value.

### HasVersion

`func (o *Payload) HasVersion() bool`

HasVersion returns a boolean if a field has been set.

### GetCarriers

`func (o *Payload) GetCarriers() []string`

GetCarriers returns the Carriers field if non-nil, zero value otherwise.

### GetCarriersOk

`func (o *Payload) GetCarriersOk() (*[]string, bool)`

GetCarriersOk returns a tuple with the Carriers field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCarriers

`func (o *Payload) SetCarriers(v []string)`

SetCarriers sets Carriers field to given value.

### HasCarriers

`func (o *Payload) HasCarriers() bool`

HasCarriers returns a boolean if a field has been set.

### GetLaunchParameters

`func (o *Payload) GetLaunchParameters() LaunchParameters`

GetLaunchParameters returns the LaunchParameters field if non-nil, zero value otherwise.

### GetLaunchParametersOk

`func (o *Payload) GetLaunchParametersOk() (*LaunchParameters, bool)`

GetLaunchParametersOk returns a tuple with the LaunchParameters field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLaunchParameters

`func (o *Payload) SetLaunchParameters(v LaunchParameters)`

SetLaunchParameters sets LaunchParameters field to given value.

### HasLaunchParameters

`func (o *Payload) HasLaunchParameters() bool`

HasLaunchParameters returns a boolean if a field has been set.

### GetResourceRequirements

`func (o *Payload) GetResourceRequirements() ResourceRequirements`

GetResourceRequirements returns the ResourceRequirements field if non-nil, zero value otherwise.

### GetResourceRequirementsOk

`func (o *Payload) GetResourceRequirementsOk() (*ResourceRequirements, bool)`

GetResourceRequirementsOk returns a tuple with the ResourceRequirements field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetResourceRequirements

`func (o *Payload) SetResourceRequirements(v ResourceRequirements)`

SetResourceRequirements sets ResourceRequirements field to given value.

### HasResourceRequirements

`func (o *Payload) HasResourceRequirements() bool`

HasResourceRequirements returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *Payload) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *Payload) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *Payload) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *Payload) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


