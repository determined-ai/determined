# TerminatedApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**deleteAllTerminated**](TerminatedApi.md#deleteAllTerminated) | **DELETE** /terminated/{owner} | Removes all terminated environments belonging to the given owner
[**deleteTerminated**](TerminatedApi.md#deleteTerminated) | **DELETE** /terminated/{owner}/environments/{environment} | Removes a terminated environment
[**getTerminated**](TerminatedApi.md#getTerminated) | **GET** /terminated/{owner}/environments/{environment} | Gets a specific terminated environment launched by the given owner
[**getTerminatedACLs**](TerminatedApi.md#getTerminatedACLs) | **GET** /terminated/{owner}/acls | Gets the ACLs that control who can manage the terminated environments belonging to the given owner
[**listAllTerminated**](TerminatedApi.md#listAllTerminated) | **GET** /terminated | Gets all terminated environments that the user can view
[**listOwnedTerminated**](TerminatedApi.md#listOwnedTerminated) | **GET** /terminated/{owner} | Gets all terminated environments belonging to the given owner
[**setTerminatedACLs**](TerminatedApi.md#setTerminatedACLs) | **PUT** /terminated/{owner}/acls | Sets the ACLs that control who can manage the terminated environments belonging to the given owner



## deleteAllTerminated

> deleteAllTerminated(owner)

Removes all terminated environments belonging to the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        try {
            apiInstance.deleteAllTerminated(owner);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#deleteAllTerminated");
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

null (empty response body)

### Authorization

[BearerAuth](../README.md#BearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: Not defined


### HTTP response details
| Status code | Description | Response headers |
|-------------|-------------|------------------|
| **204** | All owned environments were removed |  -  |
| **403** | User does not have permission to manage terminated environment with the given owner |  -  |


## deleteTerminated

> deleteTerminated(owner, environment)

Removes a terminated environment

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            apiInstance.deleteTerminated(owner, environment);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#deleteTerminated");
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
| **204** | The terminated environment was deleted |  -  |
| **403** | User does not have permission to manage terminated environment with the given owner |  -  |
| **404** | The given environment does not exist or has already been deleted permanently |  -  |


## getTerminated

> com.cray.analytics.capsules.model.EnvironmentManifest getTerminated(owner, environment)

Gets a specific terminated environment launched by the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        String environment = "abcdef1234"; // String | The environment that you wish to access
        try {
            com.cray.analytics.capsules.model.EnvironmentManifest result = apiInstance.getTerminated(owner, environment);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#getTerminated");
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
| **200** | A manifest describing the terminated environment |  -  |
| **301** | The given environment is still running and should be looked up elsewhere |  -  |
| **403** | User does not have permission to view terminated environment with the given owner |  -  |
| **404** | The given environment does not exist or has an invalid state |  -  |


## getTerminatedACLs

> com.cray.analytics.capsules.security.ACL getTerminatedACLs(owner)

Gets the ACLs that control who can manage the terminated environments belonging to the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        try {
            com.cray.analytics.capsules.security.ACL result = apiInstance.getTerminatedACLs(owner);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#getTerminatedACLs");
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
| **200** | ACLs for the given owners terminated environments |  -  |
| **403** | User does not have permission to manage terminated environment ACLs with the given owner |  -  |


## listAllTerminated

> Map&lt;String, List&lt;com.cray.analytics.capsules.model.DispatchInfo&gt;&gt; listAllTerminated(limit, offset, reverse, eventLimit, state)

Gets all terminated environments that the user can view

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        Integer limit = 56; // Integer | Number of results to limit to, used in conjunction with offset to page through results
        Integer offset = 0; // Integer | Number of results to offset by, used in conjunction with limit to page through results
        Boolean reverse = false; // Boolean | Whether to reverse the default sort order in the returned results
        Integer eventLimit = 56; // Integer | Number of events to limit to per DispatchInfo
        List<com.cray.analytics.capsules.model.DispatchState> state = Arrays.asList(); // List<com.cray.analytics.capsules.model.DispatchState> | Results must be in the given state(s)
        try {
            Map<String, List<com.cray.analytics.capsules.model.DispatchInfo>> result = apiInstance.listAllTerminated(limit, offset, reverse, eventLimit, state);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#listAllTerminated");
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
| **200** | List of terminated environments |  -  |
| **403** | User does not have permission to view terminated environment with the given owner |  -  |


## listOwnedTerminated

> Map&lt;String, List&lt;com.cray.analytics.capsules.model.DispatchInfo&gt;&gt; listOwnedTerminated(owner, limit, offset, reverse, eventLimit, state)

Gets all terminated environments belonging to the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        Integer limit = 56; // Integer | Number of results to limit to, used in conjunction with offset to page through results
        Integer offset = 0; // Integer | Number of results to offset by, used in conjunction with limit to page through results
        Boolean reverse = false; // Boolean | Whether to reverse the default sort order in the returned results
        Integer eventLimit = 56; // Integer | Number of events to limit to per DispatchInfo
        List<com.cray.analytics.capsules.model.DispatchState> state = Arrays.asList(); // List<com.cray.analytics.capsules.model.DispatchState> | Results must be in the given state(s)
        try {
            Map<String, List<com.cray.analytics.capsules.model.DispatchInfo>> result = apiInstance.listOwnedTerminated(owner, limit, offset, reverse, eventLimit, state);
            System.out.println(result);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#listOwnedTerminated");
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
| **200** | List of owned terminated environments |  -  |
| **403** | User does not have permission to view terminated environment with the given owner |  -  |


## setTerminatedACLs

> setTerminatedACLs(owner, comCrayAnalyticsCapsulesSecurityACL)

Sets the ACLs that control who can manage the terminated environments belonging to the given owner

### Example

```java
// Import classes:
import com.cray.analytics.capsules.dispatch.client.invoker.ApiClient;
import com.cray.analytics.capsules.dispatch.client.invoker.ApiException;
import com.cray.analytics.capsules.dispatch.client.invoker.Configuration;
import com.cray.analytics.capsules.dispatch.client.invoker.auth.*;
import com.cray.analytics.capsules.dispatch.client.invoker.models.*;
import com.cray.analytics.capsules.dispatch.client.api.TerminatedApi;

public class Example {
    public static void main(String[] args) {
        ApiClient defaultClient = Configuration.getDefaultApiClient();
        defaultClient.setBasePath("http://localhost");
        
        // Configure HTTP bearer authorization: BearerAuth
        HttpBearerAuth BearerAuth = (HttpBearerAuth) defaultClient.getAuthentication("BearerAuth");
        BearerAuth.setBearerToken("BEARER TOKEN");

        TerminatedApi apiInstance = new TerminatedApi(defaultClient);
        String owner = "lhamilton"; // String | The username of the user whose resources that you wish to access
        com.cray.analytics.capsules.security.ACL comCrayAnalyticsCapsulesSecurityACL = new com.cray.analytics.capsules.security.ACL(); // com.cray.analytics.capsules.security.ACL | The ACLs to set
        try {
            apiInstance.setTerminatedACLs(owner, comCrayAnalyticsCapsulesSecurityACL);
        } catch (ApiException e) {
            System.err.println("Exception when calling TerminatedApi#setTerminatedACLs");
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
| **403** | User does not have permission to manage terminated environment ACLs with the given owner |  -  |

