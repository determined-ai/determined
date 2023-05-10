# MonitoringApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**canManageEnvironment**](MonitoringApi.md#canManageEnvironment) | **GET** /monitoring/{owner}/environments/{environment}/management | Gets the management status of the environment
[**deleteEnvironment**](MonitoringApi.md#deleteEnvironment) | **DELETE** /monitoring/{owner}/environments/{environment}/management | Deletes an environment regardless of status (destructive) which is not the same as terminating it, the environment may still be running but no longer visible to the dispatch server after this operation succeeds
[**deleteEnvironmentLog**](MonitoringApi.md#deleteEnvironmentLog) | **DELETE** /monitoring/{owner}/environments/{environment}/logs/{log} | Deletes a specific log file for the environment
[**deleteEnvironmentLogs**](MonitoringApi.md#deleteEnvironmentLogs) | **DELETE** /monitoring/{owner}/environments/{environment}/logs | Deletes all available log files for the environment
[**getEnvironmentDetails**](MonitoringApi.md#getEnvironmentDetails) | **GET** /monitoring/{owner}/environments/{environment} | Gets the current details of the environment
[**getEnvironmentStatus**](MonitoringApi.md#getEnvironmentStatus) | **GET** /monitoring/{owner}/environments/{environment}/status | Gets the status of an environment
[**listEnvironmentLogs**](MonitoringApi.md#listEnvironmentLogs) | **GET** /monitoring/{owner}/environments/{environment}/logs | Gets the content of a log file from the environment
[**loadEnvironmentLog**](MonitoringApi.md#loadEnvironmentLog) | **GET** /monitoring/{owner}/environments/{environment}/logs/{log} | Gets the available log files for the environment



## canManageEnvironment

> com.cray.analytics.capsules.model.DispatchManagementStatus canManageEnvironment(owner, environment)

Gets the management status of the environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            com.cray.analytics.capsules.model.DispatchManagementStatus result = apiInstance.canManageEnvironment(owner, environment);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#canManageEnvironment");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |

### Return type

[**com.cray.analytics.capsules.model.DispatchManagementStatus**](com.cray.analytics.capsules.model.DispatchManagementStatus.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Management Status report |  -  |
| **403** | User does not have permission to view environments with the given owner |  -  |
| **404** | The specified environment does not exist |  -  |


## deleteEnvironment

> deleteEnvironment(owner, environment)

Deletes an environment regardless of status (destructive) which is not the same as terminating it, the environment may still be running but no longer visible to the dispatch server after this operation succeeds

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            apiInstance.deleteEnvironment(owner, environment);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#deleteEnvironment");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | Environment was deleted |  -  |
| **403** | User does not have permission to delete environments with the given owner |  -  |
| **404** | The specified environment does not exist |  -  |


## deleteEnvironmentLog

> deleteEnvironmentLog(owner, environment, log)

Deletes a specific log file for the environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        String log = "output.log"; // String | The log file that you wish to access
        try {
            apiInstance.deleteEnvironmentLog(owner, environment, log);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#deleteEnvironmentLog");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |
 **log** | **String**| The log file that you wish to access |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | Specified log file were deleted |  -  |
| **403** | User does not have permission to manage logs from environments with the given owner |  -  |
| **404** | The given environment or log file does not exist |  -  |


## deleteEnvironmentLogs

> deleteEnvironmentLogs(owner, environment)

Deletes all available log files for the environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            apiInstance.deleteEnvironmentLogs(owner, environment);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#deleteEnvironmentLogs");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | Log files were deleted |  -  |
| **403** | User does not have permission to manage logs from environments with the given owner |  -  |
| **404** | The given environment does not exist |  -  |


## getEnvironmentDetails

> com.cray.analytics.capsules.model.EnvironmentManifest getEnvironmentDetails(owner, environment)

Gets the current details of the environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            com.cray.analytics.capsules.model.EnvironmentManifest result = apiInstance.getEnvironmentDetails(owner, environment);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#getEnvironmentDetails");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |

### Return type

[**com.cray.analytics.capsules.model.EnvironmentManifest**](com.cray.analytics.capsules.model.EnvironmentManifest.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Environment Manifest |  -  |
| **403** | User does not have permission to view environments with the given owner |  -  |
| **404** | The specified environment does not exist |  -  |


## getEnvironmentStatus

> com.cray.analytics.capsules.model.DispatchInfo getEnvironmentStatus(owner, environment, eventLimit, refresh)

Gets the status of an environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        Integer eventLimit = 56; // Integer | Number of events to limit to per DispatchInfo
        Boolean refresh = false; // Boolean | Whether to actively refresh information prior to returning it
        try {
            com.cray.analytics.capsules.model.DispatchInfo result = apiInstance.getEnvironmentStatus(owner, environment, eventLimit, refresh);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#getEnvironmentStatus");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |
 **eventLimit** | **Integer**| Number of events to limit to per DispatchInfo | [optional]
 **refresh** | **Boolean**| Whether to actively refresh information prior to returning it | [optional] [default to false]

### Return type

[**com.cray.analytics.capsules.model.DispatchInfo**](com.cray.analytics.capsules.model.DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | The status of the environment |  -  |
| **403** | User does not have permission to view environments with the given owner |  -  |
| **404** | The specified environment does not exist |  -  |


## listEnvironmentLogs

> Map&lt;String, List&lt;com.cray.analytics.capsules.model.LogSource&gt;&gt; listEnvironmentLogs(owner, environment)

Gets the content of a log file from the environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            Map<String, List<com.cray.analytics.capsules.model.LogSource>> result = apiInstance.listEnvironmentLogs(owner, environment);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#listEnvironmentLogs");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |

### Return type

[**Map&lt;String, List&lt;com.cray.analytics.capsules.model.LogSource&gt;&gt;**](List.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of log files |  -  |
| **403** | User does not have permission to view logs from environments with the given owner |  -  |


## loadEnvironmentLog

> File loadEnvironmentLog(owner, environment, log, ifModifiedSince, ifUnmodifiedSince, range, ifRange)

Gets the available log files for the environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.MonitoringApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        MonitoringApi apiInstance = new MonitoringApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        String log = "output.log"; // String | The log file that you wish to access
        String ifModifiedSince = "ifModifiedSince_example"; // String | Makes the request conditional on the resource having been modified since the given date.  If the resource is unmodified a 304 Not Modified is returned.
        String ifUnmodifiedSince = "ifUnmodifiedSince_example"; // String | Makes the request conditional on the resource having not been modified since the given date.  If the resource has been modified a 412 Precondition Failed is returned.
        String range = "range_example"; // String | Specifies a range of the resource to return instead of the full resource.  Byte based ranges per RFC 7233 are supported, additionally line based ranges are also permitted.  These support the same range syntax as byte ranges except using lines instead of bytes as the unit.  The main difference between byte and line ranges is that byte ranges use zero based indexing whereas line ranges use 1 based indexing.  So a range of 1-5 would be the 2nd through 6th bytes but the 1st through 5th  lines.  Note that per the specification range boundaries are always inclusive.  If a range is statisfiable you will receive a 206 Partial Content response with a Content-Range header indicating the portion of the resource returned.  If the range is unsatisfiable then you will receive a 416 Range Unsatisfiable response.  If the range matches the full size of the content then you will just receive a 200 OK response with the full resource.  Range specifications that are invalid are silently discarded and will just result in the server returning a normal 200 OK response.  Ranges can be specified relative to the start/end of the content per RFC 7233.  Note that when line  ranges are used in this way the returned Content-Range header may not match the lines returned because the server does not know in advance how many lines are in the resource.  However the starting line will always be correct and thus can be used by clients to display line numbers if they so wish.  Note that ONLY a single range may be requested.  If multiple ranges are requested the server will just return the full resource. 
        String ifRange = "ifRange_example"; // String | Make a Range request conditional on the resource having not been modified since the given date.  If the resource has been modified since the given date then the Range request will not be honoured and the full resource will be returned instead.
        try {
            File result = apiInstance.loadEnvironmentLog(owner, environment, log, ifModifiedSince, ifUnmodifiedSince, range, ifRange);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling MonitoringApi#loadEnvironmentLog");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **owner** | **String**| The username of the user whose resources that you wish to access |
 **environment** | **String**| The environment that you wish to access |
 **log** | **String**| The log file that you wish to access |
 **ifModifiedSince** | **String**| Makes the request conditional on the resource having been modified since the given date.  If the resource is unmodified a 304 Not Modified is returned. | [optional]
 **ifUnmodifiedSince** | **String**| Makes the request conditional on the resource having not been modified since the given date.  If the resource has been modified a 412 Precondition Failed is returned. | [optional]
 **range** | **String**| Specifies a range of the resource to return instead of the full resource.  Byte based ranges per RFC 7233 are supported, additionally line based ranges are also permitted.  These support the same range syntax as byte ranges except using lines instead of bytes as the unit.  The main difference between byte and line ranges is that byte ranges use zero based indexing whereas line ranges use 1 based indexing.  So a range of 1-5 would be the 2nd through 6th bytes but the 1st through 5th  lines.  Note that per the specification range boundaries are always inclusive.  If a range is statisfiable you will receive a 206 Partial Content response with a Content-Range header indicating the portion of the resource returned.  If the range is unsatisfiable then you will receive a 416 Range Unsatisfiable response.  If the range matches the full size of the content then you will just receive a 200 OK response with the full resource.  Range specifications that are invalid are silently discarded and will just result in the server returning a normal 200 OK response.  Ranges can be specified relative to the start/end of the content per RFC 7233.  Note that when line  ranges are used in this way the returned Content-Range header may not match the lines returned because the server does not know in advance how many lines are in the resource.  However the starting line will always be correct and thus can be used by clients to display line numbers if they so wish.  Note that ONLY a single range may be requested.  If multiple ranges are requested the server will just return the full resource.  | [optional]
 **ifRange** | **String**| Make a Range request conditional on the resource having not been modified since the given date.  If the resource has been modified since the given date then the Range request will not be honoured and the full resource will be returned instead. | [optional]

### Return type

[**File**](File.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/octet-stream, multipart/byteranges


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Full log file contents (at time of the request) |  -  |
| **206** | A subset of the log file contents, returned when a valid Range request is made |  -  |
| **304** | The log file has not changed |  -  |
| **403** | User does not have permission to view logs from environments with the given owner |  -  |
| **404** | Log file or environment does not exist |  -  |
| **412** | The log file has changed |  -  |
| **416** | The requested ranges of bytes can not be satisifed for this log file |  -  |
| **500** | Could not retrieve log |  -  |

