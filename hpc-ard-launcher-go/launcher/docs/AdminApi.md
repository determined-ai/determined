# AdminApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getLoggerLevel**](AdminApi.md#getLoggerLevel) | **GET** /admin/loggers/{logger} | Gets the log level for a specific logger
[**getRootLogLevel**](AdminApi.md#getRootLogLevel) | **GET** /admin/log-level | Gets the current root log level
[**listAllLoggerLevels**](AdminApi.md#listAllLoggerLevels) | **GET** /admin/loggers | Gets the log level for all loggers
[**setAllLoggerLevels**](AdminApi.md#setAllLoggerLevels) | **PUT** /admin/loggers | Sets the log level for multiple loggers
[**setLoggerLevel**](AdminApi.md#setLoggerLevel) | **PUT** /admin/loggers/{logger} | Sets the log level for a specific logger
[**setRootLogLevel**](AdminApi.md#setRootLogLevel) | **PUT** /admin/log-level | Sets the root log level



## getLoggerLevel

> com.cray.analytics.capsules.model.admin.LogLevel getLoggerLevel(logger)

Gets the log level for a specific logger

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.AdminApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        AdminApi apiInstance = new AdminApi(defaultClient);
        String logger = "io.swagger.oas.inflector.controllers.OpenAPIOperationController"; // String | The logger whose log level you wish to access
        try {
            com.cray.analytics.capsules.model.admin.LogLevel result = apiInstance.getLoggerLevel(logger);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling AdminApi#getLoggerLevel");
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
 **logger** | **String**| The logger whose log level you wish to access |

### Return type

[**com.cray.analytics.capsules.model.admin.LogLevel**](com.cray.analytics.capsules.model.admin.LogLevel.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Loggers current log level |  -  |
| **400** | Not a valid logger |  -  |
| **403** | User is not permitted to set log level |  -  |


## getRootLogLevel

> com.cray.analytics.capsules.model.admin.LogLevel getRootLogLevel()

Gets the current root log level

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.AdminApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        AdminApi apiInstance = new AdminApi(defaultClient);
        try {
            com.cray.analytics.capsules.model.admin.LogLevel result = apiInstance.getRootLogLevel();
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling AdminApi#getRootLogLevel");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**com.cray.analytics.capsules.model.admin.LogLevel**](com.cray.analytics.capsules.model.admin.LogLevel.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Current root log level |  -  |
| **403** | User is not permitted to read the log level |  -  |


## listAllLoggerLevels

> Map&lt;String, com.cray.analytics.capsules.model.admin.LogLevel&gt; listAllLoggerLevels()

Gets the log level for all loggers

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.AdminApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        AdminApi apiInstance = new AdminApi(defaultClient);
        try {
            Map<String, com.cray.analytics.capsules.model.admin.LogLevel> result = apiInstance.listAllLoggerLevels();
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling AdminApi#listAllLoggerLevels");
            System.err.println("Status code: " + e.getCode());
            System.err.println("Reason: " + e.getResponseBody());
            System.err.println("Response headers: " + e.getResponseHeaders());
            e.printStackTrace();
        }
    }
}
```

### Parameters

This endpoint does not need any parameter.

### Return type

[**Map&lt;String, com.cray.analytics.capsules.model.admin.LogLevel&gt;**](com.cray.analytics.capsules.model.admin.LogLevel.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | All loggers current log levels |  -  |


## setAllLoggerLevels

> setAllLoggerLevels(requestBody)

Sets the log level for multiple loggers

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.AdminApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        AdminApi apiInstance = new AdminApi(defaultClient);
        Map<String, com.cray.analytics.capsules.model.admin.LogLevel> requestBody = new HashMap(); // Map<String, com.cray.analytics.capsules.model.admin.LogLevel> | The loggers and levels to set for each logger
        try {
            apiInstance.setAllLoggerLevels(requestBody);
        } catch (ApiException e) {
            System.err.println("Exception when calling AdminApi#setAllLoggerLevels");
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
 **requestBody** | [**Map&lt;String, com.cray.analytics.capsules.model.admin.LogLevel&gt;**](com.cray.analytics.capsules.model.admin.LogLevel.md)| The loggers and levels to set for each logger |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The requested log levels were updated |  -  |
| **400** | Not a valid log level |  -  |
| **403** | User is not permitted to set log levels |  -  |


## setLoggerLevel

> setLoggerLevel(logger, body)

Sets the log level for a specific logger

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.AdminApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        AdminApi apiInstance = new AdminApi(defaultClient);
        String logger = "io.swagger.oas.inflector.controllers.OpenAPIOperationController"; // String | The logger whose log level you wish to access
        String body = "body_example"; // String | The log level to set
        try {
            apiInstance.setLoggerLevel(logger, body);
        } catch (ApiException e) {
            System.err.println("Exception when calling AdminApi#setLoggerLevel");
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
 **logger** | **String**| The logger whose log level you wish to access |
 **body** | **String**| The log level to set |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The logger level was updated |  -  |
| **400** | Not a valid log level or logger |  -  |
| **403** | User is not permitted to set logger level |  -  |


## setRootLogLevel

> setRootLogLevel(body)

Sets the root log level

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.AdminApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        AdminApi apiInstance = new AdminApi(defaultClient);
        String body = "body_example"; // String | The log level to set
        try {
            apiInstance.setRootLogLevel(body);
        } catch (ApiException e) {
            System.err.println("Exception when calling AdminApi#setRootLogLevel");
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
 **body** | **String**| The log level to set |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | The root log level was updated |  -  |
| **400** | Not a valid log level |  -  |
| **403** | User is not permitted to set log level |  -  |

