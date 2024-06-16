package service

import (
	"context"
	"gogogo/pkg/proto/api"
	"testing"
)

func TestUserService(t *testing.T) {

	ctx := context.Background()

	var userCURDSvr UserCURDServer
	var userSvr UserServer

	var userId int32
	{
		var req api.LoginRequest
		req.UserName = "pancake"
		resp, err := userSvr.Login(ctx, &req)
		if err != nil {
			t.Fatal(err)
		}

		if resp.User.ID == 0 ||
			resp.User.UserName != "pancake" {
			t.Fatal("user info is error : ", resp.User)
		}
		userId = resp.User.ID
	}
	{
		resp, err := userCURDSvr.GetUserList(ctx, nil)
		if err != nil {
			t.Fatal(err)
		}
		if len(resp.UserList) == 0 {
			t.Fatal("user list is empty")
		}
		isFound := false
		for _, user := range resp.UserList {
			if user.UserName == "pancake" {
				if user.ID != userId {
					t.Fatal("user id is error")
				}
				isFound = true
				break
			}
		}
		if !isFound {
			t.Fatal("user pancake is not found")
		}
	}
	{
		var req api.EditUserNameRequest
		req.ID = userId
		req.UserName = "pancake2"
		_, err := userSvr.EditUserName(ctx, &req)
		if err != nil {
			t.Fatal(err)
		}
	}
	{
		var req api.DelUserByIDListRequest
		req.IDList = append(req.IDList, userId)
		_, err := userCURDSvr.DelUserByIDList(ctx, &req)
		if err != nil {
			t.Fatal(err)
		}
	}
}
