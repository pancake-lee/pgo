package service

import (
	"context"

	api "github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papp"
)

func (s *UserServer) GetUserPermissions(_ctx context.Context, req *api.GetUserPermissionsRequest) (*api.GetUserPermissionsResponse, error) {
	ctx := papp.NewAppCtx(_ctx)

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
