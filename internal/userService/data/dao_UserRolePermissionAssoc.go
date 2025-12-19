package data

import (
	"context"

	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
)

func (*userRolePermissionAssocDAO) GetByRoleIDs(ctx context.Context, roleIDs []int32) ([]*model.UserRolePermissionAssoc, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := db.GetQuery()
	return q.UserRolePermissionAssoc.WithContext(ctx).Where(q.UserRolePermissionAssoc.RoleID.In(roleIDs...)).Find()
}
