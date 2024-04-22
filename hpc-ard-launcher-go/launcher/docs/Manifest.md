# Manifest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ManifestVersion** | **string** |  | 
**ClientMetadata** | [**ClientMetadata**](ClientMetadata.md) |  | 
**WarehouseMetadata** | Pointer to [**WarehouseMetadata**](WarehouseMetadata.md) |  | [optional] 
**DispatchMetadata** | Pointer to [**DispatchMetadata**](DispatchMetadata.md) |  | [optional] 
**Payloads** | Pointer to [**[]Payload**](Payload.md) |  | [optional] 
**SharedData** | Pointer to [**[]Data**](Data.md) |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewManifest

`func NewManifest(manifestVersion string, clientMetadata ClientMetadata, ) *Manifest`

NewManifest instantiates a new Manifest object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewManifestWithDefaults

`func NewManifestWithDefaults() *Manifest`

NewManifestWithDefaults instantiates a new Manifest object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetManifestVersion

`func (o *Manifest) GetManifestVersion() string`

GetManifestVersion returns the ManifestVersion field if non-nil, zero value otherwise.

### GetManifestVersionOk

`func (o *Manifest) GetManifestVersionOk() (*string, bool)`

GetManifestVersionOk returns a tuple with the ManifestVersion field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetManifestVersion

`func (o *Manifest) SetManifestVersion(v string)`

SetManifestVersion sets ManifestVersion field to given value.


### GetClientMetadata

`func (o *Manifest) GetClientMetadata() ClientMetadata`

GetClientMetadata returns the ClientMetadata field if non-nil, zero value otherwise.

### GetClientMetadataOk

`func (o *Manifest) GetClientMetadataOk() (*ClientMetadata, bool)`

GetClientMetadataOk returns a tuple with the ClientMetadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetClientMetadata

`func (o *Manifest) SetClientMetadata(v ClientMetadata)`

SetClientMetadata sets ClientMetadata field to given value.


### GetWarehouseMetadata

`func (o *Manifest) GetWarehouseMetadata() WarehouseMetadata`

GetWarehouseMetadata returns the WarehouseMetadata field if non-nil, zero value otherwise.

### GetWarehouseMetadataOk

`func (o *Manifest) GetWarehouseMetadataOk() (*WarehouseMetadata, bool)`

GetWarehouseMetadataOk returns a tuple with the WarehouseMetadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWarehouseMetadata

`func (o *Manifest) SetWarehouseMetadata(v WarehouseMetadata)`

SetWarehouseMetadata sets WarehouseMetadata field to given value.

### HasWarehouseMetadata

`func (o *Manifest) HasWarehouseMetadata() bool`

HasWarehouseMetadata returns a boolean if a field has been set.

### GetDispatchMetadata

`func (o *Manifest) GetDispatchMetadata() DispatchMetadata`

GetDispatchMetadata returns the DispatchMetadata field if non-nil, zero value otherwise.

### GetDispatchMetadataOk

`func (o *Manifest) GetDispatchMetadataOk() (*DispatchMetadata, bool)`

GetDispatchMetadataOk returns a tuple with the DispatchMetadata field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDispatchMetadata

`func (o *Manifest) SetDispatchMetadata(v DispatchMetadata)`

SetDispatchMetadata sets DispatchMetadata field to given value.

### HasDispatchMetadata

`func (o *Manifest) HasDispatchMetadata() bool`

HasDispatchMetadata returns a boolean if a field has been set.

### GetPayloads

`func (o *Manifest) GetPayloads() []Payload`

GetPayloads returns the Payloads field if non-nil, zero value otherwise.

### GetPayloadsOk

`func (o *Manifest) GetPayloadsOk() (*[]Payload, bool)`

GetPayloadsOk returns a tuple with the Payloads field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPayloads

`func (o *Manifest) SetPayloads(v []Payload)`

SetPayloads sets Payloads field to given value.

### HasPayloads

`func (o *Manifest) HasPayloads() bool`

HasPayloads returns a boolean if a field has been set.

### GetSharedData

`func (o *Manifest) GetSharedData() []Data`

GetSharedData returns the SharedData field if non-nil, zero value otherwise.

### GetSharedDataOk

`func (o *Manifest) GetSharedDataOk() (*[]Data, bool)`

GetSharedDataOk returns a tuple with the SharedData field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSharedData

`func (o *Manifest) SetSharedData(v []Data)`

SetSharedData sets SharedData field to given value.

### HasSharedData

`func (o *Manifest) HasSharedData() bool`

HasSharedData returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *Manifest) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *Manifest) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *Manifest) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *Manifest) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


