package ecode

import (
	"github.com/zhufuyi/sponge/pkg/errcode"
)

// userDept business-level http error codes.
// the userDeptNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	userDeptNO       = 78
	userDeptName     = "userDept"
	userDeptBaseCode = errcode.HCode(userDeptNO)

	ErrCreateUserDept     = errcode.NewError(userDeptBaseCode+1, "failed to create "+userDeptName)
	ErrDeleteByIDUserDept = errcode.NewError(userDeptBaseCode+2, "failed to delete "+userDeptName)
	ErrUpdateByIDUserDept = errcode.NewError(userDeptBaseCode+3, "failed to update "+userDeptName)
	ErrGetByIDUserDept    = errcode.NewError(userDeptBaseCode+4, "failed to get "+userDeptName+" details")
	ErrListUserDept       = errcode.NewError(userDeptBaseCode+5, "failed to list of "+userDeptName)

	ErrDeleteByIDsUserDept    = errcode.NewError(userDeptBaseCode+6, "failed to delete by batch ids "+userDeptName)
	ErrGetByConditionUserDept = errcode.NewError(userDeptBaseCode+7, "failed to get "+userDeptName+" details by conditions")
	ErrListByIDsUserDept      = errcode.NewError(userDeptBaseCode+8, "failed to list by batch ids "+userDeptName)
	ErrListByLastIDUserDept   = errcode.NewError(userDeptBaseCode+9, "failed to list by last id "+userDeptName)

	// error codes are globally unique, adding 1 to the previous error code
)
