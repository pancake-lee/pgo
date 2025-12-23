package service

import (
	"context"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papitable"
)

func OnMtblUpdateProject(ctx context.Context) error {
	return NewMtblProject(ctx).HandleMtblEvent()
}

func NewMtblProject(ctx context.Context) *papitable.BaseDataProvider {
	return &papitable.BaseDataProvider{
		Ctx:         ctx,
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
}

// --------------------------------------------------
// ProjectDAOWrapper 意义在于转换 MtblDAO 接口定义的any类型
type ProjectDAOWrapper struct{}

func (w *ProjectDAOWrapper) Add(ctx context.Context, do any) error {
	return data.ProjectDAO.Add(ctx, do.(*data.ProjectDO))
}

func (w *ProjectDAOWrapper) UpdateByID(ctx context.Context, do any) error {
	return data.ProjectDAO.UpdateByID(ctx, do.(*data.ProjectDO))
}

func (w *ProjectDAOWrapper) GetAll(ctx context.Context) ([]any, error) {
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

func (w *ProjectDAOWrapper) GetByID(ctx context.Context, id int32) (any, error) {
	return data.ProjectDAO.GetByID(ctx, id)
}

func (w *ProjectDAOWrapper) GetByMtblRecordID(ctx context.Context, recordId string) (any, error) {
	return data.ProjectDAO.GetByMtblRecordID(ctx, recordId)
}

func (w *ProjectDAOWrapper) DeleteByMtblRecordID(ctx context.Context, recordId string) error {
	return data.ProjectDAO.DeleteByMtblRecordID(ctx, recordId)
}
