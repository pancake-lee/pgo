package service

import (
	"context"
	"slices"
	"testing"
	"time"

	"github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/pkg/putil"
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

	nowStr := putil.TimeToStrDefault(time.Now())
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

		i := slices.IndexFunc(resp.UserList, func(user *api.UserInfo) bool {
			return user.UserName == userName
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
