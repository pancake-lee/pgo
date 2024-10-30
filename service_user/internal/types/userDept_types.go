package types

import (
	"time"

	"github.com/zhufuyi/sponge/pkg/ggorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.


// CreateUserDeptRequest request params
type CreateUserDeptRequest struct {
	DeptPath  string `json:"deptPath" binding:""`
	DeptName  string `json:"deptName" binding:""`
}

// UpdateUserDeptByIDRequest request params
type UpdateUserDeptByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id

	DeptPath  string `json:"deptPath" binding:""`
	DeptName  string `json:"deptName" binding:""`
}

// UserDeptObjDetail detail
type UserDeptObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id

	DeptPath  string `json:"deptPath"`
	DeptName  string `json:"deptName"`
}


// CreateUserDeptReply only for api docs
type CreateUserDeptReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// UpdateUserDeptByIDReply only for api docs
type UpdateUserDeptByIDReply struct {
	Result
}

// GetUserDeptByIDReply only for api docs
type GetUserDeptByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDept UserDeptObjDetail `json:"userDept"`
	} `json:"data"` // return data
}

// DeleteUserDeptByIDReply only for api docs
type DeleteUserDeptByIDReply struct {
	Result
}

// DeleteUserDeptsByIDsReply only for api docs
type DeleteUserDeptsByIDsReply struct {
	Result
}

// ListUserDeptsRequest request params
type ListUserDeptsRequest struct {
	query.Params
}

// ListUserDeptsReply only for api docs
type ListUserDeptsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDepts []UserDeptObjDetail `json:"userDepts"`
	} `json:"data"` // return data
}

// DeleteUserDeptsByIDsRequest request params
type DeleteUserDeptsByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// GetUserDeptByConditionRequest request params
type GetUserDeptByConditionRequest struct {
	query.Conditions
}

// GetUserDeptByConditionReply only for api docs
type GetUserDeptByConditionReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDept UserDeptObjDetail `json:"userDept"`
	} `json:"data"` // return data
}

// ListUserDeptsByIDsRequest request params
type ListUserDeptsByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// ListUserDeptsByIDsReply only for api docs
type ListUserDeptsByIDsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDepts []UserDeptObjDetail `json:"userDepts"`
	} `json:"data"` // return data
}
