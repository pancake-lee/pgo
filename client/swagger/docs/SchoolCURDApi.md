# {{classname}}

All URIs are relative to */*

Method | HTTP request | Description
------------- | ------------- | -------------
[**SchoolCURDAddCourseSwapRequest**](SchoolCURDApi.md#SchoolCURDAddCourseSwapRequest) | **Post** /course-swap-request | 
[**SchoolCURDDelCourseSwapRequestByIDList**](SchoolCURDApi.md#SchoolCURDDelCourseSwapRequestByIDList) | **Delete** /course-swap-request | 
[**SchoolCURDGetCourseSwapRequestList**](SchoolCURDApi.md#SchoolCURDGetCourseSwapRequestList) | **Get** /course-swap-request | 

# **SchoolCURDAddCourseSwapRequest**
> ApiAddCourseSwapRequestResponse SchoolCURDAddCourseSwapRequest(ctx, body)


--------------------------------------------------  tbl : course_swap_request

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddCourseSwapRequestRequest**](ApiAddCourseSwapRequestRequest.md)|  | 

### Return type

[**ApiAddCourseSwapRequestResponse**](api.AddCourseSwapRequestResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **SchoolCURDDelCourseSwapRequestByIDList**
> ApiEmpty SchoolCURDDelCourseSwapRequestByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***SchoolCURDApiSchoolCURDDelCourseSwapRequestByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a SchoolCURDApiSchoolCURDDelCourseSwapRequestByIDListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiEmpty**](api.Empty.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **SchoolCURDGetCourseSwapRequestList**
> ApiGetCourseSwapRequestListResponse SchoolCURDGetCourseSwapRequestList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***SchoolCURDApiSchoolCURDGetCourseSwapRequestListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a SchoolCURDApiSchoolCURDGetCourseSwapRequestListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetCourseSwapRequestListResponse**](api.GetCourseSwapRequestListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

