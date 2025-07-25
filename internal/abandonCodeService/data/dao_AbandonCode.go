package data

import (
	"context"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/internal/pkg/perr"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

type AbandonCodeDO = model.AbandonCode

type abandonCodeDAO struct{}

var AbandonCodeDAO abandonCodeDAO

func (*abandonCodeDAO) Add(ctx context.Context, abandonCode *AbandonCodeDO) error {
	if abandonCode == nil {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetPG().AbandonCode
	err := q.WithContext(ctx).Create(abandonCode)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*abandonCodeDAO) GetAll(ctx context.Context,
) (abandonCodeList []*AbandonCodeDO, err error) {
	q := db.GetPG().AbandonCode
	abandonCodeList, err = q.WithContext(ctx).Find()
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	return abandonCodeList, nil
}

// MARK REMOVE IF NO PRIMARY KEY START
func (*abandonCodeDAO) UpdateByIdx1(ctx context.Context, do *AbandonCodeDO) error {
	if do.Idx1 == 0 {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetPG().AbandonCode
	_, err := q.WithContext(ctx).Where(q.Idx1.Eq(do.Idx1)).Updates(do)
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*abandonCodeDAO) DelByIdx1(ctx context.Context, idx1 int32) error {
	if idx1 == 0 {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetPG().AbandonCode
	_, err := q.WithContext(ctx).Where(q.Idx1.Eq(idx1)).Delete()
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*abandonCodeDAO) DelByIdx1List(ctx context.Context, idx1List []int32) error {
	if len(idx1List) == 0 {
		return nil
	}
	q := db.GetPG().AbandonCode
	_, err := q.WithContext(ctx).
		Where(q.Idx1.In(idx1List...)).Delete()
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}

func (*abandonCodeDAO) GetByIdx1(ctx context.Context, idx1 int32,
) (abandonCode *AbandonCodeDO, err error) {
	if idx1 == 0 {
		return abandonCode, plogger.LogErr(perr.ErrParamInvalid)
	}

	q := db.GetPG().AbandonCode
	abandonCode, err = q.WithContext(ctx).
		Where(q.Idx1.Eq(idx1)).First()
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	return abandonCode, nil
}

func (*abandonCodeDAO) GetByIdx1List(ctx context.Context, idx1List []int32,
) (abandonCodeMap map[int32]*AbandonCodeDO, err error) {
	if len(idx1List) == 0 {
		return nil, nil
	}

	q := db.GetPG().AbandonCode
	l, err := q.WithContext(ctx).
		Where(q.Idx1.In(idx1List...)).Find()
	if err != nil {
		return nil, plogger.LogErr(err)
	}
	abandonCodeMap = make(map[int32]*AbandonCodeDO)
	for _, i := range l {
		abandonCodeMap[i.Idx1] = i
	}
	return abandonCodeMap, nil
}

// MARK REMOVE IF NO PRIMARY KEY END
