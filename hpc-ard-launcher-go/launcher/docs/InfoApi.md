# InfoApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getApi**](InfoApi.md#getApi) | **GET** /info/api | Gets the API definition
[**getApiVersion**](InfoApi.md#getApiVersion) | **GET** /info/version/api | Gets the API version
[**getServerRevision**](InfoApi.md#getServerRevision) | **GET** /info/version/revision | Gets the Server revision
[**getServerVersion**](InfoApi.md#getServerVersion) | **GET** /info/version/server | Gets the Server version



## getApi

> Map&lt;String, Object&gt; getApi()

Gets the API definition

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.InfoApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        InfoApi apiInstance = new InfoApi(defaultClient);
        try {
            Map<String, Object> result = apiInstance.getApi();
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling InfoApi#getApi");
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

**Map&lt;String, Object&gt;**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | API Definition |  -  |


## getApiVersion

> String getApiVersion()

Gets the API version

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.InfoApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        InfoApi apiInstance = new InfoApi(defaultClient);
        try {
            String result = apiInstance.getApiVersion();
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling InfoApi#getApiVersion");
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

**String**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | API Version |  -  |


## getServerRevision

> String getServerRevision()

Gets the Server revision

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.InfoApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        InfoApi apiInstance = new InfoApi(defaultClient);
        try {
            String result = apiInstance.getServerRevision();
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling InfoApi#getServerRevision");
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

**String**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Server Revision |  -  |


## getServerVersion

> String getServerVersion()

Gets the Server version

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.InfoApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        InfoApi apiInstance = new InfoApi(defaultClient);
        try {
            String result = apiInstance.getServerVersion();
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling InfoApi#getServerVersion");
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

**String**

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | Server Version |  -  |

