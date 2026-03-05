# {{classname}}

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**UserCURDAddProject**](UserCURDApi.md#UserCURDAddProject) | **Post** /project | 
[**UserCURDAddUser**](UserCURDApi.md#UserCURDAddUser) | **Post** /user | 
[**UserCURDAddUserDept**](UserCURDApi.md#UserCURDAddUserDept) | **Post** /user-dept | 
[**UserCURDAddUserDeptAssoc**](UserCURDApi.md#UserCURDAddUserDeptAssoc) | **Post** /user-dept-assoc | 
[**UserCURDAddUserJob**](UserCURDApi.md#UserCURDAddUserJob) | **Post** /user-job | 
[**UserCURDAddUserProjectAssoc**](UserCURDApi.md#UserCURDAddUserProjectAssoc) | **Post** /user-project-assoc | 
[**UserCURDAddUserRole**](UserCURDApi.md#UserCURDAddUserRole) | **Post** /user-role | 
[**UserCURDAddUserRoleAssoc**](UserCURDApi.md#UserCURDAddUserRoleAssoc) | **Post** /user-role-assoc | 
[**UserCURDAddUserRolePermissionAssoc**](UserCURDApi.md#UserCURDAddUserRolePermissionAssoc) | **Post** /user-role-permission-assoc | 
[**UserCURDDelProjectByIDList**](UserCURDApi.md#UserCURDDelProjectByIDList) | **Delete** /project | 
[**UserCURDDelUserByIDList**](UserCURDApi.md#UserCURDDelUserByIDList) | **Delete** /user | 
[**UserCURDDelUserDeptByIDList**](UserCURDApi.md#UserCURDDelUserDeptByIDList) | **Delete** /user-dept | 
[**UserCURDDelUserJobByIDList**](UserCURDApi.md#UserCURDDelUserJobByIDList) | **Delete** /user-job | 
[**UserCURDDelUserProjectAssocByIDList**](UserCURDApi.md#UserCURDDelUserProjectAssocByIDList) | **Delete** /user-project-assoc | 
[**UserCURDDelUserRoleAssocByIDList**](UserCURDApi.md#UserCURDDelUserRoleAssocByIDList) | **Delete** /user-role-assoc | 
[**UserCURDDelUserRoleByIDList**](UserCURDApi.md#UserCURDDelUserRoleByIDList) | **Delete** /user-role | 
[**UserCURDDelUserRolePermissionAssocByIDList**](UserCURDApi.md#UserCURDDelUserRolePermissionAssocByIDList) | **Delete** /user-role-permission-assoc | 
[**UserCURDGetProjectList**](UserCURDApi.md#UserCURDGetProjectList) | **Get** /project | 
[**UserCURDGetUserDeptAssocList**](UserCURDApi.md#UserCURDGetUserDeptAssocList) | **Get** /user-dept-assoc | 
[**UserCURDGetUserDeptList**](UserCURDApi.md#UserCURDGetUserDeptList) | **Get** /user-dept | 
[**UserCURDGetUserJobList**](UserCURDApi.md#UserCURDGetUserJobList) | **Get** /user-job | 
[**UserCURDGetUserList**](UserCURDApi.md#UserCURDGetUserList) | **Get** /user | 
[**UserCURDGetUserProjectAssocList**](UserCURDApi.md#UserCURDGetUserProjectAssocList) | **Get** /user-project-assoc | 
[**UserCURDGetUserRoleAssocList**](UserCURDApi.md#UserCURDGetUserRoleAssocList) | **Get** /user-role-assoc | 
[**UserCURDGetUserRoleList**](UserCURDApi.md#UserCURDGetUserRoleList) | **Get** /user-role | 
[**UserCURDGetUserRolePermissionAssocList**](UserCURDApi.md#UserCURDGetUserRolePermissionAssocList) | **Get** /user-role-permission-assoc | 
[**UserCURDUpdateProject**](UserCURDApi.md#UserCURDUpdateProject) | **Patch** /project | 
[**UserCURDUpdateUserDept**](UserCURDApi.md#UserCURDUpdateUserDept) | **Patch** /user-dept | 
[**UserCURDUpdateUserDeptAssoc**](UserCURDApi.md#UserCURDUpdateUserDeptAssoc) | **Patch** /user-dept-assoc | 
[**UserCURDUpdateUserJob**](UserCURDApi.md#UserCURDUpdateUserJob) | **Patch** /user-job | 
[**UserCURDUpdateUserProjectAssoc**](UserCURDApi.md#UserCURDUpdateUserProjectAssoc) | **Patch** /user-project-assoc | 
[**UserCURDUpdateUserRole**](UserCURDApi.md#UserCURDUpdateUserRole) | **Patch** /user-role | 
[**UserCURDUpdateUserRoleAssoc**](UserCURDApi.md#UserCURDUpdateUserRoleAssoc) | **Patch** /user-role-assoc | 
[**UserCURDUpdateUserRolePermissionAssoc**](UserCURDApi.md#UserCURDUpdateUserRolePermissionAssoc) | **Patch** /user-role-permission-assoc | 

# **UserCURDAddProject**
> ApiAddProjectResponse UserCURDAddProject(ctx, body)


--------------------------------------------------  tbl : project

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddProjectRequest**](ApiAddProjectRequest.md)|  | 

### Return type

[**ApiAddProjectResponse**](api.AddProjectResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

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

# **UserCURDAddUserProjectAssoc**
> ApiAddUserProjectAssocResponse UserCURDAddUserProjectAssoc(ctx, body)


--------------------------------------------------  tbl : user_project_assoc

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserProjectAssocRequest**](ApiAddUserProjectAssocRequest.md)|  | 

### Return type

[**ApiAddUserProjectAssocResponse**](api.AddUserProjectAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDAddUserRole**
> ApiAddUserRoleResponse UserCURDAddUserRole(ctx, body)


--------------------------------------------------  tbl : user_role

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserRoleRequest**](ApiAddUserRoleRequest.md)|  | 

### Return type

[**ApiAddUserRoleResponse**](api.AddUserRoleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDAddUserRoleAssoc**
> ApiAddUserRoleAssocResponse UserCURDAddUserRoleAssoc(ctx, body)


--------------------------------------------------  tbl : user_role_assoc

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserRoleAssocRequest**](ApiAddUserRoleAssocRequest.md)|  | 

### Return type

[**ApiAddUserRoleAssocResponse**](api.AddUserRoleAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDAddUserRolePermissionAssoc**
> ApiAddUserRolePermissionAssocResponse UserCURDAddUserRolePermissionAssoc(ctx, body)


--------------------------------------------------  tbl : user_role_permission_assoc

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddUserRolePermissionAssocRequest**](ApiAddUserRolePermissionAssocRequest.md)|  | 

### Return type

[**ApiAddUserRolePermissionAssocResponse**](api.AddUserRolePermissionAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDDelProjectByIDList**
> ApiEmpty UserCURDDelProjectByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelProjectByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelProjectByIDListOpts struct
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

# **UserCURDDelUserProjectAssocByIDList**
> ApiEmpty UserCURDDelUserProjectAssocByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserProjectAssocByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserProjectAssocByIDListOpts struct
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

# **UserCURDDelUserRoleAssocByIDList**
> ApiEmpty UserCURDDelUserRoleAssocByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserRoleAssocByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserRoleAssocByIDListOpts struct
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

# **UserCURDDelUserRoleByIDList**
> ApiEmpty UserCURDDelUserRoleByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserRoleByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserRoleByIDListOpts struct
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

# **UserCURDDelUserRolePermissionAssocByIDList**
> ApiEmpty UserCURDDelUserRolePermissionAssocByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDDelUserRolePermissionAssocByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDDelUserRolePermissionAssocByIDListOpts struct
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

# **UserCURDGetProjectList**
> ApiGetProjectListResponse UserCURDGetProjectList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetProjectListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetProjectListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **projNameList** | [**optional.Interface of []string**](string.md)|  | 
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetProjectListResponse**](api.GetProjectListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserDeptAssocList**
> ApiGetUserDeptAssocListResponse UserCURDGetUserDeptAssocList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserDeptAssocListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserDeptAssocListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userIDList** | [**optional.Interface of []int32**](int32.md)|  | 
 **deptIDList** | [**optional.Interface of []int32**](int32.md)|  | 
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

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
 **deptPathList** | [**optional.Interface of []string**](string.md)|  | 
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
 **jobNameList** | [**optional.Interface of []string**](string.md)|  | 
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
 **userNameList** | [**optional.Interface of []string**](string.md)|  | 
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserListResponse**](api.GetUserListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserProjectAssocList**
> ApiGetUserProjectAssocListResponse UserCURDGetUserProjectAssocList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserProjectAssocListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserProjectAssocListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userIDList** | [**optional.Interface of []int32**](int32.md)|  | 
 **projIDList** | [**optional.Interface of []int32**](int32.md)|  | 
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserProjectAssocListResponse**](api.GetUserProjectAssocListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserRoleAssocList**
> ApiGetUserRoleAssocListResponse UserCURDGetUserRoleAssocList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserRoleAssocListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserRoleAssocListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **userIDList** | [**optional.Interface of []int32**](int32.md)|  | 
 **roleIDList** | [**optional.Interface of []int32**](int32.md)|  | 
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserRoleAssocListResponse**](api.GetUserRoleAssocListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserRoleList**
> ApiGetUserRoleListResponse UserCURDGetUserRoleList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserRoleListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserRoleListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserRoleListResponse**](api.GetUserRoleListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDGetUserRolePermissionAssocList**
> ApiGetUserRolePermissionAssocListResponse UserCURDGetUserRolePermissionAssocList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***UserCURDApiUserCURDGetUserRolePermissionAssocListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UserCURDApiUserCURDGetUserRolePermissionAssocListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetUserRolePermissionAssocListResponse**](api.GetUserRolePermissionAssocListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateProject**
> ApiUpdateProjectResponse UserCURDUpdateProject(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateProjectRequest**](ApiUpdateProjectRequest.md)|  | 

### Return type

[**ApiUpdateProjectResponse**](api.UpdateProjectResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserDept**
> ApiUpdateUserDeptResponse UserCURDUpdateUserDept(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserDeptRequest**](ApiUpdateUserDeptRequest.md)|  | 

### Return type

[**ApiUpdateUserDeptResponse**](api.UpdateUserDeptResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserDeptAssoc**
> ApiUpdateUserDeptAssocResponse UserCURDUpdateUserDeptAssoc(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserDeptAssocRequest**](ApiUpdateUserDeptAssocRequest.md)|  | 

### Return type

[**ApiUpdateUserDeptAssocResponse**](api.UpdateUserDeptAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserJob**
> ApiUpdateUserJobResponse UserCURDUpdateUserJob(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserJobRequest**](ApiUpdateUserJobRequest.md)|  | 

### Return type

[**ApiUpdateUserJobResponse**](api.UpdateUserJobResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserProjectAssoc**
> ApiUpdateUserProjectAssocResponse UserCURDUpdateUserProjectAssoc(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserProjectAssocRequest**](ApiUpdateUserProjectAssocRequest.md)|  | 

### Return type

[**ApiUpdateUserProjectAssocResponse**](api.UpdateUserProjectAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserRole**
> ApiUpdateUserRoleResponse UserCURDUpdateUserRole(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserRoleRequest**](ApiUpdateUserRoleRequest.md)|  | 

### Return type

[**ApiUpdateUserRoleResponse**](api.UpdateUserRoleResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserRoleAssoc**
> ApiUpdateUserRoleAssocResponse UserCURDUpdateUserRoleAssoc(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserRoleAssocRequest**](ApiUpdateUserRoleAssocRequest.md)|  | 

### Return type

[**ApiUpdateUserRoleAssocResponse**](api.UpdateUserRoleAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UserCURDUpdateUserRolePermissionAssoc**
> ApiUpdateUserRolePermissionAssocResponse UserCURDUpdateUserRolePermissionAssoc(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateUserRolePermissionAssocRequest**](ApiUpdateUserRolePermissionAssocRequest.md)|  | 

### Return type

[**ApiUpdateUserRolePermissionAssocResponse**](api.UpdateUserRolePermissionAssocResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

