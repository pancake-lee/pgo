package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport"

	api "github.com/pancake-lee/pgo/internal/pkg/api"
	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papp"
)

// permissionAggregationHeader 标识练习请求使用的权限合并算法。
const permissionAggregationHeader = "X-Pgo-Permission-Aggregation"

// GetUserPermissions 查询指定项目的角色权限并按动作合并。
func (s *UserServer) GetUserPermissions(
	_ctx context.Context, req *api.GetUserPermissionsRequest,
) (*api.GetUserPermissionsResponse, error) {
	aggregation, err := s.getPermissionAggregation(_ctx)
	if err != nil {
		return nil, err
	}
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
	roleList, err := data.UserRoleDAO.GetByIDsAndProjectID(
		ctx,
		candidateRoleIDList,
		req.ProjectID,
	)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}

	var finalRoleIDList []int32
	for _, r := range roleList {
		finalRoleIDList = append(finalRoleIDList, r.ID)
	}

	// 3. Get Permissions
	permList, err := data.UserRolePermissionAssocDAO.GetByRoleIDs(
		ctx,
		finalRoleIDList,
	)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}

	var permMap map[string]string
	if aggregation == "list" {
		permMap = mergePermissionsByList(permList)
	} else {
		permMap = mergePermissionsByMap(permList)
	}

	return &api.GetUserPermissionsResponse{ActionToPathPattern: permMap}, nil
}

// getPermissionAggregation 校验练习开关并回传实际采用的算法。
func (s *UserServer) getPermissionAggregation(ctx context.Context,
) (string, error) {
	aggregation := "map"
	tr, ok := transport.FromServerContext(ctx)
	if ok && tr.RequestHeader().Get(permissionAggregationHeader) != "" {
		aggregation = tr.RequestHeader().Get(permissionAggregationHeader)
	}
	if aggregation != "map" && aggregation != "list" {
		return "", api.ErrorInvalidArgument("unknown permission aggregation")
	}
	if aggregation == "list" && !s.PermissionExercise {
		return "", api.ErrorInvalidArgument(
			"set PermissionExercise: true in the service config to use list",
		)
	}
	if ok {
		tr.ReplyHeader().Set(permissionAggregationHeader, aggregation)
	}
	return aggregation, nil
}

// mergePermissionsByList 模拟逐项扫描已有结果的低效权限合并。
func mergePermissionsByList(permissionList []*model.UserRolePermissionAssoc,
) map[string]string {
	mergedList := make([]model.UserRolePermissionAssoc, 0)
	for _, permission := range permissionList {
		found := false
		for index := range mergedList {
			if mergedList[index].Action == permission.Action {
				mergedList[index].PathPattern = permission.PathPattern
				found = true
				break
			}
		}
		if !found {
			mergedList = append(mergedList, *permission)
		}
	}

	permissionMap := make(map[string]string)
	for _, permission := range mergedList {
		permissionMap[permission.Action] = permission.PathPattern
	}
	return permissionMap
}

// mergePermissionsByMap 使用动作索引直接合并权限并保持后值覆盖。
func mergePermissionsByMap(permissionList []*model.UserRolePermissionAssoc,
) map[string]string {
	permissionMap := make(map[string]string)
	for _, permission := range permissionList {
		permissionMap[permission.Action] = permission.PathPattern
	}
	return permissionMap
}
