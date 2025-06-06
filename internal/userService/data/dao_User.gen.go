// Code generated by tools/genCURD. DO NOT EDIT.

package data

import (
	"context"
	"errors"
	"pgo/internal/pkg/db"
	"pgo/internal/pkg/db/model"
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

func (*userDAO) UpdateByID(ctx context.Context, do *UserDO) error {
	if do.ID == 0 {
		return errors.New("param is invalid")
	}
	q := db.GetPG().User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(do.ID)).Updates(do)
	if err != nil {
		return err
	}
	return err
}

func (*userDAO) DelByID(ctx context.Context, iD int32) error {
	if iD == 0 {
		return errors.New("param is invalid")
	}
	q := db.GetPG().User
	_, err := q.WithContext(ctx).Where(q.ID.Eq(iD)).Delete()
	if err != nil {
		return err
	}
	return err
}

func (*userDAO) DelByIDList(ctx context.Context, iDList []int32) error {
	if len(iDList) == 0 {
		return nil
	}
	q := db.GetPG().User
	_, err := q.WithContext(ctx).
		Where(q.ID.In(iDList...)).Delete()
	if err != nil {
		return err
	}
	return err
}

func (*userDAO) GetByID(ctx context.Context, iD int32,
) (user *UserDO, err error) {
	if iD == 0 {
		return user, errors.New("param is invalid")
	}

	q := db.GetPG().User
	user, err = q.WithContext(ctx).
		Where(q.ID.Eq(iD)).First()
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (*userDAO) GetByIDList(ctx context.Context, iDList []int32,
) (userMap map[int32]*UserDO, err error) {
	if len(iDList) == 0 {
		return nil, nil
	}

	q := db.GetPG().User
	l, err := q.WithContext(ctx).
		Where(q.ID.In(iDList...)).Find()
	if err != nil {
		return nil, err
	}
	userMap = make(map[int32]*UserDO)
	for _, i := range l {
		userMap[i.ID] = i
	}
	return userMap, nil
}

