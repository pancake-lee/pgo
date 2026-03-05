package psql

import (
	"errors"
	"fmt"
	"os"

	"github.com/pancake-lee/pgo/client/common"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"github.com/spf13/cobra"
)

// --------------------------------------------------
const (
	paramNameHost     = "host"
	paramNamePort     = "port"
	paramNameUser     = "user"
	paramNameDatabase = "database"
	paramNameFilename = "filename"
	paramNameCommand  = "command"
	paramNamePassword = "password"
	cacheKeyPrefix    = "tools.psql."
)

var paramSettingList = []common.ParamItem{
	{
		Name:    paramNameHost,
		Usage:   "postgres host",
		Default: "localhost",
	}, {
		Name:    paramNamePort,
		Usage:   "postgres port",
		Default: "5432",
	}, {
		Name:    paramNameUser,
		Usage:   "postgres user",
		Default: "root",
	}, {
		Name:    paramNameDatabase,
		Usage:   "postgres database",
		Default: "postgres",
	}, {
		Name:    paramNameFilename,
		Usage:   "sql file path to execute",
		Default: "",
	}, {
		Name:    paramNameCommand,
		Usage:   "sql command to execute",
		Default: "",
	}}

var Entrypoint = common.NewToolEntrypoint(common.ToolEntrypointOption{
	ToolName:       "psql",
	Use:            "psql",
	Short:          "执行 PostgreSQL SQL 命令或脚本",
	CacheKeyPrefix: cacheKeyPrefix,
	ParamList:      paramSettingList,
	Run:            Run,
	InteractiveHook: func(values common.ParamMap) common.ParamMap {
		// 为了密码不存储缓存文件，而是通过[VAR=XXX pgo psql ...]方式传递
		password := putil.Interact.Input("Password (可空，空则沿用环境变量 PGPASSWORD): ")
		values[paramNamePassword] = password
		return values
	},
	CobraSetup: func(cmd *cobra.Command) func(values common.ParamMap) common.ParamMap {
		password := cmd.Flags().String(paramNamePassword, "", "postgres password (optional, fallback to env PGPASSWORD)")
		return func(values common.ParamMap) common.ParamMap {
			values[paramNamePassword] = *password
			return values
		}
	},
})

// --------------------------------------------------
// 运行参数，定义的参数列表最终转换成当前程序使用的运行选项

type RunOptions struct {
	Host     string
	PortStr  string
	User     string
	Password string
	Database string
	Filename string
	Command  string
}

// cobra参数值转换为“当前程序的”运行选项
func convParamToRunOpt(values common.ParamMap) RunOptions {
	return RunOptions{
		Host:     values[paramNameHost],
		PortStr:  values[paramNamePort],
		User:     values[paramNameUser],
		Password: values[paramNamePassword],
		Database: values[paramNameDatabase],
		Filename: values[paramNameFilename],
		Command:  values[paramNameCommand],
	}
}

// --------------------------------------------------
func Run(values common.ParamMap) error {
	options := convParamToRunOpt(values)

	if options.Host == "" {
		return errors.New("host is empty")
	}
	if options.User == "" {
		return errors.New("user is empty")
	}
	if options.Database == "" {
		return errors.New("database is empty")
	}

	port, err := putil.StrToInt32(options.PortStr)
	if err != nil {
		return fmt.Errorf("invalid port %q: %w", options.PortStr, err)
	}

	password := options.Password
	if password == "" {
		password = os.Getenv("PGPASSWORD")
	}

	// --------------------------------------------------
	err = pdb.InitPG(options.Host, options.User, password, options.Database, port)
	if err != nil {
		return fmt.Errorf("init postgres failed: %w", err)
	}

	if options.Command != "" {
		plogger.Debugf("sql cmd  [%s]", options.Command)
		_, err = pdb.Exec(options.Command)
		if err != nil {
			return fmt.Errorf("execute sql command failed: %w", err)
		}
		return nil
	}
	if options.Filename != "" {
		plogger.Debugf("sql file[%s]", options.Filename)
		err = pdb.ExecFile(options.Filename)
		if err != nil {
			return fmt.Errorf("execute sql file failed: %w", err)
		}
		return nil
	}

	return errors.New("either command or file must be provided")
}
