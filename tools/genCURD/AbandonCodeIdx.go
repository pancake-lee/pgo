package main

import (
	"context"
	"errors"
	"gogogo/pkg/db"
)

// ignore above this line

func (*abandonCodeDAO) DelById(ctx context.Context, idx1 int32) error {
	if idx1 == 0 {
		return errors.New("param is invalid")
	}
	q := db.GetPG().AbandonCode
	_, err := q.WithContext(ctx).Where(q.Idx1.Eq(idx1)).Delete()
	if err != nil {
		return err
	}
	return err
}

func (*abandonCodeDAO) GetById(ctx context.Context, idx1 int32,
) (abandonCode *AbandonCodeDO, err error) {
	if idx1 == 0 {
		return abandonCode, errors.New("param is invalid")
	}

	q := db.GetPG().AbandonCode
	abandonCode, err = q.WithContext(ctx).
		Where(q.Idx1.Eq(idx1)).First()
	if err != nil {
		return nil, err
	}
	return abandonCode, nil
}
