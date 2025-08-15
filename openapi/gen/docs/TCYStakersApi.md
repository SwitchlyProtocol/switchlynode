# \SWCYStakersApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TcyStaker**](SWCYStakersApi.md#TcyStaker) | **Get** /switchly/tcy_staker/{address} | 
[**TcyStakers**](SWCYStakersApi.md#TcyStakers) | **Get** /switchly/tcy_stakers | 



## TcyStaker

> SWCYStaker TcyStaker(ctx, address).Height(height).Execute()





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
    resp, r, err := apiClient.SWCYStakersApi.TcyStaker(context.Background(), address).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWCYStakersApi.TcyStaker``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `TcyStaker`: SWCYStaker
    fmt.Fprintf(os.Stdout, "Response from `SWCYStakersApi.TcyStaker`: %v\n", resp)
}
```

### Path Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**address** | **string** |  | 

### Other Parameters

Other parameters are passed through a pointer to a apiTcyStakerRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**SWCYStaker**](SWCYStaker.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## TcyStakers

> []SWCYStakerSummary TcyStakers(ctx).Height(height).Execute()





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
    resp, r, err := apiClient.SWCYStakersApi.TcyStakers(context.Background()).Height(height).Execute()
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error when calling `SWCYStakersApi.TcyStakers``: %v\n", err)
        fmt.Fprintf(os.Stderr, "Full HTTP response: %v\n", r)
    }
    // response from `TcyStakers`: []SWCYStakerSummary
    fmt.Fprintf(os.Stdout, "Response from `SWCYStakersApi.TcyStakers`: %v\n", resp)
}
```

### Path Parameters



### Other Parameters

Other parameters are passed through a pointer to a apiTcyStakersRequest struct via the builder pattern


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **height** | **int64** | optional block height, defaults to current tip | 

### Return type

[**[]SWCYStakerSummary**](SWCYStakerSummary.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

