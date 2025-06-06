package service

import (
	"context"

	api "pgo/api"
	"pgo/internal/userService/data"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type UserServer struct {
	api.UnimplementedUserServer
}

func (s *UserServer) Reg(grpcSrv *grpc.Server, httpSrv *http.Server) {
	if grpcSrv != nil {
		api.RegisterUserServer(grpcSrv, s)
	}
	if httpSrv != nil {
		api.RegisterUserHTTPServer(httpSrv, s)
	}
}

func (s *UserServer) Login(
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
	resp.User.ID = userData.ID
	resp.User.UserName = userData.UserName
	return resp, nil
}
func (s *UserServer) EditUserName(
	ctx context.Context, req *api.EditUserNameRequest,
) (resp *api.Empty, err error) {
	if req.ID == 0 || req.UserName == "" {
		return nil, api.ErrorInvalidArgument("argument invalid")
	}

	err = data.UserDAO.EditUserName(ctx, req.ID, req.UserName)
	if err != nil {
		return nil, err
	}
	return nil, nil
}

func (s *UserServer) DelUserDeptAssoc(
	ctx context.Context, req *api.DelUserDeptAssocRequest,
) (resp *api.Empty, err error) {
	if req.UserID == 0 || req.DeptID == 0 {
		return nil, api.ErrorInvalidArgument("argument invalid")
	}

	err = data.UserDeptAssocDAO.DelByPrimaryKey(ctx, req.UserID, req.DeptID)
	if err != nil {
		return nil, err
	}
	return nil, nil
}
