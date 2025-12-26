package data

import (
	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (*userRolePermissionAssocDAO) GetByRoleIDs(ctx *papp.AppCtx, roleIDs []int32) ([]*model.UserRolePermissionAssoc, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := db.GetQuery()
	return q.UserRolePermissionAssoc.WithContext(ctx).Where(q.UserRolePermissionAssoc.RoleID.In(roleIDs...)).Find()
}
