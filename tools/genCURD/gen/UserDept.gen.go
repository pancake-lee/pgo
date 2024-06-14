package main

import (
	"context"
	"errors"
	"gogogo/pkg/db"
	"gogogo/pkg/db/dao/model"
)

type UserDeptDO = model.UserDept

type userDeptDAO struct{}

var UserDeptDAO userDeptDAO

func (*userDeptDAO) Add(ctx context.Context, userDept *UserDeptDO) error {
	if userDept == nil {
		return errors.New("param is invalid")
	}
	q := db.GetPG().UserDept
	err := q.WithContext(ctx).Create(userDept)
	if err != nil {
		return err
	}
	return err
}

func (*userDeptDAO) GetAll(ctx context.Context,
) (userDeptList []*UserDeptDO, err error) {
	q := db.GetPG().UserDept
	userDeptList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	return userDeptList, nil
}

func (*userDeptDAO) DelById(ctx context.Context, id int32) error {
	if id == 0 {
		return errors.New("param is invalid")
	}
	q := db.GetPG().UserDept
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return err
}

func (*userDeptDAO) GetById(ctx context.Context, id int32,
) (userDept *UserDeptDO, err error) {
	if id == 0 {
		return userDept, errors.New("param is invalid")
	}

	q := db.GetPG().UserDept
	userDept, err = q.WithContext(ctx).
		Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return userDept, nil
}
