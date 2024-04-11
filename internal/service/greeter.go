package service

import (
	"context"

	v1 "gogogo/api/helloworld/v1"
)

// GreeterService is a greeter service.
type GreeterService struct {
	v1.UnimplementedGreeterServer
}

// SayHello implements helloworld.GreeterServer.
func (s *GreeterService) SayHello(ctx context.Context, in *v1.HelloRequest) (*v1.HelloReply, error) {
	return &v1.HelloReply{Message: "Hello " + in.Name}, nil
}
