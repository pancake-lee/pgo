# {{classname}}

All URIs are relative to *http://127.0.0.1:8080*

Method | HTTP request | Description
------------- | ------------- | -------------
[**TaskCURDAddTask**](TaskCURDApi.md#TaskCURDAddTask) | **Post** /task | 
[**TaskCURDDelTaskByIDList**](TaskCURDApi.md#TaskCURDDelTaskByIDList) | **Delete** /task | 
[**TaskCURDGetTaskList**](TaskCURDApi.md#TaskCURDGetTaskList) | **Get** /task | 
[**TaskCURDUpdateTask**](TaskCURDApi.md#TaskCURDUpdateTask) | **Patch** /task | 

# **TaskCURDAddTask**
> ApiAddTaskResponse TaskCURDAddTask(ctx, body)


--------------------------------------------------  tbl : task

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiAddTaskRequest**](ApiAddTaskRequest.md)|  | 

### Return type

[**ApiAddTaskResponse**](api.AddTaskResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TaskCURDDelTaskByIDList**
> ApiEmpty TaskCURDDelTaskByIDList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***TaskCURDApiTaskCURDDelTaskByIDListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a TaskCURDApiTaskCURDDelTaskByIDListOpts struct
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

# **TaskCURDGetTaskList**
> ApiGetTaskListResponse TaskCURDGetTaskList(ctx, optional)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***TaskCURDApiTaskCURDGetTaskListOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a TaskCURDApiTaskCURDGetTaskListOpts struct
Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **iDList** | [**optional.Interface of []int32**](int32.md)|  | 

### Return type

[**ApiGetTaskListResponse**](api.GetTaskListResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **TaskCURDUpdateTask**
> ApiUpdateTaskResponse TaskCURDUpdateTask(ctx, body)


### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **body** | [**ApiUpdateTaskRequest**](ApiUpdateTaskRequest.md)|  | 

### Return type

[**ApiUpdateTaskResponse**](api.UpdateTaskResponse.md)

### Authorization

No authorization required

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

