package service

import (
	"context"

	api "github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/internal/userService/data"
)

func (s *UserServer) GetUserPermissions(ctx context.Context, req *api.GetUserPermissionsRequest) (*api.GetUserPermissionsResponse, error) {
	// 1. Get all RoleIDs for the user
	userRoleAssocList, err := data.UserRoleAssocDAO.GetByUserID(ctx, req.UserID)
	if err != nil {
		return nil, err
	}

	var candidateRoleIDList []int32
	for _, ura := range userRoleAssocList {
		candidateRoleIDList = append(candidateRoleIDList, ura.RoleID)
	}

	// 2. Filter by ProjectID
	roleList, err := data.UserRoleDAO.GetByIDsAndProjectID(ctx, candidateRoleIDList, req.ProjectID)
	if err != nil {
		return nil, err
	}

	var finalRoleIDList []int32
	for _, r := range roleList {
		finalRoleIDList = append(finalRoleIDList, r.ID)
	}

	// 3. Get Permissions
	permList, err := data.UserRolePermissionAssocDAO.GetByRoleIDs(ctx, finalRoleIDList)
	if err != nil {
		return nil, err
	}

	permMap := make(map[string]string)
	for _, p := range permList {
		permMap[p.Action] = p.PathPattern
	}

	return &api.GetUserPermissionsResponse{ActionToPathPattern: permMap}, nil
}
