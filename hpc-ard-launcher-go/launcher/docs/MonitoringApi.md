# \MonitoringApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CanManageEnvironment**](MonitoringApi.md#CanManageEnvironment) | **Get** /monitoring/{owner}/environments/{environment}/management | Gets the management status of the environment
[**DeleteEnvironment**](MonitoringApi.md#DeleteEnvironment) | **Delete** /monitoring/{owner}/environments/{environment}/management | Deletes an environment regardless of status (destructive) which is not the same as terminating it, the environment may still be running but no longer visible to the dispatch server after this operation succeeds
[**DeleteEnvironmentLog**](MonitoringApi.md#DeleteEnvironmentLog) | **Delete** /monitoring/{owner}/environments/{environment}/logs/{log} | Deletes a specific log file for the environment
[**DeleteEnvironmentLogs**](MonitoringApi.md#DeleteEnvironmentLogs) | **Delete** /monitoring/{owner}/environments/{environment}/logs | Deletes all available log files for the environment
[**GetEnvironmentDetails**](MonitoringApi.md#GetEnvironmentDetails) | **Get** /monitoring/{owner}/environments/{environment} | Gets the current details of the environment
[**GetEnvironmentStatus**](MonitoringApi.md#GetEnvironmentStatus) | **Get** /monitoring/{owner}/environments/{environment}/status | Gets the status of an environment
[**ListEnvironmentLogs**](MonitoringApi.md#ListEnvironmentLogs) | **Get** /monitoring/{owner}/environments/{environment}/logs | Gets the content of a log file from the environment
[**LoadEnvironmentLog**](MonitoringApi.md#LoadEnvironmentLog) | **Get** /monitoring/{owner}/environments/{environment}/logs/{log} | Gets the available log files for the environment



## CanManageEnvironment

> DispatchManagementStatus CanManageEnvironment(ctx, owner, environment).Execute()

Gets the management status of the environment

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
    resp, r, err := api_client.MonitoringApi.CanManageEnvironment(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.CanManageEnvironment``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `CanManageEnvironment`: DispatchManagementStatus
    fmt.Fprintf(os.Stdout, "Response from `MonitoringApi.CanManageEnvironment`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiCanManageEnvironmentRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**DispatchManagementStatus**](DispatchManagementStatus.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteEnvironment

> DeleteEnvironment(ctx, owner, environment).Execute()

Deletes an environment regardless of status (destructive) which is not the same as terminating it, the environment may still be running but no longer visible to the dispatch server after this operation succeeds

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
    resp, r, err := api_client.MonitoringApi.DeleteEnvironment(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.DeleteEnvironment``: %v\n", err)
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

Other parameters are passed through a pointer to a apiDeleteEnvironmentRequest struct via the builder pattern


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


## DeleteEnvironmentLog

> DeleteEnvironmentLog(ctx, owner, environment, log).Execute()

Deletes a specific log file for the environment

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
    log := "output.log" // string | The log file that you wish to access

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.MonitoringApi.DeleteEnvironmentLog(context.Background(), owner, environment, log).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.DeleteEnvironmentLog``: %v\n", err)
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
**log** | **string** | The log file that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiDeleteEnvironmentLogRequest struct via the builder pattern


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


## DeleteEnvironmentLogs

> DeleteEnvironmentLogs(ctx, owner, environment).Execute()

Deletes all available log files for the environment

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
    resp, r, err := api_client.MonitoringApi.DeleteEnvironmentLogs(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.DeleteEnvironmentLogs``: %v\n", err)
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

Other parameters are passed through a pointer to a apiDeleteEnvironmentLogsRequest struct via the builder pattern


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


## GetEnvironmentDetails

> Manifest GetEnvironmentDetails(ctx, owner, environment).Execute()

Gets the current details of the environment

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
    resp, r, err := api_client.MonitoringApi.GetEnvironmentDetails(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.GetEnvironmentDetails``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetEnvironmentDetails`: Manifest
    fmt.Fprintf(os.Stdout, "Response from `MonitoringApi.GetEnvironmentDetails`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetEnvironmentDetailsRequest struct via the builder pattern


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


## GetEnvironmentStatus

> DispatchInfo GetEnvironmentStatus(ctx, owner, environment).EventLimit(eventLimit).Refresh(refresh).Execute()

Gets the status of an environment

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
    eventLimit := int32(56) // int32 | Number of events to limit to per DispatchInfo (optional)
    refresh := true // bool | Whether to actively refresh information prior to returning it (optional) (default to false)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.MonitoringApi.GetEnvironmentStatus(context.Background(), owner, environment).EventLimit(eventLimit).Refresh(refresh).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.GetEnvironmentStatus``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `GetEnvironmentStatus`: DispatchInfo
    fmt.Fprintf(os.Stdout, "Response from `MonitoringApi.GetEnvironmentStatus`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiGetEnvironmentStatusRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **eventLimit** | **int32** | Number of events to limit to per DispatchInfo | 
 **refresh** | **bool** | Whether to actively refresh information prior to returning it | [default to false]

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


## ListEnvironmentLogs

> map[string][]EnvironmentLogSource ListEnvironmentLogs(ctx, owner, environment).Execute()

Gets the content of a log file from the environment

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
    resp, r, err := api_client.MonitoringApi.ListEnvironmentLogs(context.Background(), owner, environment).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.ListEnvironmentLogs``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `ListEnvironmentLogs`: map[string][]EnvironmentLogSource
    fmt.Fprintf(os.Stdout, "Response from `MonitoringApi.ListEnvironmentLogs`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiListEnvironmentLogsRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



### Return type

[**map[string][]EnvironmentLogSource**](array.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## LoadEnvironmentLog

> *os.File LoadEnvironmentLog(ctx, owner, environment, log).IfModifiedSince(ifModifiedSince).IfUnmodifiedSince(ifUnmodifiedSince).Range_(range_).IfRange(ifRange).Execute()

Gets the available log files for the environment

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
    log := "output.log" // string | The log file that you wish to access
    ifModifiedSince := "ifModifiedSince_example" // string | Makes the request conditional on the resource having been modified since the given date.  If the resource is unmodified a 304 Not Modified is returned. (optional)
    ifUnmodifiedSince := "ifUnmodifiedSince_example" // string | Makes the request conditional on the resource having not been modified since the given date.  If the resource has been modified a 412 Precondition Failed is returned. (optional)
    range_ := "range__example" // string | Specifies a range of the resource to return instead of the full resource.  Byte based ranges per RFC 7233 are supported, additionally line based ranges are also permitted.  These support the same range syntax as byte ranges except using lines instead of bytes as the unit.  The main difference between byte and line ranges is that byte ranges use zero based indexing whereas line ranges use 1 based indexing.  So a range of 1-5 would be the 2nd through 6th bytes but the 1st through 5th  lines.  Note that per the specification range boundaries are always inclusive.  If a range is statisfiable you will receive a 206 Partial Content response with a Content-Range header indicating the portion of the resource returned.  If the range is unsatisfiable then you will receive a 416 Range Unsatisfiable response.  If the range matches the full size of the content then you will just receive a 200 OK response with the full resource.  Range specifications that are invalid are silently discarded and will just result in the server returning a normal 200 OK response.  Ranges can be specified relative to the start/end of the content per RFC 7233.  Note that when line  ranges are used in this way the returned Content-Range header may not match the lines returned because the server does not know in advance how many lines are in the resource.  However the starting line will always be correct and thus can be used by clients to display line numbers if they so wish.  Note that ONLY a single range may be requested.  If multiple ranges are requested the server will just return the full resource.  (optional)
    ifRange := "ifRange_example" // string | Make a Range request conditional on the resource having not been modified since the given date.  If the resource has been modified since the given date then the Range request will not be honoured and the full resource will be returned instead. (optional)

    configuration := openapiclient.NewConfiguration()
    api_client := openapiclient.NewAPIClient(configuration)
    resp, r, err := api_client.MonitoringApi.LoadEnvironmentLog(context.Background(), owner, environment, log).IfModifiedSince(ifModifiedSince).IfUnmodifiedSince(ifUnmodifiedSince).Range_(range_).IfRange(ifRange).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `MonitoringApi.LoadEnvironmentLog``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `LoadEnvironmentLog`: *os.File
    fmt.Fprintf(os.Stdout, "Response from `MonitoringApi.LoadEnvironmentLog`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**owner** | **string** | The username of the user whose resources that you wish to access | 
**environment** | **string** | The environment that you wish to access | 
**log** | **string** | The log file that you wish to access | 

### Other Parameters

Other parameters are passed through a pointer to a apiLoadEnvironmentLogRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **ifModifiedSince** | **string** | Makes the request conditional on the resource having been modified since the given date.  If the resource is unmodified a 304 Not Modified is returned. | 
 **ifUnmodifiedSince** | **string** | Makes the request conditional on the resource having not been modified since the given date.  If the resource has been modified a 412 Precondition Failed is returned. | 
 **range_** | **string** | Specifies a range of the resource to return instead of the full resource.  Byte based ranges per RFC 7233 are supported, additionally line based ranges are also permitted.  These support the same range syntax as byte ranges except using lines instead of bytes as the unit.  The main difference between byte and line ranges is that byte ranges use zero based indexing whereas line ranges use 1 based indexing.  So a range of 1-5 would be the 2nd through 6th bytes but the 1st through 5th  lines.  Note that per the specification range boundaries are always inclusive.  If a range is statisfiable you will receive a 206 Partial Content response with a Content-Range header indicating the portion of the resource returned.  If the range is unsatisfiable then you will receive a 416 Range Unsatisfiable response.  If the range matches the full size of the content then you will just receive a 200 OK response with the full resource.  Range specifications that are invalid are silently discarded and will just result in the server returning a normal 200 OK response.  Ranges can be specified relative to the start/end of the content per RFC 7233.  Note that when line  ranges are used in this way the returned Content-Range header may not match the lines returned because the server does not know in advance how many lines are in the resource.  However the starting line will always be correct and thus can be used by clients to display line numbers if they so wish.  Note that ONLY a single range may be requested.  If multiple ranges are requested the server will just return the full resource.  | 
 **ifRange** | **string** | Make a Range request conditional on the resource having not been modified since the given date.  If the resource has been modified since the given date then the Range request will not be honoured and the full resource will be returned instead. | 

### Return type

[***os.File**](*os.File.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/octet-stream, multipart/byteranges, application/problem+json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

