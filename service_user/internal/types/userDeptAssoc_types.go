package types

import (
	"time"

	"github.com/zhufuyi/sponge/pkg/ggorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.


// CreateUserDeptAssocRequest request params
type CreateUserDeptAssocRequest struct {
	UserID  int `json:"userID" binding:""`
	DeptID  int `json:"deptID" binding:""`
	JobID  int `json:"jobID" binding:""`
}

// UpdateUserDeptAssocByIDRequest request params
type UpdateUserDeptAssocByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id

	UserID  int `json:"userID" binding:""`
	DeptID  int `json:"deptID" binding:""`
	JobID  int `json:"jobID" binding:""`
}

// UserDeptAssocObjDetail detail
type UserDeptAssocObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id

	UserID  int `json:"userID"`
	DeptID  int `json:"deptID"`
	JobID  int `json:"jobID"`
}


// CreateUserDeptAssocReply only for api docs
type CreateUserDeptAssocReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// UpdateUserDeptAssocByIDReply only for api docs
type UpdateUserDeptAssocByIDReply struct {
	Result
}

// GetUserDeptAssocByIDReply only for api docs
type GetUserDeptAssocByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDeptAssoc UserDeptAssocObjDetail `json:"userDeptAssoc"`
	} `json:"data"` // return data
}

// DeleteUserDeptAssocByIDReply only for api docs
type DeleteUserDeptAssocByIDReply struct {
	Result
}

// DeleteUserDeptAssocsByIDsReply only for api docs
type DeleteUserDeptAssocsByIDsReply struct {
	Result
}

// ListUserDeptAssocsRequest request params
type ListUserDeptAssocsRequest struct {
	query.Params
}

// ListUserDeptAssocsReply only for api docs
type ListUserDeptAssocsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDeptAssocs []UserDeptAssocObjDetail `json:"userDeptAssocs"`
	} `json:"data"` // return data
}

// DeleteUserDeptAssocsByIDsRequest request params
type DeleteUserDeptAssocsByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// GetUserDeptAssocByConditionRequest request params
type GetUserDeptAssocByConditionRequest struct {
	query.Conditions
}

// GetUserDeptAssocByConditionReply only for api docs
type GetUserDeptAssocByConditionReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDeptAssoc UserDeptAssocObjDetail `json:"userDeptAssoc"`
	} `json:"data"` // return data
}

// ListUserDeptAssocsByIDsRequest request params
type ListUserDeptAssocsByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// ListUserDeptAssocsByIDsReply only for api docs
type ListUserDeptAssocsByIDsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		UserDeptAssocs []UserDeptAssocObjDetail `json:"userDeptAssocs"`
	} `json:"data"` // return data
}
