package service

import (
	"context"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func OnMtblUpdateUser(ctx context.Context) error {
	return NewMtblUser(ctx).HandleMtblEvent()
}

func NewMtblUser(ctx context.Context) *papitable.BaseDataProvider {
	ret := papitable.BaseDataProvider{
		Ctx:         ctx,
		DatasheetID: conf.UserSvcConf.APITable.UserSheetID,
		TableConfig: &papitable.TableConfig{
			TableName:  "人员表",
			PrimaryCol: papitable.NewTextCol("姓名"),
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
	ret.WithLogger(plogger.GetDefaultLogWarper())
	return &ret
}

// --------------------------------------------------
type UserDAOWrapper struct{}

func (d *UserDAOWrapper) Add(ctx context.Context, do any) error {
	return data.UserDAO.Add(ctx, do.(*data.UserDO))
}

func (d *UserDAOWrapper) UpdateByID(ctx context.Context, do any) error {
	return data.UserDAO.UpdateByID(ctx, do.(*data.UserDO))
}

func (d *UserDAOWrapper) GetAll(ctx context.Context) ([]any, error) {
	list, err := data.UserDAO.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]any, len(list))
	for i, v := range list {
		ret[i] = v
	}
	return ret, nil
}

func (d *UserDAOWrapper) GetByID(ctx context.Context, id int32) (any, error) {
	return data.UserDAO.GetByID(ctx, id)
}

func (d *UserDAOWrapper) GetByMtblRecordID(ctx context.Context, recordId string) (any, error) {
	return data.UserDAO.SelectByRecordId(ctx, recordId)
}

func (d *UserDAOWrapper) DeleteByMtblRecordID(ctx context.Context, recordId string) error {
	return data.UserDAO.DelByRecordID(ctx, recordId)
}
