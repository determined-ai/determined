# \TerminatedApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**DeleteAllTerminated**](TerminatedApi.md#DeleteAllTerminated) | **Delete** /terminated/{owner} | Removes all terminated environments belonging to the given owner
[**DeleteTerminated**](TerminatedApi.md#DeleteTerminated) | **Delete** /terminated/{owner}/environments/{environment} | Removes a terminated environment
[**GetTerminated**](TerminatedApi.md#GetTerminated) | **Get** /terminated/{owner}/environments/{environment} | Gets a specific terminated environment launched by the given owner
[**GetTerminatedACLs**](TerminatedApi.md#GetTerminatedACLs) | **Get** /terminated/{owner}/acls | Gets the ACLs that control who can manage the terminated environments belonging to the given owner
[**ListAllTerminated**](TerminatedApi.md#ListAllTerminated) | **Get** /terminated | Gets all terminated environments that the user can view
[**ListOwnedTerminated**](TerminatedApi.md#ListOwnedTerminated) | **Get** /terminated/{owner} | Gets all terminated environments belonging to the given owner
[**SetTerminatedACLs**](TerminatedApi.md#SetTerminatedACLs) | **Put** /terminated/{owner}/acls | Sets the ACLs that control who can manage the terminated environments belonging to the given owner



## DeleteAllTerminated

> DeleteAllTerminated(ctx, owner).Execute()

Removes all terminated environments belonging to the given owner

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    owner := "lhamilton" // string | The username of the user whose resources that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.DeleteAllTerminated(context.Background(), owner).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.DeleteAllTerminated``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteAllTerminatedRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteTerminated

> DeleteTerminated(ctx, owner, environment).Execute()

Removes a terminated environment

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    owner := "lhamilton" // string | The username of the user whose resources that you wish to access
    environment := "abcdef1234" // string | The environment that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.DeleteTerminated(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.DeleteTerminated``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteTerminatedRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetTerminated

> Manifest GetTerminated(ctx, owner, environment).Execute()

Gets a specific terminated environment launched by the given owner

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    owner := "lhamilton" // string | The username of the user whose resources that you wish to access
    environment := "abcdef1234" // string | The environment that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.GetTerminated(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.GetTerminated``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetTerminated`: Manifest
    fmt.Fprintf(os.Stdout, "Response from `TerminatedApi.GetTerminated`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetTerminatedRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**Manifest**](Manifest.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetTerminatedACLs

> ACLS GetTerminatedACLs(ctx, owner).Execute()

Gets the ACLs that control who can manage the terminated environments belonging to the given owner

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    owner := "lhamilton" // string | The username of the user whose resources that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.GetTerminatedACLs(context.Background(), owner).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.GetTerminatedACLs``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetTerminatedACLs`: ACLS
    fmt.Fprintf(os.Stdout, "Response from `TerminatedApi.GetTerminatedACLs`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetTerminatedACLsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**ACLS**](ACLS.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAllTerminated

> []DispatchInfo ListAllTerminated(ctx).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()

Gets all terminated environments that the user can view

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    limit := int32(56) // int32 | Number of results to limit to, used in conjunction with offset to page through results (optional)
    offset := int32(56) // int32 | Number of results to offset by, used in conjunction with limit to page through results (optional) (default to 0)
    reverse := true // bool | Whether to reverse the default sort order in the returned results (optional) (default to false)
    eventLimit := int32(56) // int32 | Number of events to limit to per DispatchInfo (optional)
    state := []openapiclient.DispatchState{openapiclient.DispatchState("UNKNOWN")} // []DispatchState | Results must be in the given state(s) (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.ListAllTerminated(context.Background()).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.ListAllTerminated``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListAllTerminated`: []DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `TerminatedApi.ListAllTerminated`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListAllTerminatedRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **limit** | **int32** | Number of results to limit to, used in conjunction with offset to page through results | 
 **offset** | **int32** | Number of results to offset by, used in conjunction with limit to page through results | [default to 0]
 **reverse** | **bool** | Whether to reverse the default sort order in the returned results | [default to false]
 **eventLimit** | **int32** | Number of events to limit to per DispatchInfo | 
 **state** | [**[]DispatchState**](DispatchState.md) | Results must be in the given state(s) | 

### Return type

[**[]DispatchInfo**](DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListOwnedTerminated

> []DispatchInfo ListOwnedTerminated(ctx, owner).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()

Gets all terminated environments belonging to the given owner

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    owner := "lhamilton" // string | The username of the user whose resources that you wish to access
    limit := int32(56) // int32 | Number of results to limit to, used in conjunction with offset to page through results (optional)
    offset := int32(56) // int32 | Number of results to offset by, used in conjunction with limit to page through results (optional) (default to 0)
    reverse := true // bool | Whether to reverse the default sort order in the returned results (optional) (default to false)
    eventLimit := int32(56) // int32 | Number of events to limit to per DispatchInfo (optional)
    state := []openapiclient.DispatchState{openapiclient.DispatchState("UNKNOWN")} // []DispatchState | Results must be in the given state(s) (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.ListOwnedTerminated(context.Background(), owner).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.ListOwnedTerminated``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListOwnedTerminated`: []DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `TerminatedApi.ListOwnedTerminated`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiListOwnedTerminatedRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **limit** | **int32** | Number of results to limit to, used in conjunction with offset to page through results | 
 **offset** | **int32** | Number of results to offset by, used in conjunction with limit to page through results | [default to 0]
 **reverse** | **bool** | Whether to reverse the default sort order in the returned results | [default to false]
 **eventLimit** | **int32** | Number of events to limit to per DispatchInfo | 
 **state** | [**[]DispatchState**](DispatchState.md) | Results must be in the given state(s) | 

### Return type

[**[]DispatchInfo**](DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SetTerminatedACLs

> SetTerminatedACLs(ctx, owner).ACLS(aCLS).Execute()

Sets the ACLs that control who can manage the terminated environments belonging to the given owner

### Example

```go
package main

import (
    "context"
    "fmt"
    "os"
    openapiclient "./openapi"
)

func main() {
    owner := "lhamilton" // string | The username of the user whose resources that you wish to access
    aCLS := *openapiclient.NewACLS() // ACLS | The ACLs to set

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.TerminatedApi.SetTerminatedACLs(context.Background(), owner).ACLS(aCLS).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `TerminatedApi.SetTerminatedACLs``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiSetTerminatedACLsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **aCLS** | [**ACLS**](ACLS.md) | The ACLs to set | 

### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

