package papp

import (
	"context"
	"os"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"

	_ "go.uber.org/automaxprocs"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/logging"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport"
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

type diagnosticsConfig struct {
	Enabled bool
	Addr    string `default:"127.0.0.1:19090"`
	Pprof   bool
}

type ServiceConfig struct {
	Http        httpConfig
	Grpc        grpcConfig
	Diagnostics diagnosticsConfig
}

// --------------------------------------------------
type kratosServer interface {
	Reg(grpcSrv *grpc.Server, httpSrv *http.Server)
}

func RunKratosApp(kratosServers ...kratosServer) {

	kLogger := log.With(plogger.GetDefaultLoggerNoCaller(),
		// 这里必须log.Valuer转换，否则只是一个匿名函数指针
		// log包实现代码判断了这个类型，才会动态获取值
		"tid", log.Valuer(func(ctx context.Context) interface{} {
			tid := tracing.TraceID()(ctx)
			if tid == "" {
				t := ctx.Value(putil.PgoTraceIDKey)
				if t != nil {
					tid = t
				}
			}
			return putil.StrPrefixByNum(putil.AnyToStr(tid), 6)
		}),
		"sid", log.Valuer(func(ctx context.Context) interface{} {
			return putil.StrPrefixByNum(
				putil.AnyToStr(tracing.SpanID()(ctx)), 6)
		}),
	)
	plogger.SetDefaultLogger(kLogger)
	plogger.SetPrefixKeys("tid")

	var conf ServiceConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		panic(err)
	}

	hasHTTP := pconfig.Has("Http")
	hasGRPC := pconfig.Has("Grpc")
	if !hasHTTP && !hasGRPC {
		panic("neither Http nor Grpc config found, at least one must be configured")
	}

	var grpcSrv *grpc.Server
	if hasGRPC {
		var opts = []grpc.ServerOption{
			grpc.Middleware(
				recovery.Recovery(),
				tracingAndObservabilityMiddleware(),
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
	if hasHTTP {
		var opts = []http.ServerOption{
			http.Middleware(
				recovery.Recovery(),
				tracingAndObservabilityMiddleware(),
				authMiddleware2(),
				logging.Server(kLogger),
			),
			http.Filter(cors.New(cors.Options{
				AllowedOrigins: []string{"*"},
				AllowedMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
				AllowedHeaders: []string{"*"},
				ExposedHeaders: []string{"Accept", "Accept-Encoding",
					"X-CSRF-Token", "X-Request-ID", "Authorization", "Content-Type", "Content-Length"},
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

	var serverList []transport.Server
	if grpcSrv != nil {
		serverList = append(serverList, grpcSrv)
	}
	if httpSrv != nil {
		serverList = append(serverList, httpSrv)
	}
	if conf.Diagnostics.Enabled {
		serverList = append(serverList, newDiagnosticsServer(conf.Diagnostics.Addr, conf.Diagnostics.Pprof))
	}

	app := kratos.New(
		kratos.ID(id),
		kratos.Name(name),
		kratos.Version(version),
		kratos.Metadata(map[string]string{}),
		kratos.Logger(plogger.GetDefaultLogger()),
		kratos.Server(serverList...),
	)

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}
