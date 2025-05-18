package service

import (
	"context"
	"pgo/pkg/proto/api"
	"pgo/pkg/util"
	"testing"
	"time"
)

// 1：添加用户
// 2：获取用户列表，找到新添加的用户
// 3：修改用户名称
// 4：用新的名称登录
// 5：删除用户
func TestUserService(t *testing.T) {

	ctx := context.Background()

	var userCURDSvr UserCURDServer
	var userSvr UserServer

	nowStr := util.TimeToStrDefault(time.Now())
	var userName string = "pancake" + nowStr
	var newUserName string = "pancake2" + nowStr
	var userId int32
	{
		userId = testAddUser(t, userName)
		defer testDelUser(t, userId)
	}
	{
		resp, err := userCURDSvr.GetUserList(ctx,
			&api.GetUserListRequest{IDList: []int32{userId}})
		if err != nil {
			t.Fatal(err)
		}
		i := util.SliceIndex(resp.UserList, func(i int) bool {
			return resp.UserList[i].UserName == userName
		})
		if i == -1 {
			t.Fatal("user pancake is not found")
		}
		if resp.UserList[i].ID != userId {
			t.Fatal("user id is error")
		}
	}
	{
		_, err := userSvr.EditUserName(ctx,
			&api.EditUserNameRequest{ID: userId, UserName: newUserName})
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		resp, err := userSvr.Login(ctx,
			&api.LoginRequest{UserName: newUserName})
		if err != nil {
			t.Fatal(err)
		}
		if resp.User.ID != userId || resp.User.UserName != newUserName {
			t.Fatal("user info is error : ", resp.User)
		}
	}
}

// 1：创建用户、部门、职位
// 2：把用户以职位1的身份加入部门1
// 3：把用户以职位2的身份加入部门2
// 4：检查数据
// 5：删除数据
func TestUserDeptJob(t *testing.T) {
	ctx := context.Background()

	var userCURDSvr UserCURDServer

	nowStr := util.TimeToStrDefault(time.Now())
	var userName string = "pancake" + nowStr
	var deptName1 string = "dept1_" + nowStr
	var deptName2 string = "dept2_" + nowStr
	var jobName1 string = "job1_" + nowStr
	var jobName2 string = "job2_" + nowStr

	userId := testAddUser(t, userName)
	defer testDelUser(t, userId)

	deptId1 := testAddDept(t, deptName1)
	defer testDelDept(t, deptId1)
	deptId2 := testAddDept(t, deptName2)
	defer testDelDept(t, deptId2)

	jobId1 := testAddJob(t, jobName1)
	defer testDelJob(t, jobId1)
	jobId2 := testAddJob(t, jobName2)
	defer testDelJob(t, jobId2)

	testAddUserToDept(t, userId, deptId1, jobId1)
	defer testDelUserFromDept(t, userId, deptId1)
	testAddUserToDept(t, userId, deptId2, jobId2)
	defer testDelUserFromDept(t, userId, deptId2)

	{
		resp, err := userCURDSvr.GetUserDeptAssocList(ctx,
			&api.GetUserDeptAssocListRequest{})
		if err != nil {
			t.Fatal(err)
		}
		i := util.SliceIndex(resp.UserDeptAssocList, func(i int) bool {
			item := resp.UserDeptAssocList[i]
			return item.UserID == userId &&
				item.DeptID == deptId1 &&
				item.JobID == jobId1
		})
		if i == -1 {
			t.Log(resp.UserDeptAssocList)
			t.Fatal("user dept assoc is not found")
		}
		i = util.SliceIndex(resp.UserDeptAssocList, func(i int) bool {
			item := resp.UserDeptAssocList[i]
			return item.UserID == userId &&
				item.DeptID == deptId2 &&
				item.JobID == jobId2
		})
		if i == -1 {
			t.Log(resp.UserDeptAssocList)
			t.Fatal("user dept assoc is not found")
		}
	}
}

func testAddUser(t *testing.T, userName string) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUser(ctx,
		&api.AddUserRequest{User: &api.UserInfo{UserName: userName}})
	if err != nil {
		t.Fatal(err)
	}
	if resp.User.ID == 0 || resp.User.UserName != userName {
		t.Fatal("user info is error : ", resp.User)
	}
	return resp.User.ID
}
func testDelUser(t *testing.T, userId int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserByIDList(ctx,
		&api.DelUserByIDListRequest{IDList: []int32{userId}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddDept(t *testing.T, deptName string) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUserDept(ctx,
		&api.AddUserDeptRequest{UserDept: &api.UserDeptInfo{
			DeptPath: "/" + deptName, DeptName: deptName}})
	if err != nil {
		t.Fatal(err)
	}
	return resp.UserDept.ID
}
func testDelDept(t *testing.T, deptId int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserDeptByIDList(ctx,
		&api.DelUserDeptByIDListRequest{IDList: []int32{deptId}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddJob(t *testing.T, jobName string) int32 {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	resp, err := userCURDSvr.AddUserJob(ctx,
		&api.AddUserJobRequest{UserJob: &api.UserJobInfo{JobName: jobName}})
	if err != nil {
		t.Fatal(err)
	}
	return resp.UserJob.ID
}

func testDelJob(t *testing.T, jobId int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.DelUserJobByIDList(ctx,
		&api.DelUserJobByIDListRequest{IDList: []int32{jobId}})
	if err != nil {
		t.Fatal(err)
	}
}

func testAddUserToDept(t *testing.T, userId, deptId, jobId int32) {
	ctx := context.Background()
	var userCURDSvr UserCURDServer
	_, err := userCURDSvr.AddUserDeptAssoc(ctx,
		&api.AddUserDeptAssocRequest{UserDeptAssoc: &api.UserDeptAssocInfo{
			UserID: userId, DeptID: deptId, JobID: jobId}})
	if err != nil {
		t.Fatal(err)
	}
}
func testDelUserFromDept(t *testing.T, userId, deptId int32) {
	ctx := context.Background()
	var userSvr UserServer
	_, err := userSvr.DelUserDeptAssoc(ctx,
		&api.DelUserDeptAssocRequest{UserID: userId, DeptID: deptId})
	if err != nil {
		t.Fatal(err)
	}
}
