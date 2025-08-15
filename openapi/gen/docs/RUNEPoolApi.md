# \SWITCHPoolApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**SWITCHPool**](SWITCHPoolApi.md#SWITCHPool) | **Get** /switchly/runepool | 
[**SWITCHProvider**](SWITCHPoolApi.md#SWITCHProvider) | **Get** /switchly/rune_provider/{address} | 
[**SWITCHProviders**](SWITCHPoolApi.md#SWITCHProviders) | **Get** /switchly/rune_providers | 



## SWITCHPool

> SWITCHPoolResponse SWITCHPool(ctx).Height(height).Execute()





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
    height := int64(789) // int64 | optional block height, defaults to current tip (optional)

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.SWITCHPoolApi.SWITCHPool(context.Background()).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWITCHPoolApi.SWITCHPool``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `SWITCHPool`: SWITCHPoolResponse
    fmt.Fprintf(os.Stdout, "Response from `SWITCHPoolApi.SWITCHPool`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSWITCHPoolRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**SWITCHPoolResponse**](SWITCHPoolResponse.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SWITCHProvider

> SWITCHProvider SWITCHProvider(ctx, address).Height(height).Execute()





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
    address := "thor1zupk5lmc84r2dh738a9g3zscavannjy3nzplwt" // string | 
    height := int64(789) // int64 | optional block height, defaults to current tip (optional)

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.SWITCHPoolApi.SWITCHProvider(context.Background(), address).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWITCHPoolApi.SWITCHProvider``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `SWITCHProvider`: SWITCHProvider
    fmt.Fprintf(os.Stdout, "Response from `SWITCHPoolApi.SWITCHProvider`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**address** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiSWITCHProviderRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**SWITCHProvider**](SWITCHProvider.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SWITCHProviders

> []SWITCHProvider SWITCHProviders(ctx).Height(height).Execute()





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
    height := int64(789) // int64 | optional block height, defaults to current tip (optional)

    configuration := openapiclient.NewConfiguration()
    apiClient := openapiclient.NewAPIClient(configuration)
    resp, r, err := apiClient.SWITCHPoolApi.SWITCHProviders(context.Background()).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWITCHPoolApi.SWITCHProviders``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `SWITCHProviders`: []SWITCHProvider
    fmt.Fprintf(os.Stdout, "Response from `SWITCHPoolApi.SWITCHProviders`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiSWITCHProvidersRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**[]SWITCHProvider**](SWITCHProvider.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

