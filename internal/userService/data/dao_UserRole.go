package data

import (
	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*userRoleDAO) GetByIDsAndProjectID(ctx *papp.AppCtx, roleIDs []int32, projectID int32) ([]*model.UserRole, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := db.GetQuery()
	return q.UserRole.WithContext(ctx).Where(q.UserRole.ID.In(roleIDs...), q.UserRole.ProjID.Eq(projectID)).Find()
}
