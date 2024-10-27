package types

import (
	"time"

	"github.com/zhufuyi/sponge/pkg/ggorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.

// CreateUserJobRequest request params
type CreateUserJobRequest struct {
	JobName string `json:"jobName" binding:""`
}

// UpdateUserJobByIDRequest request params
type UpdateUserJobByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id

	JobName string `json:"jobName" binding:""`
}

// UserJobObjDetail detail
type UserJobObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id

	JobName string `json:"jobName"`
}

// CreateUserJobReply only for api docs
type CreateUserJobReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// UpdateUserJobByIDReply only for api docs
type UpdateUserJobByIDReply struct {
	Result
}

// GetUserJobByIDReply only for api docs
type GetUserJobByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserJob UserJobObjDetail `json:"userJob"`
	} `json:"data"` // return data
}

// DeleteUserJobByIDReply only for api docs
type DeleteUserJobByIDReply struct {
	Result
}

// DeleteUserJobsByIDsReply only for api docs
type DeleteUserJobsByIDsReply struct {
	Result
}

// ListUserJobsRequest request params
type ListUserJobsRequest struct {
	query.Params
}

// ListUserJobsReply only for api docs
type ListUserJobsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserJobs []UserJobObjDetail `json:"userJobs"`
	} `json:"data"` // return data
}

// DeleteUserJobsByIDsRequest request params
type DeleteUserJobsByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// GetUserJobByConditionRequest request params
type GetUserJobByConditionRequest struct {
	query.Conditions
}

// GetUserJobByConditionReply only for api docs
type GetUserJobByConditionReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserJob UserJobObjDetail `json:"userJob"`
	} `json:"data"` // return data
}

// ListUserJobsByIDsRequest request params
type ListUserJobsByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// ListUserJobsByIDsReply only for api docs
type ListUserJobsByIDsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserJobs []UserJobObjDetail `json:"userJobs"`
	} `json:"data"` // return data
}
