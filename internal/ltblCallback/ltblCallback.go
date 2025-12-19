package main

import (
	"flag"

	_ "github.com/pancake-lee/pgo/internal/ltblCallback/service" // Register handlers
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/predis"
)

// ltblCallback 本地数据库变更回调服务
// ltbl = local table 则本地数据库
// 是相对于mtbl = multi table 多维表格而言的

func main() {
	l := flag.Bool("l", false, "log to console, default is false")
	c := flag.String("c", "",
		"config folder, should have common.yaml and ${execName}.yaml")
	flag.Parse()

	pconfig.MustInitConfig(*c)
	plogger.InitFromConfig(*l)
	predis.MustInitRedisByConfig()

	var dbConf pdb.MysqlConfig
	err := pconfig.Scan(&dbConf)
	if err != nil {
		panic(err)
	}

	client, err := pdb.NewCanal(dbConf)
	if err != nil {
		plogger.Error(err)
		return
	}

	if err := client.Run(); err != nil {
		plogger.Error(err)
	}
}
