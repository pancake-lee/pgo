package app

import (
	"os"
	"pgo/pkg/config"
	"time"

	_ "go.uber.org/automaxprocs"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string

	id, _ = os.Hostname()
)

// --------------------------------------------------
type httpConfig struct {
	Addr    string
	Timeout int `default:"10"` // seconds
}

type grpcConfig struct {
	Addr    string
	Timeout int `default:"10"` // seconds
}

type ServiceConfig struct {
	Http httpConfig
	Grpc grpcConfig
}

// --------------------------------------------------
type kratosServer interface {
	Reg(grpcSrv *grpc.Server, httpSrv *http.Server)
}

func RunKratosApp(kratosServers ...kratosServer) {
	var conf ServiceConfig
	err := config.Scan(&conf)
	if err != nil {
		panic(err)
	}

	var grpcSrv *grpc.Server
	{
		var opts = []grpc.ServerOption{
			grpc.Middleware(
				recovery.Recovery(),
			),
		}
		if conf.Grpc.Addr != "" {
			opts = append(opts, grpc.Network(conf.Grpc.Addr))
		}
		if conf.Grpc.Timeout != 0 {
			opts = append(opts, grpc.Timeout(time.Millisecond*
				time.Duration(conf.Grpc.Timeout)))
		}

		grpcSrv = grpc.NewServer(opts...)
	}
	var httpSrv *http.Server
	{
		var opts = []http.ServerOption{
			http.Middleware(
				recovery.Recovery(),
			),
		}
		if conf.Http.Addr != "" {
			opts = append(opts, http.Network(conf.Http.Addr))
		}
		if conf.Http.Timeout != 0 {
			opts = append(opts, http.Timeout(time.Millisecond*
				time.Duration(conf.Http.Timeout)))
		}

		httpSrv = http.NewServer(opts...)
	}

	for _, s := range kratosServers {
		s.Reg(grpcSrv, httpSrv)
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
	app := kratos.New(
		kratos.ID(id),
		kratos.Name(Name),
		kratos.Version(Version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(logger),
		kratos.Server(
			grpcSrv,
			httpSrv,
		),
	)

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
