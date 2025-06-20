// Code generated by protoc-gen-go-http. DO NOT EDIT.
// versions:
// - protoc-gen-go-http v2.8.4
// - protoc             v6.31.0
// source: z_taskService.gen.proto

package api

import (
	context "context"
	http "github.com/go-kratos/kratos/v2/transport/http"
	binding "github.com/go-kratos/kratos/v2/transport/http/binding"
)

// This is a compile-time assertion to ensure that this generated file
// is compatible with the kratos package it is being compiled against.
var _ = new(context.Context)
var _ = binding.EncodeURL

const _ = http.SupportPackageIsVersion1

const OperationTaskCURDAddTask = "/api.taskCURD/AddTask"
const OperationTaskCURDDelTaskByIDList = "/api.taskCURD/DelTaskByIDList"
const OperationTaskCURDGetTaskList = "/api.taskCURD/GetTaskList"
const OperationTaskCURDUpdateTask = "/api.taskCURD/UpdateTask"

type TaskCURDHTTPServer interface {
	// AddTask --------------------------------------------------
	// tbl : task
	AddTask(context.Context, *AddTaskRequest) (*AddTaskResponse, error)
	DelTaskByIDList(context.Context, *DelTaskByIDListRequest) (*Empty, error)
	GetTaskList(context.Context, *GetTaskListRequest) (*GetTaskListResponse, error)
	UpdateTask(context.Context, *UpdateTaskRequest) (*UpdateTaskResponse, error)
}

func RegisterTaskCURDHTTPServer(s *http.Server, srv TaskCURDHTTPServer) {
	r := s.Route("/")
	r.POST("/task", _TaskCURD_AddTask0_HTTP_Handler(srv))
	r.GET("/task", _TaskCURD_GetTaskList0_HTTP_Handler(srv))
	r.PATCH("/task", _TaskCURD_UpdateTask0_HTTP_Handler(srv))
	r.DELETE("/task", _TaskCURD_DelTaskByIDList0_HTTP_Handler(srv))
}

func _TaskCURD_AddTask0_HTTP_Handler(srv TaskCURDHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in AddTaskRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationTaskCURDAddTask)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.AddTask(ctx, req.(*AddTaskRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*AddTaskResponse)
		return ctx.Result(200, reply)
	}
}

func _TaskCURD_GetTaskList0_HTTP_Handler(srv TaskCURDHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in GetTaskListRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationTaskCURDGetTaskList)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.GetTaskList(ctx, req.(*GetTaskListRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*GetTaskListResponse)
		return ctx.Result(200, reply)
	}
}

func _TaskCURD_UpdateTask0_HTTP_Handler(srv TaskCURDHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in UpdateTaskRequest
		if err := ctx.Bind(&in); err != nil {
			return err
		}
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationTaskCURDUpdateTask)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.UpdateTask(ctx, req.(*UpdateTaskRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*UpdateTaskResponse)
		return ctx.Result(200, reply)
	}
}

func _TaskCURD_DelTaskByIDList0_HTTP_Handler(srv TaskCURDHTTPServer) func(ctx http.Context) error {
	return func(ctx http.Context) error {
		var in DelTaskByIDListRequest
		if err := ctx.BindQuery(&in); err != nil {
			return err
		}
		http.SetOperation(ctx, OperationTaskCURDDelTaskByIDList)
		h := ctx.Middleware(func(ctx context.Context, req interface{}) (interface{}, error) {
			return srv.DelTaskByIDList(ctx, req.(*DelTaskByIDListRequest))
		})
		out, err := h(ctx, &in)
		if err != nil {
			return err
		}
		reply := out.(*Empty)
		return ctx.Result(200, reply)
	}
}

type TaskCURDHTTPClient interface {
	AddTask(ctx context.Context, req *AddTaskRequest, opts ...http.CallOption) (rsp *AddTaskResponse, err error)
	DelTaskByIDList(ctx context.Context, req *DelTaskByIDListRequest, opts ...http.CallOption) (rsp *Empty, err error)
	GetTaskList(ctx context.Context, req *GetTaskListRequest, opts ...http.CallOption) (rsp *GetTaskListResponse, err error)
	UpdateTask(ctx context.Context, req *UpdateTaskRequest, opts ...http.CallOption) (rsp *UpdateTaskResponse, err error)
}

type TaskCURDHTTPClientImpl struct {
	cc *http.Client
}

func NewTaskCURDHTTPClient(client *http.Client) TaskCURDHTTPClient {
	return &TaskCURDHTTPClientImpl{client}
}

func (c *TaskCURDHTTPClientImpl) AddTask(ctx context.Context, in *AddTaskRequest, opts ...http.CallOption) (*AddTaskResponse, error) {
	var out AddTaskResponse
	pattern := "/task"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationTaskCURDAddTask))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "POST", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *TaskCURDHTTPClientImpl) DelTaskByIDList(ctx context.Context, in *DelTaskByIDListRequest, opts ...http.CallOption) (*Empty, error) {
	var out Empty
	pattern := "/task"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationTaskCURDDelTaskByIDList))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "DELETE", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *TaskCURDHTTPClientImpl) GetTaskList(ctx context.Context, in *GetTaskListRequest, opts ...http.CallOption) (*GetTaskListResponse, error) {
	var out GetTaskListResponse
	pattern := "/task"
	path := binding.EncodeURL(pattern, in, true)
	opts = append(opts, http.Operation(OperationTaskCURDGetTaskList))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "GET", path, nil, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *TaskCURDHTTPClientImpl) UpdateTask(ctx context.Context, in *UpdateTaskRequest, opts ...http.CallOption) (*UpdateTaskResponse, error) {
	var out UpdateTaskResponse
	pattern := "/task"
	path := binding.EncodeURL(pattern, in, false)
	opts = append(opts, http.Operation(OperationTaskCURDUpdateTask))
	opts = append(opts, http.PathTemplate(pattern))
	err := c.cc.Invoke(ctx, "PATCH", path, in, &out, opts...)
	if err != nil {
		return nil, err
	}
	return &out, nil
}
