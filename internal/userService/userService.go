package main

import (
	"flag"
	"gogogo/internal/userService/service"
	"gogogo/pkg/config"
	"gogogo/pkg/proto/api"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// confPath is the config flag.
	confPath string

	id, _ = os.Hostname()
)

func newApp(logger log.Logger, gs *grpc.Server, hs *http.Server) *kratos.App {
	return kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			gs,
			hs,
		),
	)
}

func main() {
	flag.StringVar(&confPath, "conf", "configs/config.ini", "config path, eg: -conf config.yaml")
	flag.Parse()

	config.LoadConf(confPath)

	var userCURDServer service.UserCURDServer
	var userServer service.UserServer

	var grpcSrv *grpc.Server
	{
		var opts = []grpc.ServerOption{
			grpc.Middleware(
				recovery.Recovery(),
			),
		}
		if config.GetConfStr("Grpc.Network") != "" {
			opts = append(opts, grpc.Network(config.GetConfStr("/Grpc/Network")))
		}
		if config.GetConfStr("Grpc.Addr") != "" {
			opts = append(opts, grpc.Address(config.GetConfStr("Grpc.Addr")))
		}
		if config.GetConfInt("Grpc.Timeout") != 0 {
			opts = append(opts, grpc.Timeout(time.Millisecond*
				time.Duration(config.GetConfInt("Grpc.Timeout"))))
		}
		grpcSrv = grpc.NewServer(opts...)
		api.RegisterUserCURDServer(grpcSrv, &userCURDServer)
		api.RegisterUserServer(grpcSrv, &userServer)
	}
	var httpSrv *http.Server
	{
		var opts = []http.ServerOption{
			http.Middleware(
				recovery.Recovery(),
			),
		}
		if config.GetConfStr("Http.Network") != "" {
			opts = append(opts, http.Network(config.GetConfStr("Http.Network")))
		}
		if config.GetConfStr("Http.Addr") != "" {
			opts = append(opts, http.Address(config.GetConfStr("Http.Addr")))
		}
		if config.GetConfInt("Http.Timeout") != 0 {
			opts = append(opts, http.Timeout(time.Millisecond*
				time.Duration(config.GetConfInt("Http.Timeout"))))
		}
		httpSrv = http.NewServer(opts...)
		api.RegisterUserCURDHTTPServer(httpSrv, &userCURDServer)
		api.RegisterUserHTTPServer(httpSrv, &userServer)
	}

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	app := newApp(logger, grpcSrv, httpSrv)

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
