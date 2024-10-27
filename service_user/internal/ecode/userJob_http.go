package ecode

import (
	"github.com/zhufuyi/sponge/pkg/errcode"
)

// userJob business-level http error codes.
// the userJobNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	userJobNO       = 2
	userJobName     = "userJob"
	userJobBaseCode = errcode.HCode(userJobNO)

	ErrCreateUserJob     = errcode.NewError(userJobBaseCode+1, "failed to create "+userJobName)
	ErrDeleteByIDUserJob = errcode.NewError(userJobBaseCode+2, "failed to delete "+userJobName)
	ErrUpdateByIDUserJob = errcode.NewError(userJobBaseCode+3, "failed to update "+userJobName)
	ErrGetByIDUserJob    = errcode.NewError(userJobBaseCode+4, "failed to get "+userJobName+" details")
	ErrListUserJob       = errcode.NewError(userJobBaseCode+5, "failed to list of "+userJobName)

	ErrDeleteByIDsUserJob    = errcode.NewError(userJobBaseCode+6, "failed to delete by batch ids "+userJobName)
	ErrGetByConditionUserJob = errcode.NewError(userJobBaseCode+7, "failed to get "+userJobName+" details by conditions")
	ErrListByIDsUserJob      = errcode.NewError(userJobBaseCode+8, "failed to list by batch ids "+userJobName)
	ErrListByLastIDUserJob   = errcode.NewError(userJobBaseCode+9, "failed to list by last id "+userJobName)

	// error codes are globally unique, adding 1 to the previous error code
)
