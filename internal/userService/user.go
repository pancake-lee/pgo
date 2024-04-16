package userService

import (
	"context"

	"gogogo/api"
	"gogogo/db/dao/query"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func init() {
	dsn := "host=192.168.3.18 user=gogogo password=gogogo dbname=gogogo port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	_db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		panic(err)
	}
	db = _db
}

type UserService struct {
	api.UnimplementedUserServer
}

func (s *UserService) Login(
	ctx context.Context, req *api.LoginRequest,
) (resp *api.LoginResponse, err error) {
	if req.UserName == "" {
		return nil, api.ErrorInvalidArgument("user name is empty")
	}
	q := query.Use(db).User
	userData, err := q.WithContext(ctx).
		Where(q.UserName.Eq(req.UserName)).
		Attrs(q.UserName.Value(req.UserName)).
		FirstOrCreate()
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
	q := query.Use(db).User
	userDataList, err := q.WithContext(ctx).Find()
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
	q := query.Use(db).User
	_, err = q.WithContext(ctx).
		Where(q.ID.Eq(req.Id)).
		Delete()
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
	q := query.Use(db).User
	_, err = q.WithContext(ctx).
		Where(q.ID.Eq(req.Id)).
		UpdateSimple(q.UserName.Value(req.UserName))
	if err != nil {
		return nil, err
	}
	return nil, nil
}
