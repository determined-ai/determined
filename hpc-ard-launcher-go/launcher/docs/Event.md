# Event

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Level** | Pointer to **string** |  | [optional] 
**Timestamp** | Pointer to **string** |  | [optional] 
**Reporter** | Pointer to **string** |  | [optional] 
**Message** | Pointer to **string** |  | [optional] 
**ExtraInfo** | Pointer to **map[string]string** |  | [optional] 
**AdditionalPropertiesField** | Pointer to **map[string]interface{}** |  | [optional] 

## Methods

### NewEvent

`func NewEvent() *Event`

NewEvent instantiates a new Event object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEventWithDefaults

`func NewEventWithDefaults() *Event`

NewEventWithDefaults instantiates a new Event object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetLevel

`func (o *Event) GetLevel() string`

GetLevel returns the Level field if non-nil, zero value otherwise.

### GetLevelOk

`func (o *Event) GetLevelOk() (*string, bool)`

GetLevelOk returns a tuple with the Level field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetLevel

`func (o *Event) SetLevel(v string)`

SetLevel sets Level field to given value.

### HasLevel

`func (o *Event) HasLevel() bool`

HasLevel returns a boolean if a field has been set.

### GetTimestamp

`func (o *Event) GetTimestamp() string`

GetTimestamp returns the Timestamp field if non-nil, zero value otherwise.

### GetTimestampOk

`func (o *Event) GetTimestampOk() (*string, bool)`

GetTimestampOk returns a tuple with the Timestamp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTimestamp

`func (o *Event) SetTimestamp(v string)`

SetTimestamp sets Timestamp field to given value.

### HasTimestamp

`func (o *Event) HasTimestamp() bool`

HasTimestamp returns a boolean if a field has been set.

### GetReporter

`func (o *Event) GetReporter() string`

GetReporter returns the Reporter field if non-nil, zero value otherwise.

### GetReporterOk

`func (o *Event) GetReporterOk() (*string, bool)`

GetReporterOk returns a tuple with the Reporter field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetReporter

`func (o *Event) SetReporter(v string)`

SetReporter sets Reporter field to given value.

### HasReporter

`func (o *Event) HasReporter() bool`

HasReporter returns a boolean if a field has been set.

### GetMessage

`func (o *Event) GetMessage() string`

GetMessage returns the Message field if non-nil, zero value otherwise.

### GetMessageOk

`func (o *Event) GetMessageOk() (*string, bool)`

GetMessageOk returns a tuple with the Message field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetMessage

`func (o *Event) SetMessage(v string)`

SetMessage sets Message field to given value.

### HasMessage

`func (o *Event) HasMessage() bool`

HasMessage returns a boolean if a field has been set.

### GetExtraInfo

`func (o *Event) GetExtraInfo() map[string]string`

GetExtraInfo returns the ExtraInfo field if non-nil, zero value otherwise.

### GetExtraInfoOk

`func (o *Event) GetExtraInfoOk() (*map[string]string, bool)`

GetExtraInfoOk returns a tuple with the ExtraInfo field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetExtraInfo

`func (o *Event) SetExtraInfo(v map[string]string)`

SetExtraInfo sets ExtraInfo field to given value.

### HasExtraInfo

`func (o *Event) HasExtraInfo() bool`

HasExtraInfo returns a boolean if a field has been set.

### GetAdditionalPropertiesField

`func (o *Event) GetAdditionalPropertiesField() map[string]interface{}`

GetAdditionalPropertiesField returns the AdditionalPropertiesField field if non-nil, zero value otherwise.

### GetAdditionalPropertiesFieldOk

`func (o *Event) GetAdditionalPropertiesFieldOk() (*map[string]interface{}, bool)`

GetAdditionalPropertiesFieldOk returns a tuple with the AdditionalPropertiesField field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalPropertiesField

`func (o *Event) SetAdditionalPropertiesField(v map[string]interface{})`

SetAdditionalPropertiesField sets AdditionalPropertiesField field to given value.

### HasAdditionalPropertiesField

`func (o *Event) HasAdditionalPropertiesField() bool`

HasAdditionalPropertiesField returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


