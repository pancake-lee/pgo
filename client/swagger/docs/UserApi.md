# {{classname}}

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**UserDelUserDeptAssoc**](UserApi.md#UserDelUserDeptAssoc) | **Delete** /user-dept-assoc | 
[**UserEditUserName**](UserApi.md#UserEditUserName) | **Patch** /user | 
[**UserGetUserPermissions**](UserApi.md#UserGetUserPermissions) | **Get** /user/permissions | 
[**UserLogin**](UserApi.md#UserLogin) | **Post** /user/token | 

# **UserDelUserDeptAssoc**
> ApiEmpty UserDelUserDeptAssoc(ctx, optional)


从部门中移除用户

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserApiUserDelUserDeptAssocOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserApiUserDelUserDeptAssocOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userID** | **optional.Int32**|  | 
 **deptID** | **optional.Int32**|  | 

### Return type

[**ApiEmpty**](api.Empty.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserEditUserName**
> ApiEmpty UserEditUserName(ctx, body)


修改用户名

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiEditUserNameRequest**](ApiEditUserNameRequest.md)|  | 

### Return type

[**ApiEmpty**](api.Empty.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserGetUserPermissions**
> ApiGetUserPermissionsResponse UserGetUserPermissions(ctx, optional)


--------------------------------------------------  查询用户在某项目内所有权限，去掉角色这一层，直接列出所有拥有的权限值

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserApiUserGetUserPermissionsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserApiUserGetUserPermissionsOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userID** | **optional.Int32**|  | 
 **projectID** | **optional.Int32**|  | 

### Return type

[**ApiGetUserPermissionsResponse**](api.GetUserPermissionsResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserLogin**
> ApiLoginResponse UserLogin(ctx, body)


登录或注册，其实可以理解为只是通过用户账号密码新建一个token，用于其他接口鉴权

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiLoginRequest**](ApiLoginRequest.md)|  | 

### Return type

[**ApiLoginResponse**](api.LoginResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

