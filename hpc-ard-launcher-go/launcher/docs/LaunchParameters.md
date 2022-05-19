# LaunchParameters

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Mode** | Pointer to **string** |  | [optional] 
**Environment** | Pointer to **map[string]string** |  | [optional] 
**Configuration** | Pointer to **map[string]string** |  | [optional] 
**Data** | Pointer to [**[]Data**](Data.md) |  | [optional] 
**Images** | Pointer to **map[string]string** |  | [optional] 
**Dependencies** | Pointer to **[]string** |  | [optional] 
**Arguments** | Pointer to **[]string** |  | [optional] 
**Custom** | Pointer to **map[string][]string** |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewLaunchParameters

`func NewLaunchParameters() *LaunchParameters`

NewLaunchParameters instantiates a new LaunchParameters object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewLaunchParametersWithDefaults

`func NewLaunchParametersWithDefaults() *LaunchParameters`

NewLaunchParametersWithDefaults instantiates a new LaunchParameters object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetMode

`func (o *LaunchParameters) GetMode() string`

GetMode returns the Mode field if non-nil, zero value otherwise.

### GetModeOk

`func (o *LaunchParameters) GetModeOk() (*string, bool)`

GetModeOk returns a tuple with the Mode field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMode

`func (o *LaunchParameters) SetMode(v string)`

SetMode sets Mode field to given value.

### HasMode

`func (o *LaunchParameters) HasMode() bool`

HasMode returns a boolean if a field has been set.

### GetEnvironment

`func (o *LaunchParameters) GetEnvironment() map[string]string`

GetEnvironment returns the Environment field if non-nil, zero value otherwise.

### GetEnvironmentOk

`func (o *LaunchParameters) GetEnvironmentOk() (*map[string]string, bool)`

GetEnvironmentOk returns a tuple with the Environment field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEnvironment

`func (o *LaunchParameters) SetEnvironment(v map[string]string)`

SetEnvironment sets Environment field to given value.

### HasEnvironment

`func (o *LaunchParameters) HasEnvironment() bool`

HasEnvironment returns a boolean if a field has been set.

### GetConfiguration

`func (o *LaunchParameters) GetConfiguration() map[string]string`

GetConfiguration returns the Configuration field if non-nil, zero value otherwise.

### GetConfigurationOk

`func (o *LaunchParameters) GetConfigurationOk() (*map[string]string, bool)`

GetConfigurationOk returns a tuple with the Configuration field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetConfiguration

`func (o *LaunchParameters) SetConfiguration(v map[string]string)`

SetConfiguration sets Configuration field to given value.

### HasConfiguration

`func (o *LaunchParameters) HasConfiguration() bool`

HasConfiguration returns a boolean if a field has been set.

### GetData

`func (o *LaunchParameters) GetData() []Data`

GetData returns the Data field if non-nil, zero value otherwise.

### GetDataOk

`func (o *LaunchParameters) GetDataOk() (*[]Data, bool)`

GetDataOk returns a tuple with the Data field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetData

`func (o *LaunchParameters) SetData(v []Data)`

SetData sets Data field to given value.

### HasData

`func (o *LaunchParameters) HasData() bool`

HasData returns a boolean if a field has been set.

### GetImages

`func (o *LaunchParameters) GetImages() map[string]string`

GetImages returns the Images field if non-nil, zero value otherwise.

### GetImagesOk

`func (o *LaunchParameters) GetImagesOk() (*map[string]string, bool)`

GetImagesOk returns a tuple with the Images field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetImages

`func (o *LaunchParameters) SetImages(v map[string]string)`

SetImages sets Images field to given value.

### HasImages

`func (o *LaunchParameters) HasImages() bool`

HasImages returns a boolean if a field has been set.

### GetDependencies

`func (o *LaunchParameters) GetDependencies() []string`

GetDependencies returns the Dependencies field if non-nil, zero value otherwise.

### GetDependenciesOk

`func (o *LaunchParameters) GetDependenciesOk() (*[]string, bool)`

GetDependenciesOk returns a tuple with the Dependencies field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDependencies

`func (o *LaunchParameters) SetDependencies(v []string)`

SetDependencies sets Dependencies field to given value.

### HasDependencies

`func (o *LaunchParameters) HasDependencies() bool`

HasDependencies returns a boolean if a field has been set.

### GetArguments

`func (o *LaunchParameters) GetArguments() []string`

GetArguments returns the Arguments field if non-nil, zero value otherwise.

### GetArgumentsOk

`func (o *LaunchParameters) GetArgumentsOk() (*[]string, bool)`

GetArgumentsOk returns a tuple with the Arguments field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetArguments

`func (o *LaunchParameters) SetArguments(v []string)`

SetArguments sets Arguments field to given value.

### HasArguments

`func (o *LaunchParameters) HasArguments() bool`

HasArguments returns a boolean if a field has been set.

### GetCustom

`func (o *LaunchParameters) GetCustom() map[string][]string`

GetCustom returns the Custom field if non-nil, zero value otherwise.

### GetCustomOk

`func (o *LaunchParameters) GetCustomOk() (*map[string][]string, bool)`

GetCustomOk returns a tuple with the Custom field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetCustom

`func (o *LaunchParameters) SetCustom(v map[string][]string)`

SetCustom sets Custom field to given value.

### HasCustom

`func (o *LaunchParameters) HasCustom() bool`

HasCustom returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *LaunchParameters) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *LaunchParameters) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *LaunchParameters) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *LaunchParameters) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


