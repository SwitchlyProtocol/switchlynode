# \SWCYClaimersApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TcyClaimer**](SWCYClaimersApi.md#TcyClaimer) | **Get** /switchly/tcy_claimer/{address} | 
[**TcyClaimers**](SWCYClaimersApi.md#TcyClaimers) | **Get** /switchly/tcy_claimers | 



## TcyClaimer

> SWCYClaimer TcyClaimer(ctx, address).Height(height).Execute()





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
    resp, r, err := apiClient.SWCYClaimersApi.TcyClaimer(context.Background(), address).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWCYClaimersApi.TcyClaimer``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `TcyClaimer`: SWCYClaimer
    fmt.Fprintf(os.Stdout, "Response from `SWCYClaimersApi.TcyClaimer`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**address** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiTcyClaimerRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**SWCYClaimer**](SWCYClaimer.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## TcyClaimers

> []SWCYClaimerSummary TcyClaimers(ctx).Height(height).Execute()





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
    resp, r, err := apiClient.SWCYClaimersApi.TcyClaimers(context.Background()).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWCYClaimersApi.TcyClaimers``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `TcyClaimers`: []SWCYClaimerSummary
    fmt.Fprintf(os.Stdout, "Response from `SWCYClaimersApi.TcyClaimers`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiTcyClaimersRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**[]SWCYClaimerSummary**](SWCYClaimerSummary.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

