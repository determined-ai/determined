# LaunchApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**addCredential**](LaunchApi.md#addCredential) | **PUT** /launch/credentials/{owner}/{name} | Creates/updates a credential that the Dispatch Centre can use to launch environments on behalf of the user
[**hasCredential**](LaunchApi.md#hasCredential) | **HEAD** /launch/credentials/{owner}/{name} | Determines whether a given credential has been provided
[**launch**](LaunchApi.md#launch) | **PUT** /launch | Launches the runtime environment described by the provided manifest in a synchronous manner
[**launchAsync**](LaunchApi.md#launchAsync) | **PUT** /launch/async | Launches the runtime environment described by the provided manifest in an asynchronous manner
[**removeCredential**](LaunchApi.md#removeCredential) | **DELETE** /launch/credentials/{owner}/{name} | Removes a credential



## addCredential

> addCredential(owner, name, body)

Creates/updates a credential that the Dispatch Centre can use to launch environments on behalf of the user

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.LaunchApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        LaunchApi apiInstance = new LaunchApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String name = "track-analysis"; // String | The name of the resource that you wish to access
        File body = new File("/path/to/file"); // File | The credential data to store
        try {
            apiInstance.addCredential(owner, name, body);
        } catch (ApiException e) {
            System.err.println("Exception when calling LaunchApi#addCredential");
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
 **name** | **String**| The name of the resource that you wish to access |
 **body** | **File**| The credential data to store |

### Return type

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/octet-stream
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Credential was created/updated |  -  |
| **403** | User does not have permission to manage credentials for the given owner |  -  |
| **500** | Could not add credential |  -  |


## hasCredential

> hasCredential(owner, name)

Determines whether a given credential has been provided

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.LaunchApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        LaunchApi apiInstance = new LaunchApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String name = "track-analysis"; // String | The name of the resource that you wish to access
        try {
            apiInstance.hasCredential(owner, name);
        } catch (ApiException e) {
            System.err.println("Exception when calling LaunchApi#hasCredential");
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
 **name** | **String**| The name of the resource that you wish to access |

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
| **204** | The specified credential exists |  -  |
| **404** | The specified credential does not exist |  -  |


## launch

> com.cray.analytics.capsules.model.DispatchInfo launch(comCrayAnalyticsCapsulesModelEnvironmentManifest, impersonate, dispatchId)

Launches the runtime environment described by the provided manifest in a synchronous manner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.LaunchApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        LaunchApi apiInstance = new LaunchApi(defaultClient);
        com.cray.analytics.capsules.model.EnvironmentManifest comCrayAnalyticsCapsulesModelEnvironmentManifest = new com.cray.analytics.capsules.model.EnvironmentManifest(); // com.cray.analytics.capsules.model.EnvironmentManifest | The manifest to launch
        String impersonate = "impersonate_example"; // String | User to impersonate (user encoded in authorization token must be configured as an administrator)
        String dispatchId = "dispatchId_example"; // String | Force the use of a specific DispatchID instead of generation of a new one.
        try {
            com.cray.analytics.capsules.model.DispatchInfo result = apiInstance.launch(comCrayAnalyticsCapsulesModelEnvironmentManifest, impersonate, dispatchId);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling LaunchApi#launch");
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
 **comCrayAnalyticsCapsulesModelEnvironmentManifest** | [**com.cray.analytics.capsules.model.EnvironmentManifest**](com.cray.analytics.capsules.model.EnvironmentManifest.md)| The manifest to launch |
 **impersonate** | **String**| User to impersonate (user encoded in authorization token must be configured as an administrator) | [optional]
 **dispatchId** | **String**| Force the use of a specific DispatchID instead of generation of a new one. | [optional]

### Return type

[**com.cray.analytics.capsules.model.DispatchInfo**](com.cray.analytics.capsules.model.DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **201** | Runtime environment was launched successfully |  -  |
| **400** | Invalid environment manifest |  -  |
| **403** | User does not have permission to launch environments as the specified impersonation user |  -  |
| **500** | Runtime environment couldn&#39;t be launched |  -  |


## launchAsync

> com.cray.analytics.capsules.model.DispatchInfo launchAsync(comCrayAnalyticsCapsulesModelEnvironmentManifest, impersonate, dispatchId)

Launches the runtime environment described by the provided manifest in an asynchronous manner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.LaunchApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        LaunchApi apiInstance = new LaunchApi(defaultClient);
        com.cray.analytics.capsules.model.EnvironmentManifest comCrayAnalyticsCapsulesModelEnvironmentManifest = new com.cray.analytics.capsules.model.EnvironmentManifest(); // com.cray.analytics.capsules.model.EnvironmentManifest | The manifest to launch
        String impersonate = "impersonate_example"; // String | User to impersonate (user encoded in authorization token must be configured as an administrator)
        String dispatchId = "dispatchId_example"; // String | Force the use of a specific DispatchID instead of generation of a new one.
        try {
            com.cray.analytics.capsules.model.DispatchInfo result = apiInstance.launchAsync(comCrayAnalyticsCapsulesModelEnvironmentManifest, impersonate, dispatchId);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling LaunchApi#launchAsync");
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
 **comCrayAnalyticsCapsulesModelEnvironmentManifest** | [**com.cray.analytics.capsules.model.EnvironmentManifest**](com.cray.analytics.capsules.model.EnvironmentManifest.md)| The manifest to launch |
 **impersonate** | **String**| User to impersonate (user encoded in authorization token must be configured as an administrator) | [optional]
 **dispatchId** | **String**| Force the use of a specific DispatchID instead of generation of a new one. | [optional]

### Return type

[**com.cray.analytics.capsules.model.DispatchInfo**](com.cray.analytics.capsules.model.DispatchInfo.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: application/json, application/yaml
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **202** | Runtime environment was accepted for launch |  -  |
| **400** | Invalid environment manifest |  -  |
| **403** | User does not have permission to launch environments as the specified impersonation user |  -  |
| **500** | Runtime environment couldn&#39;t be launched |  -  |


## removeCredential

> removeCredential(owner, name)

Removes a credential

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.LaunchApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        LaunchApi apiInstance = new LaunchApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String name = "track-analysis"; // String | The name of the resource that you wish to access
        try {
            apiInstance.removeCredential(owner, name);
        } catch (ApiException e) {
            System.err.println("Exception when calling LaunchApi#removeCredential");
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
 **name** | **String**| The name of the resource that you wish to access |

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
| **204** | The specified credential was successfully removed |  -  |
| **403** | User does not have permission to manage credentials for the given owner |  -  |
| **404** | The specified credential does not exist |  -  |

