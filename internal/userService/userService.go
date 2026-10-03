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

// main 初始化用户服务配置与业务、诊断接口。
func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	pdb.MustInitMysqlByConfig()

	err := pconfig.Scan(&conf.UserSvcConf)
	if err != nil {
		panic(err)
	}
	// --------------------------------------------------
	err = papitable.InitAPITableByConfig()
	if err != nil {
		panic(err)
	}

	pmq.MustInitMQByConfig()

	if conf.UserSvcConf.APITable.UserSheetID != "" {
		pmq.DefaultClient.DeclareServerEvent("apitable",
			"apitable.change."+conf.UserSvcConf.APITable.UserSheetID,
			true, service.OnMtblUpdateUser)
	}

	if conf.UserSvcConf.APITable.ProjectSheetID != "" {
		pmq.DefaultClient.DeclareServerEvent("apitable",
			"apitable.change."+conf.UserSvcConf.APITable.ProjectSheetID,
			true, service.OnMtblUpdateProject)
	}

	// --------------------------------------------------
	var userCURDServer service.UserCURDServer
	userServer := service.UserServer{
		PermissionExercise: conf.UserSvcConf.PermissionExercise,
	}
	papp.AddWhiteList("/", "/user/token")
	papp.SetHTTPAuthKey(conf.UserSvcConf.TokenSK)
	papp.RunKratosApp(&userServer, &userCURDServer)
}
