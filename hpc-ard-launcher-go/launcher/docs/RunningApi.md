# RunningApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**getRunning**](RunningApi.md#getRunning) | **GET** /running/{owner}/environments/{environment} | Gets a specific running environment launched by the given owner
[**getRunningACLs**](RunningApi.md#getRunningACLs) | **GET** /running/{owner}/acls | Gets the ACLs that control who can manage the running environments belonging to the given owner
[**getUserInterfaces**](RunningApi.md#getUserInterfaces) | **GET** /running/{owner}/environments/{environment}/uis | Gets the User Interfaces associated with the running environment launched by the given owner
[**listAllRunning**](RunningApi.md#listAllRunning) | **GET** /running | Gets all running environments that the user can view
[**listOwnedRunning**](RunningApi.md#listOwnedRunning) | **GET** /running/{owner} | Gets all running environments launched by the given owner
[**setRunningACLs**](RunningApi.md#setRunningACLs) | **PUT** /running/{owner}/acls | Sets the ACLs that control who can manage the running environments belonging to the given owner
[**terminateAllRunning**](RunningApi.md#terminateAllRunning) | **DELETE** /running/{owner} | Terminates all running environments owned by the given owner
[**terminateRunning**](RunningApi.md#terminateRunning) | **DELETE** /running/{owner}/environments/{environment} | Terminates a running environment
[**terminateRunningAsync**](RunningApi.md#terminateRunningAsync) | **DELETE** /running/{owner}/environments/{environment}/async | Terminates a running environment in an asynchronous manner



## getRunning

> com.cray.analytics.capsules.model.EnvironmentManifest getRunning(owner, environment)

Gets a specific running environment launched by the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            com.cray.analytics.capsules.model.EnvironmentManifest result = apiInstance.getRunning(owner, environment);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#getRunning");
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
| **200** | A manifest describing the running environment |  -  |
| **301** | The given environment has been terminated and can now be looked up elsewhere |  -  |
| **403** | User does not have permission to view environments with the given owner |  -  |
| **404** | The given environment does not exist or is no longer running |  -  |


## getRunningACLs

> com.cray.analytics.capsules.security.ACL getRunningACLs(owner)

Gets the ACLs that control who can manage the running environments belonging to the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        try {
            com.cray.analytics.capsules.security.ACL result = apiInstance.getRunningACLs(owner);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#getRunningACLs");
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

### Return type

[**com.cray.analytics.capsules.security.ACL**](com.cray.analytics.capsules.security.ACL.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | ACLs for the given owners running environments |  -  |
| **403** | User does not have permission to manage running environment ACLs with the given owner |  -  |


## getUserInterfaces

> Map&lt;String, List&lt;com.cray.analytics.capsules.model.UserInterface&gt;&gt; getUserInterfaces(owner, environment)

Gets the User Interfaces associated with the running environment launched by the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            Map<String, List<com.cray.analytics.capsules.model.UserInterface>> result = apiInstance.getUserInterfaces(owner, environment);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#getUserInterfaces");
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

[**Map&lt;String, List&lt;com.cray.analytics.capsules.model.UserInterface&gt;&gt;**](List.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | A list of user interfaces available in the running environment |  -  |
| **301** | The given environment has been terminated and can now be looked up elsewhere |  -  |
| **403** | User does not have permission to view environments with the given owner |  -  |
| **404** | The given environment does not exist |  -  |


## listAllRunning

> Map&lt;String, List&lt;com.cray.analytics.capsules.model.DispatchInfo&gt;&gt; listAllRunning(limit, offset, reverse, eventLimit, state)

Gets all running environments that the user can view

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        Integer limit = 56; // Integer | Number of results to limit to, used in conjunction with offset to page through results
        Integer offset = 0; // Integer | Number of results to offset by, used in conjunction with limit to page through results
        Boolean reverse = false; // Boolean | Whether to reverse the default sort order in the returned results
        Integer eventLimit = 56; // Integer | Number of events to limit to per DispatchInfo
        List<com.cray.analytics.capsules.model.DispatchState> state = Arrays.asList(); // List<com.cray.analytics.capsules.model.DispatchState> | Results must be in the given state(s)
        try {
            Map<String, List<com.cray.analytics.capsules.model.DispatchInfo>> result = apiInstance.listAllRunning(limit, offset, reverse, eventLimit, state);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#listAllRunning");
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
 **limit** | **Integer**| Number of results to limit to, used in conjunction with offset to page through results | [optional]
 **offset** | **Integer**| Number of results to offset by, used in conjunction with limit to page through results | [optional] [default to 0]
 **reverse** | **Boolean**| Whether to reverse the default sort order in the returned results | [optional] [default to false]
 **eventLimit** | **Integer**| Number of events to limit to per DispatchInfo | [optional]
 **state** | [**List&lt;com.cray.analytics.capsules.model.DispatchState&gt;**](com.cray.analytics.capsules.model.DispatchState.md)| Results must be in the given state(s) | [optional]

### Return type

[**Map&lt;String, List&lt;com.cray.analytics.capsules.model.DispatchInfo&gt;&gt;**](List.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of running environments |  -  |
| **403** | User does not have permission to view environments |  -  |


## listOwnedRunning

> Map&lt;String, List&lt;com.cray.analytics.capsules.model.DispatchInfo&gt;&gt; listOwnedRunning(owner, limit, offset, reverse, eventLimit, state)

Gets all running environments launched by the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        Integer limit = 56; // Integer | Number of results to limit to, used in conjunction with offset to page through results
        Integer offset = 0; // Integer | Number of results to offset by, used in conjunction with limit to page through results
        Boolean reverse = false; // Boolean | Whether to reverse the default sort order in the returned results
        Integer eventLimit = 56; // Integer | Number of events to limit to per DispatchInfo
        List<com.cray.analytics.capsules.model.DispatchState> state = Arrays.asList(); // List<com.cray.analytics.capsules.model.DispatchState> | Results must be in the given state(s)
        try {
            Map<String, List<com.cray.analytics.capsules.model.DispatchInfo>> result = apiInstance.listOwnedRunning(owner, limit, offset, reverse, eventLimit, state);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#listOwnedRunning");
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
 **limit** | **Integer**| Number of results to limit to, used in conjunction with offset to page through results | [optional]
 **offset** | **Integer**| Number of results to offset by, used in conjunction with limit to page through results | [optional] [default to 0]
 **reverse** | **Boolean**| Whether to reverse the default sort order in the returned results | [optional] [default to false]
 **eventLimit** | **Integer**| Number of events to limit to per DispatchInfo | [optional]
 **state** | [**List&lt;com.cray.analytics.capsules.model.DispatchState&gt;**](com.cray.analytics.capsules.model.DispatchState.md)| Results must be in the given state(s) | [optional]

### Return type

[**Map&lt;String, List&lt;com.cray.analytics.capsules.model.DispatchInfo&gt;&gt;**](List.md)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json, application/yaml


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **200** | List of owned running environments |  -  |
| **403** | User does not have permission to view environments with the given owner |  -  |


## setRunningACLs

> setRunningACLs(owner, comCrayAnalyticsCapsulesSecurityACL)

Sets the ACLs that control who can manage the running environments belonging to the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        com.cray.analytics.capsules.security.ACL comCrayAnalyticsCapsulesSecurityACL = new com.cray.analytics.capsules.security.ACL(); // com.cray.analytics.capsules.security.ACL | The ACLs to set
        try {
            apiInstance.setRunningACLs(owner, comCrayAnalyticsCapsulesSecurityACL);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#setRunningACLs");
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
 **comCrayAnalyticsCapsulesSecurityACL** | [**com.cray.analytics.capsules.security.ACL**](com.cray.analytics.capsules.security.ACL.md)| The ACLs to set |

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
| **204** | ACLs were successfully updated |  -  |
| **403** | User does not have permission to manage running environment ACLs with the given owner |  -  |


## terminateAllRunning

> terminateAllRunning(owner, force)

Terminates all running environments owned by the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        Boolean force = false; // Boolean | Whether to force termination
        try {
            apiInstance.terminateAllRunning(owner, force);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#terminateAllRunning");
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
 **force** | **Boolean**| Whether to force termination | [optional] [default to false]

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
| **204** | All running environments were terminated successfully |  -  |
| **403** | User does not have permission to terminate environments with the given owner |  -  |
| **500** | Could not terminate all running environments |  -  |


## terminateRunning

> com.cray.analytics.capsules.model.DispatchInfo terminateRunning(owner, environment, force)

Terminates a running environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        Boolean force = false; // Boolean | Whether to force termination
        try {
            com.cray.analytics.capsules.model.DispatchInfo result = apiInstance.terminateRunning(owner, environment, force);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#terminateRunning");
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
 **force** | **Boolean**| Whether to force termination | [optional] [default to false]

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
| **202** | Runtime environment was terminated |  -  |
| **301** | The given environment has been terminated and can now be looked up elsewhere |  -  |
| **403** | User does not have permission to terminate environments with the given owner |  -  |
| **404** | The given environment does not exist or has already been terminated |  -  |
| **500** | Could not terminate running environment |  -  |


## terminateRunningAsync

> com.cray.analytics.capsules.model.DispatchInfo terminateRunningAsync(owner, environment, force)

Terminates a running environment in an asynchronous manner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.RunningApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        RunningApi apiInstance = new RunningApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        Boolean force = false; // Boolean | Whether to force termination
        try {
            com.cray.analytics.capsules.model.DispatchInfo result = apiInstance.terminateRunningAsync(owner, environment, force);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling RunningApi#terminateRunningAsync");
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
 **force** | **Boolean**| Whether to force termination | [optional] [default to false]

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
| **202** | Runtime environment was accepted for termination |  -  |
| **301** | The given environment has already been terminated and can now be looked up elsewhere |  -  |
| **403** | User does not have permission to terminate environments with the given owner |  -  |
| **404** | The given environment does not exist or has already been terminated |  -  |
| **500** | Could not terminate running environment |  -  |

