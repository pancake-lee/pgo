package main

import (
	"flag"

	"github.com/pancake-lee/pgo/internal/userService/conf"
	"github.com/pancake-lee/pgo/internal/userService/service"
	"github.com/pancake-lee/pgo/pkg/papitable"
	"github.com/pancake-lee/pgo/pkg/papp"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	pdb.MustInitMysqlByConfig()

	// --------------------------------------------------
	err := papitable.InitAPITableByConfig()
	if err != nil {
		panic(err)
	}

	pmq.MustInitMQByConfig()
	pmq.DefaultClient.DeclareServerEvent(
		"apitable_event", "apitable_change", true,
		service.OnMtblUpdateUser)

	// --------------------------------------------------
	err = pconfig.Scan(&conf.UserSvcConf)
	if err != nil {
		panic(err)
	}

	var userCURDServer service.UserCURDServer
	var userServer service.UserServer
	papp.AddWhiteList("/user/token")
	papp.SetHTTPAuthKey(conf.UserSvcConf.TokenSK)
	papp.RunKratosApp(&userServer, &userCURDServer)
}
