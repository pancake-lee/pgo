# {{classname}}

All URIs are relative to */*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AbandonCodeCURDAddAbandonCode**](AbandonCodeCURDApi.md#AbandonCodeCURDAddAbandonCode) | **Post** /abandon-code | 
[**AbandonCodeCURDDelAbandonCodeByIdx1List**](AbandonCodeCURDApi.md#AbandonCodeCURDDelAbandonCodeByIdx1List) | **Delete** /abandon-code | 
[**AbandonCodeCURDGetAbandonCodeList**](AbandonCodeCURDApi.md#AbandonCodeCURDGetAbandonCodeList) | **Get** /abandon-code | 

# **AbandonCodeCURDAddAbandonCode**
> ApiAddAbandonCodeResponse AbandonCodeCURDAddAbandonCode(ctx, body)


MARK REPEAT API START 一个表的接口定义  --------------------------------------------------  tbl : abandon_code

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddAbandonCodeRequest**](ApiAddAbandonCodeRequest.md)|  | 

### Return type

[**ApiAddAbandonCodeResponse**](api.AddAbandonCodeResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AbandonCodeCURDDelAbandonCodeByIdx1List**
> ApiEmpty AbandonCodeCURDDelAbandonCodeByIdx1List(ctx, optional)


MARK REMOVE IF NO PRIMARY KEY START

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***AbandonCodeCURDApiAbandonCodeCURDDelAbandonCodeByIdx1ListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a AbandonCodeCURDApiAbandonCodeCURDDelAbandonCodeByIdx1ListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idx1List** | [**optional.Interface of []int32**](int32.md)| MARK REPLACE REQUEST IDX START 替换内容，索引字段 | 

### Return type

[**ApiEmpty**](api.Empty.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AbandonCodeCURDGetAbandonCodeList**
> ApiGetAbandonCodeListResponse AbandonCodeCURDGetAbandonCodeList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***AbandonCodeCURDApiAbandonCodeCURDGetAbandonCodeListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a AbandonCodeCURDApiAbandonCodeCURDGetAbandonCodeListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **idx1List** | [**optional.Interface of []int32**](int32.md)| MARK REPLACE REQUEST IDX START 替换内容，索引字段 | 

### Return type

[**ApiGetAbandonCodeListResponse**](api.GetAbandonCodeListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

