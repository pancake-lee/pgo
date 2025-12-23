package data

import (
	"context"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/perr"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func (*projectDAO) GetByMtblRecordID(ctx context.Context, recordId string) (*ProjectDO, error) {
	if recordId == "" {
		return nil, plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetQuery().Project
	ret, err := q.WithContext(ctx).
		Where(q.MtblRecordID.Eq(recordId)).
		First()
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (*projectDAO) DeleteByMtblRecordID(ctx context.Context, recordId string) error {
	if recordId == "" {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetQuery().Project
	_, err := q.WithContext(ctx).
		Where(q.MtblRecordID.Eq(recordId)).
		Delete()
	return err
}
