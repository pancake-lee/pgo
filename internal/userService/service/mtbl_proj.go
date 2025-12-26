package service

import (
	"context"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func OnMtblUpdateProject(_ctx context.Context) error {
	return NewMtblProject(_ctx).HandleMtblEvent()
}

func NewMtblProject(_ctx context.Context) *papitable.BaseDataProvider {
	ret := papitable.BaseDataProvider{
		Ctx:         _ctx,
		DatasheetID: conf.UserSvcConf.APITable.ProjectSheetID,
		TableConfig: &papitable.TableConfig{
			TableName:  "项目表",
			PrimaryCol: papitable.NewTextCol("项目名"),
			ColList: []*papitable.FieldConfig{
				{Col: papitable.NewTextCol("项目名"), DOField: "ProjName"},
				{Col: papitable.NewSimpleNumCol("ProjectID"), DOField: "ID"},
			},
			NewDO: func() any { return &data.ProjectDO{} },
		},
		DAO: &ProjectDAOWrapper{},
		GetIDByDO: func(record any) int32 {
			if p, ok := record.(*data.ProjectDO); ok {
				return p.ID
			}
			return 0
		},
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

func (w *ProjectDAOWrapper) GetByMtblRecordID(_ctx context.Context, recordId string) (any, error) {
	ctx := papp.NewAppCtx(_ctx)
	return data.ProjectDAO.GetByMtblRecordID(ctx, recordId)
}

func (w *ProjectDAOWrapper) DeleteByMtblRecordID(_ctx context.Context, recordId string) error {
	ctx := papp.NewAppCtx(_ctx)
	return data.ProjectDAO.DeleteByMtblRecordID(ctx, recordId)
}
