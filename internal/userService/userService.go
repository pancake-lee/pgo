package main

import (
	"flag"
	"pgo/internal/userService/conf"
	"pgo/internal/userService/service"
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

	err := config.Scan(&conf.UserSvcConf)
	if err != nil {
		panic(err)
	}

	var userCURDServer service.UserCURDServer
	var userServer service.UserServer

	app.RunKratosApp(&userServer, &userCURDServer)
}
