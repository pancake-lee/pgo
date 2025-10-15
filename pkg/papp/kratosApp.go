package papp

import (
	// 新增 context 包

	"os"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"

	_ "go.uber.org/automaxprocs"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/rs/cors"
)

// go build -ldflags "-X 'github.com/pancake-lee/pgo/pkg/papp.version=x.y.z'"
var (
	version string
)

// --------------------------------------------------
type httpConfig struct {
	Addr    string
	Timeout int `default:"1000"` // Millisecond
}

type grpcConfig struct {
	Addr    string
	Timeout int `default:"1000"` // Millisecond
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
	err := pconfig.Scan(&conf)
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
			opts = append(opts, grpc.Address(conf.Grpc.Addr))
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
				authMiddleware(),
			),
			http.Filter(cors.New(cors.Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
				ExposedHeaders: []string{"Accept", "Accept-Encoding",
					"X-CSRF-Token", "Authorization", "Content-Type", "Content-Length"},
				AllowCredentials: true,
				MaxAge:           60,
			}).Handler),
		}

		if conf.Http.Addr != "" {
			opts = append(opts, http.Address(conf.Http.Addr))
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

	name := putil.GetExecName()

	id, _ := os.Hostname()
	if id != "" {
		id += "_"
	}
	id += name

	app := kratos.New(
		kratos.ID(id),
		kratos.Name(name),
		kratos.Version(version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(plogger.GetDefaultLogger()),
		kratos.Server(grpcSrv, httpSrv),
	)

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
