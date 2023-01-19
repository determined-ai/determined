# \LaunchApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddCredential**](LaunchApi.md#AddCredential) | **Put** /launch/credentials/{owner}/{name} | Creates/updates a credential that the Dispatch Centre can use to launch environments on behalf of the user
[**HasCredential**](LaunchApi.md#HasCredential) | **Head** /launch/credentials/{owner}/{name} | Determines whether a given credential has been provided
[**Launch**](LaunchApi.md#Launch) | **Put** /launch | Launches the runtime environment described by the provided manifest in a synchronous manner
[**LaunchAsync**](LaunchApi.md#LaunchAsync) | **Put** /launch/async | Launches the runtime environment described by the provided manifest in an asynchronous manner
[**RemoveCredential**](LaunchApi.md#RemoveCredential) | **Delete** /launch/credentials/{owner}/{name} | Removes a credential



## AddCredential

> AddCredential(ctx, owner, name).Body(body).Execute()

Creates/updates a credential that the Dispatch Centre can use to launch environments on behalf of the user

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
    name := "track-analysis" // string | The name of the resource that you wish to access
    body := os.NewFile(1234, "some_file") // *os.File | The credential data to store

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.LaunchApi.AddCredential(context.Background(), owner, name).Body(body).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `LaunchApi.AddCredential``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**name** | **string** | The name of the resource that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiAddCredentialRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **body** | ***os.File** | The credential data to store | 

### Return type

 (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/octet-stream
- **Accept**: application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HasCredential

> HasCredential(ctx, owner, name).Execute()

Determines whether a given credential has been provided

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
    name := "track-analysis" // string | The name of the resource that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.LaunchApi.HasCredential(context.Background(), owner, name).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `LaunchApi.HasCredential``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**name** | **string** | The name of the resource that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiHasCredentialRequest struct via the builder pattern


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


## Launch

> DispatchInfo Launch(ctx).Manifest(manifest).Impersonate(impersonate).Execute()

Launches the runtime environment described by the provided manifest in a synchronous manner

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
    manifest := *openapiclient.NewManifest("ManifestVersion_example", *openapiclient.NewClientMetadata("Name_example")) // Manifest | The manifest to launch
    impersonate := "impersonate_example" // string | User to impersonate (user encoded in authorization token must be configured as an administrator) (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.LaunchApi.Launch(context.Background()).Manifest(manifest).Impersonate(impersonate).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `LaunchApi.Launch``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `Launch`: DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `LaunchApi.Launch`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiLaunchRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest** | [**Manifest**](Manifest.md) | The manifest to launch | 
 **impersonate** | **string** | User to impersonate (user encoded in authorization token must be configured as an administrator) | 

### Return type

[**DispatchInfo**](DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LaunchAsync

> DispatchInfo LaunchAsync(ctx).Manifest(manifest).Impersonate(impersonate).Execute()

Launches the runtime environment described by the provided manifest in an asynchronous manner

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
    manifest := *openapiclient.NewManifest("ManifestVersion_example", *openapiclient.NewClientMetadata("Name_example")) // Manifest | The manifest to launch
    impersonate := "impersonate_example" // string | User to impersonate (user encoded in authorization token must be configured as an administrator) (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.LaunchApi.LaunchAsync(context.Background()).Manifest(manifest).Impersonate(impersonate).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `LaunchApi.LaunchAsync``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `LaunchAsync`: DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `LaunchApi.LaunchAsync`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiLaunchAsyncRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **manifest** | [**Manifest**](Manifest.md) | The manifest to launch | 
 **impersonate** | **string** | User to impersonate (user encoded in authorization token must be configured as an administrator) | 

### Return type

[**DispatchInfo**](DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## RemoveCredential

> RemoveCredential(ctx, owner, name).Execute()

Removes a credential

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
    name := "track-analysis" // string | The name of the resource that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.LaunchApi.RemoveCredential(context.Background(), owner, name).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `LaunchApi.RemoveCredential``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**name** | **string** | The name of the resource that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiRemoveCredentialRequest struct via the builder pattern


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

