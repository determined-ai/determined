# ClientMetadata

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Title** | Pointer to **string** |  | [optional] 
**Description** | Pointer to **string** |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewClientMetadata

`func NewClientMetadata(name string, ) *ClientMetadata`

NewClientMetadata instantiates a new ClientMetadata object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewClientMetadataWithDefaults

`func NewClientMetadataWithDefaults() *ClientMetadata`

NewClientMetadataWithDefaults instantiates a new ClientMetadata object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ClientMetadata) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ClientMetadata) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ClientMetadata) SetName(v string)`

SetName sets Name field to given value.


### GetTitle

`func (o *ClientMetadata) GetTitle() string`

GetTitle returns the Title field if non-nil, zero value otherwise.

### GetTitleOk

`func (o *ClientMetadata) GetTitleOk() (*string, bool)`

GetTitleOk returns a tuple with the Title field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTitle

`func (o *ClientMetadata) SetTitle(v string)`

SetTitle sets Title field to given value.

### HasTitle

`func (o *ClientMetadata) HasTitle() bool`

HasTitle returns a boolean if a field has been set.

### GetDescription

`func (o *ClientMetadata) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *ClientMetadata) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *ClientMetadata) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *ClientMetadata) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *ClientMetadata) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *ClientMetadata) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *ClientMetadata) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *ClientMetadata) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


