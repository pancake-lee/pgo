package ecode

import (
	"github.com/zhufuyi/sponge/pkg/errcode"
)

// userDeptAssoc business-level http error codes.
// the userDeptAssocNO value range is 1~100, if the same error code is used, it will cause panic.
var (
	userDeptAssocNO       = 78
	userDeptAssocName     = "userDeptAssoc"
	userDeptAssocBaseCode = errcode.HCode(userDeptAssocNO)

	ErrCreateUserDeptAssoc     = errcode.NewError(userDeptAssocBaseCode+1, "failed to create "+userDeptAssocName)
	ErrDeleteByIDUserDeptAssoc = errcode.NewError(userDeptAssocBaseCode+2, "failed to delete "+userDeptAssocName)
	ErrUpdateByIDUserDeptAssoc = errcode.NewError(userDeptAssocBaseCode+3, "failed to update "+userDeptAssocName)
	ErrGetByIDUserDeptAssoc    = errcode.NewError(userDeptAssocBaseCode+4, "failed to get "+userDeptAssocName+" details")
	ErrListUserDeptAssoc       = errcode.NewError(userDeptAssocBaseCode+5, "failed to list of "+userDeptAssocName)

	ErrDeleteByIDsUserDeptAssoc    = errcode.NewError(userDeptAssocBaseCode+6, "failed to delete by batch ids "+userDeptAssocName)
	ErrGetByConditionUserDeptAssoc = errcode.NewError(userDeptAssocBaseCode+7, "failed to get "+userDeptAssocName+" details by conditions")
	ErrListByIDsUserDeptAssoc      = errcode.NewError(userDeptAssocBaseCode+8, "failed to list by batch ids "+userDeptAssocName)
	ErrListByLastIDUserDeptAssoc   = errcode.NewError(userDeptAssocBaseCode+9, "failed to list by last id "+userDeptAssocName)

	// error codes are globally unique, adding 1 to the previous error code
)
