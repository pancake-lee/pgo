package service

import (
	"context"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func OnMtblUpdateProject(_ctx context.Context) error {
	return NewMtblProject(_ctx).HandleMtblEvent()
}

func NewMtblProject(_ctx context.Context) *papitable.BaseDataProvider {
	ret := papitable.BaseDataProvider{
		Ctx:         _ctx,
		DatasheetID: conf.UserSvcConf.APITable.ProjectSheetID,
		TableConfig: &papitable.TableConfig{
			TableName: "项目表",
			FirstCol:  papitable.NewTextCol("项目名"),
			PrimaryCol: &papitable.FieldConfig{
				Col: papitable.NewSimpleNumCol("ProjectID"), DOField: "ID"},
			ColList: []*papitable.FieldConfig{
				{Col: papitable.NewTextCol("项目名"), DOField: "ProjName"},
				{Col: papitable.NewSimpleNumCol("ProjectID"), DOField: "ID"},
			},
			NewDO: func() any { return &data.ProjectDO{} },
		},
		DAO: &ProjectDAOWrapper{},
	}
	ctx := papp.NewAppCtx(_ctx)
	ret.WithLogger(ctx.Log)
	return &ret
}

// --------------------------------------------------
// ProjectDAOWrapper 意义在于转换 MtblDAO 接口定义的any类型
type ProjectDAOWrapper struct{}

func (w *ProjectDAOWrapper) Add(_ctx context.Context, do any) error {
	ctx := papp.NewAppCtx(_ctx)
	return data.ProjectDAO.Add(ctx, do.(*data.ProjectDO))
}

func (w *ProjectDAOWrapper) UpdateByID(_ctx context.Context, do any) error {
	ctx := papp.NewAppCtx(_ctx)
	return data.ProjectDAO.UpdateByID(ctx, do.(*data.ProjectDO))
}

func (w *ProjectDAOWrapper) GetAll(_ctx context.Context) ([]any, error) {
	ctx := papp.NewAppCtx(_ctx)
	list, err := data.ProjectDAO.GetAll(ctx)
	if err != nil {
		return nil, err
	}

	ret := make([]any, len(list))
	for i, v := range list {
		ret[i] = v
	}
	return ret, nil
}

func (w *ProjectDAOWrapper) GetByID(_ctx context.Context, id int32) (any, error) {
	ctx := papp.NewAppCtx(_ctx)
	return data.ProjectDAO.GetByID(ctx, id)
}

func (w *ProjectDAOWrapper) DeleteByID(_ctx context.Context, id int32) error {
	ctx := papp.NewAppCtx(_ctx)
	return data.ProjectDAO.DelByID(ctx, id)
}

func (d *ProjectDAOWrapper) UpdateMtblInfo(_ctx context.Context,
	localId string, lastEditFrom string) error {
	appCtx := papp.NewAppCtx(_ctx)
	id, err := putil.StrToInt32(localId)
	if err != nil {
		return appCtx.Log.LogErr(err)
	}

	return data.ProjectDAO.UpdateMtblInfo(appCtx, id, lastEditFrom)
}
