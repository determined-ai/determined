# ACLS

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Read** | Pointer to [**ACLInfo**](ACLInfo.md) |  | [optional] 
**Write** | Pointer to [**ACLInfo**](ACLInfo.md) |  | [optional] 
**Admin** | Pointer to [**ACLInfo**](ACLInfo.md) |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewACLS

`func NewACLS() *ACLS`

NewACLS instantiates a new ACLS object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewACLSWithDefaults

`func NewACLSWithDefaults() *ACLS`

NewACLSWithDefaults instantiates a new ACLS object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetRead

`func (o *ACLS) GetRead() ACLInfo`

GetRead returns the Read field if non-nil, zero value otherwise.

### GetReadOk

`func (o *ACLS) GetReadOk() (*ACLInfo, bool)`

GetReadOk returns a tuple with the Read field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetRead

`func (o *ACLS) SetRead(v ACLInfo)`

SetRead sets Read field to given value.

### HasRead

`func (o *ACLS) HasRead() bool`

HasRead returns a boolean if a field has been set.

### GetWrite

`func (o *ACLS) GetWrite() ACLInfo`

GetWrite returns the Write field if non-nil, zero value otherwise.

### GetWriteOk

`func (o *ACLS) GetWriteOk() (*ACLInfo, bool)`

GetWriteOk returns a tuple with the Write field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWrite

`func (o *ACLS) SetWrite(v ACLInfo)`

SetWrite sets Write field to given value.

### HasWrite

`func (o *ACLS) HasWrite() bool`

HasWrite returns a boolean if a field has been set.

### GetAdmin

`func (o *ACLS) GetAdmin() ACLInfo`

GetAdmin returns the Admin field if non-nil, zero value otherwise.

### GetAdminOk

`func (o *ACLS) GetAdminOk() (*ACLInfo, bool)`

GetAdminOk returns a tuple with the Admin field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdmin

`func (o *ACLS) SetAdmin(v ACLInfo)`

SetAdmin sets Admin field to given value.

### HasAdmin

`func (o *ACLS) HasAdmin() bool`

HasAdmin returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *ACLS) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *ACLS) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *ACLS) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *ACLS) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


