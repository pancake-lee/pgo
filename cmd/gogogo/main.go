package main

import (
	"flag"
	v1 "gogogo/api/helloworld/v1"
	"gogogo/internal/service"
	"os"
	"time"

	"github.com/go-kratos/kratos/v2"
	"github.com/go-kratos/kratos/v2/log"
	"github.com/go-kratos/kratos/v2/middleware/recovery"
	"github.com/go-kratos/kratos/v2/middleware/tracing"
	"github.com/go-kratos/kratos/v2/transport/grpc"
	"github.com/go-kratos/kratos/v2/transport/http"
	"github.com/pelletier/go-toml"

	_ "go.uber.org/automaxprocs"
)

// go build -ldflags "-X main.Version=x.y.z"
var (
	// Name is the name of the compiled software.
	Name string
	// Version is the version of the compiled software.
	Version string
	// flagconf is the config flag.
	flagconf string

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
	flag.StringVar(&flagconf, "conf", "../../configs/config.ini", "config path, eg: -conf config.yaml")
	flag.Parse()

	loadConf()

	logger := log.With(log.NewStdLogger(os.Stdout),
		"ts", log.DefaultTimestamp,
		"caller", log.DefaultCaller,
		"service.id", id,
		"service.name", Name,
		"service.version", Version,
		"trace.id", tracing.TraceID(),
		"span.id", tracing.SpanID(),
	)
	var greeterService service.GreeterService

	var grpcSrv *grpc.Server
	{
		var opts = []grpc.ServerOption{
			grpc.Middleware(
				recovery.Recovery(),
			),
		}
		if getConfStr("Grpc.Network") != "" {
			opts = append(opts, grpc.Network(getConfStr("/Grpc/Network")))
		}
		if getConfStr("Grpc.Addr") != "" {
			opts = append(opts, grpc.Address(getConfStr("Grpc.Addr")))
		}
		if getConfInt("Grpc.Timeout") != 0 {
			opts = append(opts, grpc.Timeout(time.Millisecond*
				time.Duration(getConfInt("Grpc.Timeout"))))
		}
		grpcSrv = grpc.NewServer(opts...)
		v1.RegisterGreeterServer(grpcSrv, &greeterService)
	}
	var httpSrv *http.Server
	{
		var opts = []http.ServerOption{
			http.Middleware(
				recovery.Recovery(),
			),
		}
		if getConfStr("Http.Network") != "" {
			opts = append(opts, http.Network(getConfStr("Http.Network")))
		}
		if getConfStr("Http.Addr") != "" {
			opts = append(opts, http.Address(getConfStr("Http.Addr")))
		}
		if getConfInt("Http.Timeout") != 0 {
			opts = append(opts, http.Timeout(time.Millisecond*
				time.Duration(getConfInt("Http.Timeout"))))
		}
		httpSrv = http.NewServer(opts...)
		v1.RegisterGreeterHTTPServer(httpSrv, &greeterService)
	}
	app := newApp(logger, grpcSrv, httpSrv)

	// start and wait for stop signal
	if err := app.Run(); err != nil {
		panic(err)
	}
}

// 临时简单实现配置的读取，后续肯定是要优化的
var config *toml.Tree

func loadConf() {
	c, err := toml.LoadFile(flagconf)
	if err != nil {
		panic(err)
	}
	config = c
}

func getConfStr(confKey string) string {
	if v, ok := config.Get(confKey).(string); ok {
		log.Debugf("config key: %s, value: %s", confKey, v)
		return v
	}
	return ""
}

func getConfInt(confKey string) int {
	if v, ok := config.Get(confKey).(int); ok {
		log.Debugf("config key: %s, value: %d", confKey, v)
		return v
	}
	return 0
}
