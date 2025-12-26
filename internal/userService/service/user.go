package service

import (
	"context"

	api "github.com/pancake-lee/pgo/api"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/papp"

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
	_ctx context.Context, req *api.LoginRequest,
) (resp *api.LoginResponse, err error) {
	ctx := papp.NewAppCtx(_ctx)

	if req.UserName == "" {
		return nil, api.ErrorInvalidArgument("user name is empty")
	}
	userData, err := data.UserDAO.GetOrAdd(ctx,
		&data.UserDO{UserName: req.UserName})
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}
	resp = new(api.LoginResponse)
	resp.User = new(api.UserInfo)
	resp.User.ID = userData.ID
	resp.User.UserName = userData.UserName
	resp.Token, err = papp.GenToken(userData.ID)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}
	return resp, nil
}
func (s *UserServer) EditUserName(
	_ctx context.Context, req *api.EditUserNameRequest,
) (resp *api.Empty, err error) {
	ctx := papp.NewAppCtx(_ctx)

	if req.ID == 0 || req.UserName == "" {
		return nil, api.ErrorInvalidArgument("")
	}

	err = data.UserDAO.EditUserName(ctx, req.ID, req.UserName)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}
	return nil, nil
}

func (s *UserServer) DelUserDeptAssoc(
	_ctx context.Context, req *api.DelUserDeptAssocRequest,
) (resp *api.Empty, err error) {
	ctx := papp.NewAppCtx(_ctx)

	if req.UserID == 0 || req.DeptID == 0 {
		return nil, api.ErrorInvalidArgument("")
	}

	err = data.UserDeptAssocDAO.DelByPrimaryKey(ctx, req.UserID, req.DeptID)
	if err != nil {
		return nil, ctx.Log.LogErr(err)
	}
	return nil, nil
}
