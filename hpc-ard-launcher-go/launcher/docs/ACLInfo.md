# ACLInfo

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Allowed** | Pointer to **[]string** |  | [optional] 
**Forbidden** | Pointer to **[]string** |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewACLInfo

`func NewACLInfo() *ACLInfo`

NewACLInfo instantiates a new ACLInfo object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewACLInfoWithDefaults

`func NewACLInfoWithDefaults() *ACLInfo`

NewACLInfoWithDefaults instantiates a new ACLInfo object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetAllowed

`func (o *ACLInfo) GetAllowed() []string`

GetAllowed returns the Allowed field if non-nil, zero value otherwise.

### GetAllowedOk

`func (o *ACLInfo) GetAllowedOk() (*[]string, bool)`

GetAllowedOk returns a tuple with the Allowed field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAllowed

`func (o *ACLInfo) SetAllowed(v []string)`

SetAllowed sets Allowed field to given value.

### HasAllowed

`func (o *ACLInfo) HasAllowed() bool`

HasAllowed returns a boolean if a field has been set.

### GetForbidden

`func (o *ACLInfo) GetForbidden() []string`

GetForbidden returns the Forbidden field if non-nil, zero value otherwise.

### GetForbiddenOk

`func (o *ACLInfo) GetForbiddenOk() (*[]string, bool)`

GetForbiddenOk returns a tuple with the Forbidden field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetForbidden

`func (o *ACLInfo) SetForbidden(v []string)`

SetForbidden sets Forbidden field to given value.

### HasForbidden

`func (o *ACLInfo) HasForbidden() bool`

HasForbidden returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *ACLInfo) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *ACLInfo) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *ACLInfo) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *ACLInfo) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


