package main

import (
	"context"
	"errors"
	"gogogo/pkg/db"
	"gogogo/pkg/db/dao/model"
)

type AbandonCodeDO = model.AbandonCode

type abandonCodeDAO struct{}

var AbandonCodeDAO abandonCodeDAO

func (*abandonCodeDAO) Add(ctx context.Context, abandonCode *AbandonCodeDO) error {
	if abandonCode == nil {
		return errors.New("param is invalid")
	}
	q := db.GetPG().AbandonCode
	err := q.WithContext(ctx).Create(abandonCode)
	if err != nil {
		return err
	}
	return err
}

func (*abandonCodeDAO) GetAll(ctx context.Context,
) (abandonCodeList []*AbandonCodeDO, err error) {
	q := db.GetPG().AbandonCode
	abandonCodeList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, err
	}
	return abandonCodeList, nil
}
