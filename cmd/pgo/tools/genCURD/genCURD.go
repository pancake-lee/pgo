package genCURD

import (
	"errors"
	"os"

	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/putil"
)

// --------------------------------------------------
const (
	paramNameDB      = "db"
	paramNameDSN     = "dsn"
	paramNameWorkDir = "workDir"
	cacheKeyPrefix   = "tools.genCURD."
)

var paramSettingList = []pclient.ParamItem{
	{
		Name:    paramNameDB,
		Usage:   "database type, support mysql and sqlite3",
		Default: "mysql",
	}, {
		Name:    paramNameDSN,
		Usage:   "database connection string (mysql: user:pass@tcp(host:port)/db, sqlite3: /path/to/db.sqlite)",
		Default: "",
	}, {
		Name:    paramNameWorkDir,
		Usage:   "project root directory (contains ./tools/genCURD)",
		Default: putil.GetCurDir(),
	}}

var Entrypoint = pclient.NewTool(pclient.ToolOption{
	ToolName:       "genCURD",
	Use:            "curd",
	Aliases:        []string{"genCURD"},
	Short:          "基于数据库生成 CURD 代码",
	CacheKeyPrefix: cacheKeyPrefix,
	ParamList:      paramSettingList,
	Run:            Run,
})

// --------------------------------------------------
// 运行参数，定义的参数列表最终转换成当前程序使用的运行选项

type RunOptions struct {
	DB      string
	DSN     string
	WorkDir string
}

// cobra参数值转换为"当前程序的"运行选项
func convParamToRunOpt(values pclient.ParamMap) RunOptions {
	workDir := values[paramNameWorkDir]
	if workDir == "" {
		workDir = putil.GetCurDir()
	}

	return RunOptions{
		DB:      values[paramNameDB],
		DSN:     values[paramNameDSN],
		WorkDir: workDir,
	}
}

// --------------------------------------------------
func Run(values pclient.ParamMap) error {
	options := convParamToRunOpt(values)

	if options.DB == "" {
		options.DB = "mysql"
	}
	if options.DSN == "" {
		return errors.New("dsn is empty")
	}
	if options.WorkDir == "" {
		return errors.New("workDir is empty")
	}

	oldWorkDir, err := os.Getwd()
	if err != nil {
		return err
	}

	err = os.Chdir(options.WorkDir)
	if err != nil {
		return err
	}
	defer os.Chdir(oldWorkDir)

	return runGenerate(options.DB, options.DSN)
}
