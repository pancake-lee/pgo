# {{classname}}

All URIs are relative to */*

Method | HTTP request | Description
------------- | ------------- | -------------
[**UserCURDAddUser**](UserCURDApi.md#UserCURDAddUser) | **Post** /user | 
[**UserCURDAddUserDept**](UserCURDApi.md#UserCURDAddUserDept) | **Post** /user-dept | 
[**UserCURDAddUserDeptAssoc**](UserCURDApi.md#UserCURDAddUserDeptAssoc) | **Post** /user-dept-assoc | 
[**UserCURDAddUserJob**](UserCURDApi.md#UserCURDAddUserJob) | **Post** /user-job | 
[**UserCURDDelUserByIDList**](UserCURDApi.md#UserCURDDelUserByIDList) | **Delete** /user | 
[**UserCURDDelUserDeptByIDList**](UserCURDApi.md#UserCURDDelUserDeptByIDList) | **Delete** /user-dept | 
[**UserCURDDelUserJobByIDList**](UserCURDApi.md#UserCURDDelUserJobByIDList) | **Delete** /user-job | 
[**UserCURDGetUserDeptAssocList**](UserCURDApi.md#UserCURDGetUserDeptAssocList) | **Get** /user-dept-assoc | 
[**UserCURDGetUserDeptList**](UserCURDApi.md#UserCURDGetUserDeptList) | **Get** /user-dept | 
[**UserCURDGetUserJobList**](UserCURDApi.md#UserCURDGetUserJobList) | **Get** /user-job | 
[**UserCURDGetUserList**](UserCURDApi.md#UserCURDGetUserList) | **Get** /user | 

# **UserCURDAddUser**
> ApiAddUserResponse UserCURDAddUser(ctx, body)


--------------------------------------------------  tbl : user

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserRequest**](ApiAddUserRequest.md)|  | 

### Return type

[**ApiAddUserResponse**](api.AddUserResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDAddUserDept**
> ApiAddUserDeptResponse UserCURDAddUserDept(ctx, body)


--------------------------------------------------  tbl : user_dept

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserDeptRequest**](ApiAddUserDeptRequest.md)|  | 

### Return type

[**ApiAddUserDeptResponse**](api.AddUserDeptResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDAddUserDeptAssoc**
> ApiAddUserDeptAssocResponse UserCURDAddUserDeptAssoc(ctx, body)


--------------------------------------------------  tbl : user_dept_assoc

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserDeptAssocRequest**](ApiAddUserDeptAssocRequest.md)|  | 

### Return type

[**ApiAddUserDeptAssocResponse**](api.AddUserDeptAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDAddUserJob**
> ApiAddUserJobResponse UserCURDAddUserJob(ctx, body)


--------------------------------------------------  tbl : user_job

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserJobRequest**](ApiAddUserJobRequest.md)|  | 

### Return type

[**ApiAddUserJobResponse**](api.AddUserJobResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDDelUserByIDList**
> ApiEmpty UserCURDDelUserByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserByIDListOpts struct
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

# **UserCURDDelUserDeptByIDList**
> ApiEmpty UserCURDDelUserDeptByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserDeptByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserDeptByIDListOpts struct
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

# **UserCURDDelUserJobByIDList**
> ApiEmpty UserCURDDelUserJobByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserJobByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserJobByIDListOpts struct
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

# **UserCURDGetUserDeptAssocList**
> ApiGetUserDeptAssocListResponse UserCURDGetUserDeptAssocList(ctx, )


### Required Parameters
This endpoint does not need any parameter.

### Return type

[**ApiGetUserDeptAssocListResponse**](api.GetUserDeptAssocListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserDeptList**
> ApiGetUserDeptListResponse UserCURDGetUserDeptList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserDeptListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserDeptListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserDeptListResponse**](api.GetUserDeptListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserJobList**
> ApiGetUserJobListResponse UserCURDGetUserJobList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserJobListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserJobListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserJobListResponse**](api.GetUserJobListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserList**
> ApiGetUserListResponse UserCURDGetUserList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserListResponse**](api.GetUserListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

