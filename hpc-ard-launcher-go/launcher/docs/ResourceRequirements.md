# ResourceRequirements

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Instances** | Pointer to **map[string]int32** |  | [optional] 
**Cores** | Pointer to **map[string]float32** |  | [optional] 
**Memory** | Pointer to **map[string]string** |  | [optional] 
**Gpus** | Pointer to **map[string]int32** |  | [optional] 
**Scratch** | Pointer to **map[string]string** |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewResourceRequirements

`func NewResourceRequirements() *ResourceRequirements`

NewResourceRequirements instantiates a new ResourceRequirements object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewResourceRequirementsWithDefaults

`func NewResourceRequirementsWithDefaults() *ResourceRequirements`

NewResourceRequirementsWithDefaults instantiates a new ResourceRequirements object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetInstances

`func (o *ResourceRequirements) GetInstances() map[string]int32`

GetInstances returns the Instances field if non-nil, zero value otherwise.

### GetInstancesOk

`func (o *ResourceRequirements) GetInstancesOk() (*map[string]int32, bool)`

GetInstancesOk returns a tuple with the Instances field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInstances

`func (o *ResourceRequirements) SetInstances(v map[string]int32)`

SetInstances sets Instances field to given value.

### HasInstances

`func (o *ResourceRequirements) HasInstances() bool`

HasInstances returns a boolean if a field has been set.

### GetCores

`func (o *ResourceRequirements) GetCores() map[string]float32`

GetCores returns the Cores field if non-nil, zero value otherwise.

### GetCoresOk

`func (o *ResourceRequirements) GetCoresOk() (*map[string]float32, bool)`

GetCoresOk returns a tuple with the Cores field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCores

`func (o *ResourceRequirements) SetCores(v map[string]float32)`

SetCores sets Cores field to given value.

### HasCores

`func (o *ResourceRequirements) HasCores() bool`

HasCores returns a boolean if a field has been set.

### GetMemory

`func (o *ResourceRequirements) GetMemory() map[string]string`

GetMemory returns the Memory field if non-nil, zero value otherwise.

### GetMemoryOk

`func (o *ResourceRequirements) GetMemoryOk() (*map[string]string, bool)`

GetMemoryOk returns a tuple with the Memory field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMemory

`func (o *ResourceRequirements) SetMemory(v map[string]string)`

SetMemory sets Memory field to given value.

### HasMemory

`func (o *ResourceRequirements) HasMemory() bool`

HasMemory returns a boolean if a field has been set.

### GetGpus

`func (o *ResourceRequirements) GetGpus() map[string]int32`

GetGpus returns the Gpus field if non-nil, zero value otherwise.

### GetGpusOk

`func (o *ResourceRequirements) GetGpusOk() (*map[string]int32, bool)`

GetGpusOk returns a tuple with the Gpus field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetGpus

`func (o *ResourceRequirements) SetGpus(v map[string]int32)`

SetGpus sets Gpus field to given value.

### HasGpus

`func (o *ResourceRequirements) HasGpus() bool`

HasGpus returns a boolean if a field has been set.

### GetScratch

`func (o *ResourceRequirements) GetScratch() map[string]string`

GetScratch returns the Scratch field if non-nil, zero value otherwise.

### GetScratchOk

`func (o *ResourceRequirements) GetScratchOk() (*map[string]string, bool)`

GetScratchOk returns a tuple with the Scratch field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetScratch

`func (o *ResourceRequirements) SetScratch(v map[string]string)`

SetScratch sets Scratch field to given value.

### HasScratch

`func (o *ResourceRequirements) HasScratch() bool`

HasScratch returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *ResourceRequirements) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *ResourceRequirements) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *ResourceRequirements) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *ResourceRequirements) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


