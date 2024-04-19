package service

import (
	"context"

	"gogogo/internal/userService/data"
	"gogogo/pkg/api"
)

type UserService struct {
	api.UnimplementedUserServer
}

func (s *UserService) Login(
	ctx context.Context, req *api.LoginRequest,
) (resp *api.LoginResponse, err error) {
	if req.UserName == "" {
		return nil, api.ErrorInvalidArgument("user name is empty")
	}
	userData, err := data.UserDAO.GetOrAdd(ctx,
		&data.UserDO{UserName: req.UserName})
	if err != nil {
		return nil, err
	}
	resp = new(api.LoginResponse)
	resp.User = new(api.UserInfo)
	resp.User.Id = userData.ID
	resp.User.UserName = userData.UserName
	return resp, nil
}
func (s *UserService) GetUserList(
	ctx context.Context, req *api.Empty,
) (resp *api.GetUserListResponse, err error) {

	userDataList, err := data.UserDAO.GetAll(ctx)
	if err != nil {
		return nil, err
	}
	resp = new(api.GetUserListResponse)
	resp.UserList = make([]*api.UserInfo, 0, len(userDataList))
	for _, userData := range userDataList {
		resp.UserList = append(resp.UserList, &api.UserInfo{
			Id:       userData.ID,
			UserName: userData.UserName,
		})
	}
	return resp, nil
}
func (s *UserService) DelUser(
	ctx context.Context, req *api.DelUserRequest,
) (resp *api.Empty, err error) {
	if req.Id == 0 {
		return nil, api.ErrorInvalidArgument("user id is zero")
	}
	err = data.UserDAO.Del(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
func (s *UserService) EditUserName(
	ctx context.Context, req *api.EditUserNameRequest,
) (resp *api.Empty, err error) {
	if req.Id == 0 || req.UserName == "" {
		return nil, api.ErrorInvalidArgument("argument invalid")
	}

	err = data.UserDAO.EditUserName(ctx, req.Id, req.UserName)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
