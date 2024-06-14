package main

import (
	"context"
	"errors"
	"gogogo/pkg/db"
	"gogogo/pkg/db/dao/model"
)

type UserDeptAssocDO = model.UserDeptAssoc

type userDeptAssocDAO struct{}

var UserDeptAssocDAO userDeptAssocDAO

func (*userDeptAssocDAO) Add(ctx context.Context, userDeptAssoc *UserDeptAssocDO) error {
	if userDeptAssoc == nil {
		return errors.New("param is invalid")
	}
	q := db.GetPG().UserDeptAssoc
	err := q.WithContext(ctx).Create(userDeptAssoc)
	if err != nil {
		return err
	}
	return err
}

func (*userDeptAssocDAO) GetAll(ctx context.Context,
) (userDeptAssocList []*UserDeptAssocDO, err error) {
	q := db.GetPG().UserDeptAssoc
	userDeptAssocList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	return userDeptAssocList, nil
}
