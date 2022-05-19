# \AdminApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetLoggerLevel**](AdminApi.md#GetLoggerLevel) | **Get** /admin/loggers/{logger} | Gets the log level for a specific logger
[**GetRootLogLevel**](AdminApi.md#GetRootLogLevel) | **Get** /admin/log-level | Gets the current root log level
[**ListAllLoggerLevels**](AdminApi.md#ListAllLoggerLevels) | **Get** /admin/loggers | Gets the log level for all loggers
[**SetAllLoggerLevels**](AdminApi.md#SetAllLoggerLevels) | **Put** /admin/loggers | Sets the log level for multiple loggers
[**SetLoggerLevel**](AdminApi.md#SetLoggerLevel) | **Put** /admin/loggers/{logger} | Sets the log level for a specific logger
[**SetRootLogLevel**](AdminApi.md#SetRootLogLevel) | **Put** /admin/log-level | Sets the root log level



## GetLoggerLevel

> LogLevel GetLoggerLevel(ctx, logger).Execute()

Gets the log level for a specific logger

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
    logger := "io.swagger.oas.inflector.controllers.OpenAPIOperationController" // string | The logger whose log level you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.AdminApi.GetLoggerLevel(context.Background(), logger).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `AdminApi.GetLoggerLevel``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetLoggerLevel`: LogLevel
    fmt.Fprintf(os.Stdout, "Response from `AdminApi.GetLoggerLevel`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**logger** | **string** | The logger whose log level you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetLoggerLevelRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


### Return type

[**LogLevel**](LogLevel.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetRootLogLevel

> LogLevel GetRootLogLevel(ctx).Execute()

Gets the current root log level

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

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.AdminApi.GetRootLogLevel(context.Background()).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `AdminApi.GetRootLogLevel``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetRootLogLevel`: LogLevel
    fmt.Fprintf(os.Stdout, "Response from `AdminApi.GetRootLogLevel`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiGetRootLogLevelRequest struct via the builder pattern


### Return type

[**LogLevel**](LogLevel.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListAllLoggerLevels

> map[string]LogLevel ListAllLoggerLevels(ctx).Execute()

Gets the log level for all loggers

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

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.AdminApi.ListAllLoggerLevels(context.Background()).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `AdminApi.ListAllLoggerLevels``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListAllLoggerLevels`: map[string]LogLevel
    fmt.Fprintf(os.Stdout, "Response from `AdminApi.ListAllLoggerLevels`: %v\n", resp)
}
```

### Path Parameters

This endpoint does not need any parameter.

### Other Parameters

Other parameters are passed through a pointer to a apiListAllLoggerLevelsRequest struct via the builder pattern


### Return type

[**map[string]LogLevel**](LogLevel.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SetAllLoggerLevels

> SetAllLoggerLevels(ctx).RequestBody(requestBody).Execute()

Sets the log level for multiple loggers

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
    requestBody := map[string]LogLevel{"key": openapiclient.LogLevel("false")} // map[string]LogLevel | The loggers and levels to set for each logger

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.AdminApi.SetAllLoggerLevels(context.Background()).RequestBody(requestBody).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `AdminApi.SetAllLoggerLevels``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSetAllLoggerLevelsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **requestBody** | [**map[string]LogLevel**](LogLevel.md) | The loggers and levels to set for each logger | 

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


## SetLoggerLevel

> SetLoggerLevel(ctx, logger).Body(body).Execute()

Sets the log level for a specific logger

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
    logger := "io.swagger.oas.inflector.controllers.OpenAPIOperationController" // string | The logger whose log level you wish to access
    body := string(987) // string | The log level to set

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.AdminApi.SetLoggerLevel(context.Background(), logger).Body(body).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `AdminApi.SetLoggerLevel``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**logger** | **string** | The logger whose log level you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiSetLoggerLevelRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **body** | **string** | The log level to set | 

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


## SetRootLogLevel

> SetRootLogLevel(ctx).Body(body).Execute()

Sets the root log level

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
    body := string(987) // string | The log level to set

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.AdminApi.SetRootLogLevel(context.Background()).Body(body).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `AdminApi.SetRootLogLevel``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSetRootLogLevelRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **body** | **string** | The log level to set | 

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

