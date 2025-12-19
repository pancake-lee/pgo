package service

import (
	"context"
	"testing"
	"time"

	api "github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// 1：创建用户、项目、角色
// 2：用户加入项目，分配角色
// 3：角色分配权限
// 4：检查数据，判断拥有的权限和没有的权限
// 5：删除数据
func TestUserRolePermission(t *testing.T) {
	pconfig.MustInitConfig("../../../configs/pancake.yaml")
	pdb.MustInitMysqlByConfig()

	// Drop tables to ensure clean state
	// pdb.GetGormDB().Migrator().DropTable(
	// Auto migrate tables
	// if err := pdb.GetGormDB().AutoMigrate(

	ctx := context.Background()

	// var userCURDSvr UserCURDServer
	var userSvr UserServer

	var actCode1 string = "read_data"
	// var actCode2 string = "write_data"

	nowStr := putil.TimeToStrDefault(time.Now())
	var userName string = "pancake" + nowStr

	var userId int32
	{
		userId = testAddUser(t, userName)
		defer testDelUser(t, userId)
	}

	// 1. Add Project
	projName := "test_proj_" + nowStr
	var projId int32
	{
		projId = testAddProject(t, projName, userId)
		defer testDelProject(t, projId)
	}

	// 2. Add UserProjectAssoc
	var userProjAssocId int32
	{
		userProjAssocId = testAddUserProjectAssoc(t, userId, projId)
		defer testDelUserProjectAssoc(t, userProjAssocId)
	}

	// 3. Add UserRole
	roleName := "test_role_" + nowStr
	var roleId int32
	{
		roleId = testAddUserRole(t, projId, roleName, userId)
		defer testDelUserRole(t, roleId)
	}

	// 4. Add UserRoleAssoc
	var userRoleAssocId int32
	{
		userRoleAssocId = testAddUserRoleAssoc(t, userId, roleId, userId)
		defer testDelUserRoleAssoc(t, userRoleAssocId)
	}

	// 5. Add UserRolePermissionAssoc
	var userRolePermAssocId int32
	{
		userRolePermAssocId = testAddUserRolePermissionAssoc(t, roleId, actCode1, userId)
		defer testDelUserRolePermissionAssoc(t, userRolePermAssocId)
	}

	// 6. GetUserPermissions
	permsResp, err := userSvr.GetUserPermissions(ctx, &api.GetUserPermissionsRequest{
		UserID:    userId,
		ProjectID: projId,
	})
	if err != nil {
		t.Fatal(err)
	}
	plogger.Debugf("user permission : %v", permsResp.ActionToPathPattern)
	if _, ok := permsResp.ActionToPathPattern[actCode1]; !ok {
		t.Fatalf("expected action %s, but not found", actCode1)
	}
}

func testAddProject(t *testing.T, projName string, userId int32) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddProject(ctx, &api.AddProjectRequest{
		Project: &api.ProjectInfo{
			ProjName:   projName,
			CreateUser: userId,
			UpdateUser: userId,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.Project.ID
}

func testDelProject(t *testing.T, projId int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelProjectByIDList(ctx, &api.DelProjectByIDListRequest{IDList: []int32{projId}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddUserProjectAssoc(t *testing.T, userId, projId int32) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUserProjectAssoc(ctx, &api.AddUserProjectAssocRequest{
		UserProjectAssoc: &api.UserProjectAssocInfo{
			UserID:     userId,
			ProjID:     projId,
			CreateUser: userId,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.UserProjectAssoc.ID
}

func testDelUserProjectAssoc(t *testing.T, id int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserProjectAssocByIDList(ctx, &api.DelUserProjectAssocByIDListRequest{IDList: []int32{id}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddUserRole(t *testing.T, projId int32, roleName string, userId int32) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUserRole(ctx, &api.AddUserRoleRequest{
		UserRole: &api.UserRoleInfo{
			ProjID:     projId,
			RoleName:   roleName,
			CreateUser: userId,
			UpdateUser: userId,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.UserRole.ID
}

func testDelUserRole(t *testing.T, id int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserRoleByIDList(ctx, &api.DelUserRoleByIDListRequest{IDList: []int32{id}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddUserRoleAssoc(t *testing.T, userId, roleId, createUser int32) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUserRoleAssoc(ctx, &api.AddUserRoleAssocRequest{
		UserRoleAssoc: &api.UserRoleAssocInfo{
			UserID:     userId,
			RoleID:     roleId,
			CreateUser: createUser,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.UserRoleAssoc.ID
}

func testDelUserRoleAssoc(t *testing.T, id int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserRoleAssocByIDList(ctx, &api.DelUserRoleAssocByIDListRequest{IDList: []int32{id}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddUserRolePermissionAssoc(t *testing.T, roleId int32, action string, createUser int32) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUserRolePermissionAssoc(ctx, &api.AddUserRolePermissionAssocRequest{
		UserRolePermissionAssoc: &api.UserRolePermissionAssocInfo{
			RoleID:      roleId,
			Action:      action,
			PathPattern: "/api/v1/data",
			CreateUser:  createUser,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return resp.UserRolePermissionAssoc.ID
}

func testDelUserRolePermissionAssoc(t *testing.T, id int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserRolePermissionAssocByIDList(ctx, &api.DelUserRolePermissionAssocByIDListRequest{IDList: []int32{id}})
	if err != nil {
		t.Fatal(err)
	}
}
