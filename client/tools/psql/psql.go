package psql

import (
	"os"

	"github.com/pancake-lee/pgo/client/common"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func Psql() {
	cachePath := pconfig.GetDefaultCachePath()

	host := common.GetCachedParam(cachePath,
		"tools.psql.host",
		"请输入主机地址 (host): ",
		"localhost")
	portStr := common.GetCachedParam(cachePath,
		"tools.psql.port",
		"请输入端口号 (port): ",
		"5432")

	port, err := putil.StrToInt32(portStr)
	if err != nil {
		plogger.Errorf("Error: 无效的端口号 %s", portStr)
		putil.Interact.Input("输入端口号有误，回车返回菜单")
		return
	}

	user := common.GetCachedParam(cachePath,
		"tools.psql.user",
		"请输入用户名 (user): ",
		"root")
	database := common.GetCachedParam(cachePath,
		"tools.psql.database",
		"请输入数据库名 (database): ",
		"postgres")

	filename := common.GetCachedParam(cachePath,
		"tools.psql.filename",
		"请输入SQL文件路径 (filename): ",
		"")

	command := common.GetCachedParam(cachePath,
		"tools.psql.command",
		"请输入SQL语句 (command): ",
		"")

	// --------------------------------------------------
	password := putil.Interact.Input("password (可空，空则沿用环境变量PGPASSWORD): ")
	if password == "" {
		password = os.Getenv("PGPASSWORD")
	}

	// --------------------------------------------------
	err = pdb.InitPG(host, user, password, database, port)
	if err != nil {
		plogger.Errorf("Error: %v\n", err)
		os.Exit(1)
	}

	if command != "" {
		plogger.Debugf("sql cmd  [%s]", command)
		_, err = pdb.Exec(command)
		if err != nil {
			plogger.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	if filename != "" {
		plogger.Debugf("sql file[%s]", filename)
		err = pdb.ExecFile(filename)
		if err != nil {
			plogger.Errorf("Error: %v\n", err)
			os.Exit(1)
		}
		return
	}
	plogger.Errorf("Error: either command or file must be provided")
}
