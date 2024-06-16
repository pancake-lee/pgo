package data

import (
	"context"
	"errors"
	"gogogo/pkg/db"
)

// MARK 1 标记删除此标记以上的内容，再拼接到dao_abandonCodeIdx.go最后

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

func (*abandonCodeDAO) DelByIds(ctx context.Context, idx1s []int32) error {
	if len(idx1s) == 0 {
		return nil
	}
	q := db.GetPG().AbandonCode
	_, err := q.WithContext(ctx).
		Where(q.Idx1.In(idx1s...)).Delete()
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

func (*abandonCodeDAO) GetByIds(ctx context.Context, idx1s []int32,
) (abandonCodeMap map[int32]*AbandonCodeDO, err error) {
	if len(idx1s) == 0 {
		return nil, nil
	}

	q := db.GetPG().AbandonCode
	l, err := q.WithContext(ctx).
		Where(q.Idx1.In(idx1s...)).Find()
	if err != nil {
		return nil, err
	}
	abandonCodeMap = make(map[int32]*AbandonCodeDO)
	for _, i := range l {
		abandonCodeMap[i.Idx1] = i
	}
	return abandonCodeMap, nil
}
