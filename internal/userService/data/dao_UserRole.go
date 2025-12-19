package data

import (
	"context"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
)

func (*userRoleDAO) GetByIDsAndProjectID(ctx context.Context, roleIDs []int32, projectID int32) ([]*model.UserRole, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := db.GetQuery()
	return q.UserRole.WithContext(ctx).Where(q.UserRole.ID.In(roleIDs...), q.UserRole.ProjID.Eq(projectID)).Find()
}
