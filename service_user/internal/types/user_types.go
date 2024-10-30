package types

import (
	"time"

	"github.com/zhufuyi/sponge/pkg/ggorm/query"
)

var _ time.Time

// Tip: suggested filling in the binding rules https://github.com/go-playground/validator in request struct fields tag.


// CreateUserRequest request params
type CreateUserRequest struct {
	UserName  string `json:"userName" binding:""` // The name of the user
}

// UpdateUserByIDRequest request params
type UpdateUserByIDRequest struct {
	ID uint64 `json:"id" binding:""` // uint64 id

	UserName  string `json:"userName" binding:""` // The name of the user
}

// UserObjDetail detail
type UserObjDetail struct {
	ID uint64 `json:"id"` // convert to uint64 id

	UserName  string `json:"userName"` // The name of the user
}


// CreateUserReply only for api docs
type CreateUserReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		ID uint64 `json:"id"` // id
	} `json:"data"` // return data
}

// UpdateUserByIDReply only for api docs
type UpdateUserByIDReply struct {
	Result
}

// GetUserByIDReply only for api docs
type GetUserByIDReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		User UserObjDetail `json:"user"`
	} `json:"data"` // return data
}

// DeleteUserByIDReply only for api docs
type DeleteUserByIDReply struct {
	Result
}

// DeleteUsersByIDsReply only for api docs
type DeleteUsersByIDsReply struct {
	Result
}

// ListUsersRequest request params
type ListUsersRequest struct {
	query.Params
}

// ListUsersReply only for api docs
type ListUsersReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Users []UserObjDetail `json:"users"`
	} `json:"data"` // return data
}

// DeleteUsersByIDsRequest request params
type DeleteUsersByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// GetUserByConditionRequest request params
type GetUserByConditionRequest struct {
	query.Conditions
}

// GetUserByConditionReply only for api docs
type GetUserByConditionReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		User UserObjDetail `json:"user"`
	} `json:"data"` // return data
}

// ListUsersByIDsRequest request params
type ListUsersByIDsRequest struct {
	IDs []uint64 `json:"ids" binding:"min=1"` // id list
}

// ListUsersByIDsReply only for api docs
type ListUsersByIDsReply struct {
	Code int    `json:"code"` // return code
	Msg  string `json:"msg"`  // return information description
	Data struct {
		Users []UserObjDetail `json:"users"`
	} `json:"data"` // return data
}
