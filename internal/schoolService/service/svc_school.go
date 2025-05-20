package service

import (
	"pgo/pkg/proto/api"

	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

type SchoolCURDServer struct {
	api.UnimplementedSchoolCURDServer
}

func (s *SchoolCURDServer) Reg(grpcSrv *grpc.Server, httpSrv *http.Server) {
	if grpcSrv != nil {
		api.RegisterSchoolCURDServer(grpcSrv, s)
	}
	if httpSrv != nil {
		api.RegisterSchoolCURDHTTPServer(httpSrv, s)
	}
}
