package main

import (
	"context"
	"errors"
	"gogogo/pkg/db"
	"gogogo/pkg/db/dao/model"
)

type UserJobDO = model.UserJob

type userJobDAO struct{}

var UserJobDAO userJobDAO

func (*userJobDAO) Add(ctx context.Context, userJob *UserJobDO) error {
	if userJob == nil {
		return errors.New("param is invalid")
	}
	q := db.GetPG().UserJob
	err := q.WithContext(ctx).Create(userJob)
	if err != nil {
		return err
	}
	return err
}

func (*userJobDAO) GetAll(ctx context.Context,
) (userJobList []*UserJobDO, err error) {
	q := db.GetPG().UserJob
	userJobList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	return userJobList, nil
}

func (*userJobDAO) DelById(ctx context.Context, id int32) error {
	if id == 0 {
		return errors.New("param is invalid")
	}
	q := db.GetPG().UserJob
	_, err := q.WithContext(ctx).Where(q.ID.Eq(id)).Delete()
	if err != nil {
		return err
	}
	return err
}

func (*userJobDAO) GetById(ctx context.Context, id int32,
) (userJob *UserJobDO, err error) {
	if id == 0 {
		return userJob, errors.New("param is invalid")
	}

	q := db.GetPG().UserJob
	userJob, err = q.WithContext(ctx).
		Where(q.ID.Eq(id)).First()
	if err != nil {
		return nil, err
	}
	return userJob, nil
}
