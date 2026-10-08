package service

import (
	"context"

	api "github.com/pancake-lee/pgo/internal/pkg/api"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papp"
)

// usePermissionJoin 选择权限查询实现，true 使用 JOIN，false 使用批量查询。
const usePermissionJoin = false

// GetUserPermissions 读取项目权限，支持 JOIN 与批量查询的同负载对照。
func (s *UserServer) GetUserPermissions(_ctx context.Context, req *api.GetUserPermissionsRequest) (*api.GetUserPermissionsResponse, error) {
	ctx := papp.NewAppCtx(_ctx)
	if usePermissionJoin {
		permissionList, err := data.UserRolePermissionAssocDAO.
			GetByUserAndProject(ctx, req.UserID, req.ProjectID)
		if err != nil {
			return nil, ctx.Log.LogErr(err)
		}
		permissionMap := make(map[string]string)
		for _, permission := range permissionList {
			permissionMap[permission.Action] = permission.PathPattern
		}
		return &api.GetUserPermissionsResponse{
			ActionToPathPattern: permissionMap,
		}, nil
	}
	// 1. Get all RoleIDs for the user
	userRoleAssocList, err := data.UserRoleAssocDAO.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}

	var candidateRoleIDList []int32
	for _, ura := range userRoleAssocList {
		candidateRoleIDList = append(candidateRoleIDList, ura.RoleID)
	}

	// 2. Filter by ProjectID
	roleList, err := data.UserRoleDAO.GetByIDsAndProjectID(ctx, candidateRoleIDList, req.ProjectID)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}

	var finalRoleIDList []int32
	for _, r := range roleList {
		finalRoleIDList = append(finalRoleIDList, r.ID)
	}

	// 3. Get Permissions
	permList, err := data.UserRolePermissionAssocDAO.GetByRoleIDs(ctx, finalRoleIDList)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}

	permMap := make(map[string]string)
	for _, p := range permList {
		permMap[p.Action] = p.PathPattern
	}

	return &api.GetUserPermissionsResponse{ActionToPathPattern: permMap}, nil
}
