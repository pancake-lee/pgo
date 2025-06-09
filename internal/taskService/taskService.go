package main

import (
	"flag"
	"pgo/internal/taskService/service"
	"pgo/pkg/app"
	"pgo/pkg/config"
	"pgo/pkg/logger"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "./configs/",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	config.MustInitConfig(*c)
	logger.InitServiceLogger(*l)

	var taskCURDServer service.TaskCURDServer

	app.RunKratosApp(&taskCURDServer)
}
