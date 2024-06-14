package main

import (
	"context"
	"errors"
	"gogogo/pkg/db"
	"gogogo/pkg/db/dao/model"
)

type UserDO = model.User

type userDAO struct{}

var UserDAO userDAO

func (*userDAO) Add(ctx context.Context, user *UserDO) error {
	if user == nil {
		return errors.New("param is invalid")
	}
	q := db.GetPG().User
	err := q.WithContext(ctx).Create(user)
	if err != nil {
		return err
	}
	return err
}

func (*userDAO) GetAll(ctx context.Context,
) (userList []*UserDO, err error) {
	q := db.GetPG().User
	userList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	return userList, nil
}

func (*userDAO) DelById(ctx context.Context, id int32) error {
	if id == 0 {
		return errors.New("param is invalid")
	}
	q := db.GetPG().User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return err
}

func (*userDAO) GetById(ctx context.Context, id int32,
) (user *UserDO, err error) {
	if id == 0 {
		return user, errors.New("param is invalid")
	}

	q := db.GetPG().User
	user, err = q.WithContext(ctx).
		Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return user, nil
}
