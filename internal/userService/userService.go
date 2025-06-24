package main

import (
	"flag"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/service"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "./configs/",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitServiceLogger(*l)

	err := pconfig.Scan(&conf.UserSvcConf)
	if err != nil {
		panic(err)
	}

	var userCURDServer service.UserCURDServer
	var userServer service.UserServer

	papp.RunKratosApp(&userServer, &userCURDServer)
}
