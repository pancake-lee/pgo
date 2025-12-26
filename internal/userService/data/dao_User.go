package data

import (
	"errors"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/internal/pkg/perr"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func (*userDAO) GetOrAdd(ctx *papp.AppCtx,
	user *UserDO) (*model.User, error) {
	if user == nil || user.UserName == "" {
		return nil, ctx.Log.LogErrMsg("user name is empty")
	}
	q := db.GetQuery().User
	user, err := q.WithContext(ctx).
		Where(q.UserName.Eq(user.UserName)).
		Attrs(q.UserName.Value(user.UserName)).
		FirstOrCreate()
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}
	return user, err
}

func (*userDAO) EditUserName(ctx *papp.AppCtx, id int32, userName string) error {
	if id == 0 || userName == "" {
		return errors.New("argument invalid")
	}
	q := db.GetQuery().User
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(id)).
		UpdateSimple(q.UserName.Value(userName))
	return err
}

func (*userDAO) SelectByRecordId(ctx *papp.AppCtx, recordID string) (*model.User, error) {
	if recordID == "" {
		return nil, nil
	}
	q := db.GetQuery().User
	user, err := q.WithContext(ctx).
		Where(q.MtblRecordID.Eq(recordID)).
		First()
	if err != nil {
		return nil, err
	}
	return user, nil
}
func (*userDAO) UpdateMtblInfo(ctx *papp.AppCtx, id int32,
	mtblRecordId, lastEditFrom string) error {
	if id == 0 {
		return errors.New("argument invalid")
	}
	q := db.GetQuery().User
	_, err := q.WithContext(ctx).
		Where(q.ID.Eq(id)).
		UpdateSimple(
			q.MtblRecordID.Value(mtblRecordId),
			q.LastEditFrom.Value(lastEditFrom),
		)
	return err
}

func (*userDAO) DelByRecordID(ctx *papp.AppCtx, recordID string) error {
	if recordID == "" {
		return plogger.LogErr(perr.ErrParamInvalid)
	}
	q := db.GetQuery().User
	_, err := q.WithContext(ctx).Where(q.MtblRecordID.Eq(recordID)).Delete()
	if err != nil {
		return plogger.LogErr(err)
	}
	return nil
}
