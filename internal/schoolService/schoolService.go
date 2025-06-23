package main

import (
	"flag"

	"github.com/pancake-lee/pgo/internal/schoolService/service"
	"github.com/pancake-lee/pgo/pkg/app"
	"github.com/pancake-lee/pgo/pkg/config"
	"github.com/pancake-lee/pgo/pkg/logger"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "./configs/",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	config.MustInitConfig(*c)
	logger.InitServiceLogger(*l)

	var curdServer service.SchoolCURDServer

	app.RunKratosApp(&curdServer)
}
