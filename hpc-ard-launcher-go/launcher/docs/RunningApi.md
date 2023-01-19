# \RunningApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetRunning**](RunningApi.md#GetRunning) | **Get** /running/{owner}/environments/{environment} | Gets a specific running environment launched by the given owner
[**GetRunningACLs**](RunningApi.md#GetRunningACLs) | **Get** /running/{owner}/acls | Gets the ACLs that control who can manage the running environments belonging to the given owner
[**GetUserInterfaces**](RunningApi.md#GetUserInterfaces) | **Get** /running/{owner}/environments/{environment}/uis | Gets the User Interfaces associated with the running environment launched by the given owner
[**ListAllRunning**](RunningApi.md#ListAllRunning) | **Get** /running | Gets all running environments that the user can view
[**ListOwnedRunning**](RunningApi.md#ListOwnedRunning) | **Get** /running/{owner} | Gets all running environments launched by the given owner
[**SetRunningACLs**](RunningApi.md#SetRunningACLs) | **Put** /running/{owner}/acls | Sets the ACLs that control who can manage the running environments belonging to the given owner
[**TerminateAllRunning**](RunningApi.md#TerminateAllRunning) | **Delete** /running/{owner} | Terminates all running environments owned by the given owner
[**TerminateRunning**](RunningApi.md#TerminateRunning) | **Delete** /running/{owner}/environments/{environment} | Terminates a running environment
[**TerminateRunningAsync**](RunningApi.md#TerminateRunningAsync) | **Delete** /running/{owner}/environments/{environment}/async | Terminates a running environment in an asynchronous manner



## GetRunning

> Manifest GetRunning(ctx, owner, environment).Execute()

Gets a specific running environment launched by the given owner

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
    resp, r, err := api_client.RunningApi.GetRunning(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.GetRunning``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetRunning`: Manifest
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.GetRunning`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetRunningRequest struct via the builder pattern


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


## GetRunningACLs

> ACLS GetRunningACLs(ctx, owner).Execute()

Gets the ACLs that control who can manage the running environments belonging to the given owner

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
    resp, r, err := api_client.RunningApi.GetRunningACLs(context.Background(), owner).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.GetRunningACLs``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetRunningACLs`: ACLS
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.GetRunningACLs`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetRunningACLsRequest struct via the builder pattern


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


## GetUserInterfaces

> map[string][]UserInterface GetUserInterfaces(ctx, owner, environment).Execute()

Gets the User Interfaces associated with the running environment launched by the given owner

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
    resp, r, err := api_client.RunningApi.GetUserInterfaces(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.GetUserInterfaces``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetUserInterfaces`: map[string][]UserInterface
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.GetUserInterfaces`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetUserInterfacesRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**map[string][]UserInterface**](array.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAllRunning

> map[string][]DispatchInfo ListAllRunning(ctx).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()

Gets all running environments that the user can view

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
    resp, r, err := api_client.RunningApi.ListAllRunning(context.Background()).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.ListAllRunning``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListAllRunning`: map[string][]DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.ListAllRunning`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiListAllRunningRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **limit** | **int32** | Number of results to limit to, used in conjunction with offset to page through results | 
 **offset** | **int32** | Number of results to offset by, used in conjunction with limit to page through results | [default to 0]
 **reverse** | **bool** | Whether to reverse the default sort order in the returned results | [default to false]
 **eventLimit** | **int32** | Number of events to limit to per DispatchInfo | 
 **state** | [**[]DispatchState**](DispatchState.md) | Results must be in the given state(s) | 

### Return type

[**map[string][]DispatchInfo**](array.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListOwnedRunning

> map[string][]DispatchInfo ListOwnedRunning(ctx, owner).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()

Gets all running environments launched by the given owner

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
    resp, r, err := api_client.RunningApi.ListOwnedRunning(context.Background(), owner).Limit(limit).Offset(offset).Reverse(reverse).EventLimit(eventLimit).State(state).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.ListOwnedRunning``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListOwnedRunning`: map[string][]DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.ListOwnedRunning`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiListOwnedRunningRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **limit** | **int32** | Number of results to limit to, used in conjunction with offset to page through results | 
 **offset** | **int32** | Number of results to offset by, used in conjunction with limit to page through results | [default to 0]
 **reverse** | **bool** | Whether to reverse the default sort order in the returned results | [default to false]
 **eventLimit** | **int32** | Number of events to limit to per DispatchInfo | 
 **state** | [**[]DispatchState**](DispatchState.md) | Results must be in the given state(s) | 

### Return type

[**map[string][]DispatchInfo**](array.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SetRunningACLs

> SetRunningACLs(ctx, owner).ACLS(aCLS).Execute()

Sets the ACLs that control who can manage the running environments belonging to the given owner

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
    resp, r, err := api_client.RunningApi.SetRunningACLs(context.Background(), owner).ACLS(aCLS).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.SetRunningACLs``: %v\n", err)
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

Other parameters are passed through a pointer to a apiSetRunningACLsRequest struct via the builder pattern


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


## TerminateAllRunning

> TerminateAllRunning(ctx, owner).Force(force).Execute()

Terminates all running environments owned by the given owner

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
    force := true // bool | Whether to force termination (optional) (default to false)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.RunningApi.TerminateAllRunning(context.Background(), owner).Force(force).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.TerminateAllRunning``: %v\n", err)
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

Other parameters are passed through a pointer to a apiTerminateAllRunningRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **force** | **bool** | Whether to force termination | [default to false]

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


## TerminateRunning

> DispatchInfo TerminateRunning(ctx, owner, environment).Force(force).Execute()

Terminates a running environment

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
    force := true // bool | Whether to force termination (optional) (default to false)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.RunningApi.TerminateRunning(context.Background(), owner, environment).Force(force).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.TerminateRunning``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `TerminateRunning`: DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.TerminateRunning`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiTerminateRunningRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **force** | **bool** | Whether to force termination | [default to false]

### Return type

[**DispatchInfo**](DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## TerminateRunningAsync

> DispatchInfo TerminateRunningAsync(ctx, owner, environment).Force(force).Execute()

Terminates a running environment in an asynchronous manner

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
    force := true // bool | Whether to force termination (optional) (default to false)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.RunningApi.TerminateRunningAsync(context.Background(), owner, environment).Force(force).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `RunningApi.TerminateRunningAsync``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `TerminateRunningAsync`: DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `RunningApi.TerminateRunningAsync`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiTerminateRunningAsyncRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **force** | **bool** | Whether to force termination | [default to false]

### Return type

[**DispatchInfo**](DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

