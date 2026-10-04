package data

import (
	"github.com/pancake-lee/pgo/internal/pkg/db"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/papp"
)

// GetByRoleIDs 批量读取角色权限，按主键确定重复动作的覆盖顺序。
func (*userRolePermissionAssocDAO) GetByRoleIDs(ctx *papp.AppCtx, roleIDs []int32) ([]*model.UserRolePermissionAssoc, error) {
	if len(roleIDs) == 0 {
		return nil, nil
	}

	q := db.GetQuery()
	p := q.UserRolePermissionAssoc
	return p.WithContext(ctx).Select(p.Action, p.PathPattern).
		Where(p.RoleID.In(roleIDs...)).Order(p.ID).Find()
}

// GetByUserAndProject 用一条 JOIN 读取指定用户在项目中的权限。
func (*userRolePermissionAssocDAO) GetByUserAndProject(
	ctx *papp.AppCtx, userID, projectID int32,
) ([]*model.UserRolePermissionAssoc, error) {
	q := db.GetQuery()
	p := q.UserRolePermissionAssoc
	r := q.UserRole
	a := q.UserRoleAssoc
	return p.WithContext(ctx).
		Select(p.Action, p.PathPattern).
		Join(r, r.ID.EqCol(p.RoleID)).
		Join(a, a.RoleID.EqCol(r.ID)).
		Where(a.UserID.Eq(userID), r.ProjID.Eq(projectID)).
		Order(p.ID).Find()
}
