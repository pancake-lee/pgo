package main

import (
	"flag"
	"pgo/internal/abandonCodeService/service"
	"pgo/pkg/app"
	"pgo/pkg/config"
	"pgo/pkg/logger"
)

func main() {
	c := flag.String("c", "./configs/",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	config.MustInitConfig(*c)
	logger.InitServiceLogger()

	var s service.AbandonCodeCURDServer

	app.RunKratosApp(&s)
}
