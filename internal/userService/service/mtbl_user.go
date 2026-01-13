package service

import (
	"context"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func OnMtblUpdateUser(_ctx context.Context) error {
	return NewMtblUser(_ctx).HandleMtblEvent()
}

func NewMtblUser(_ctx context.Context) *papitable.BaseDataProvider {
	ret := papitable.BaseDataProvider{
		Ctx:         _ctx,
		DatasheetID: conf.UserSvcConf.APITable.UserSheetID,
		TableConfig: &papitable.TableConfig{
			TableName: "人员表",
			FirstCol:  papitable.NewTextCol("姓名"),
			PrimaryCol: &papitable.FieldConfig{
				Col: papitable.NewSimpleNumCol("UserID"), DOField: "ID"},
			ColList: []*papitable.FieldConfig{
				{Col: papitable.NewTextCol("姓名"), DOField: "UserName"},
				{Col: papitable.NewSimpleNumCol("UserID"), DOField: "ID"},
			},
			NewDO: func() any { return &data.UserDO{} },
		},
		DAO: &UserDAOWrapper{},
		GetIDByDO: func(record any) int32 {
			if do, ok := record.(*data.UserDO); ok {
				return do.ID
			}
			return 0
		},
	}
	ctx := papp.NewAppCtx(_ctx)
	ret.WithLogger(ctx.Log)
	return &ret
}

// --------------------------------------------------
type UserDAOWrapper struct{}

func (d *UserDAOWrapper) Add(_ctx context.Context, do any) error {
	appCtx := papp.NewAppCtx(_ctx)
	return data.UserDAO.Add(appCtx, do.(*data.UserDO))
}

func (d *UserDAOWrapper) UpdateByID(_ctx context.Context, do any) error {
	appCtx := papp.NewAppCtx(_ctx)
	return data.UserDAO.UpdateByID(appCtx, do.(*data.UserDO))
}

func (d *UserDAOWrapper) GetAll(_ctx context.Context) ([]any, error) {
	appCtx := papp.NewAppCtx(_ctx)
	list, err := data.UserDAO.GetAll(appCtx)
	if err != nil {
		return nil, err
	}
	ret := make([]any, len(list))
	for i, v := range list {
		ret[i] = v
	}
	return ret, nil
}

func (d *UserDAOWrapper) GetByID(_ctx context.Context, id int32) (any, error) {
	appCtx := papp.NewAppCtx(_ctx)
	return data.UserDAO.GetByID(appCtx, id)
}

func (d *UserDAOWrapper) DeleteByMtblRecordID(_ctx context.Context, recordId string) error {
	appCtx := papp.NewAppCtx(_ctx)
	return data.UserDAO.DelByRecordID(appCtx, recordId)
}

func (d *UserDAOWrapper) UpdateMtblInfo(_ctx context.Context, localId string,
	recordId, lastEditFrom string) error {
	appCtx := papp.NewAppCtx(_ctx)
	id, err := putil.StrToInt32(localId)
	if err != nil {
		return appCtx.Log.LogErr(err)
	}

	return data.UserDAO.UpdateMtblInfo(appCtx, id, recordId, lastEditFrom)
}
