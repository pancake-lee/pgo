package genGORM

import (
	"errors"
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pclient"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"gorm.io/gen"
	"gorm.io/gorm"
)

// --------------------------------------------------
const (
	paramNameDB           = "db"
	paramNameDSN          = "dsn"
	paramNameOutPath      = "outPath"
	paramNameOutFile      = "outFile"
	paramNameModelPkgName = "modelPkgName"
	cacheKeyPrefix        = "tools.genGORM."
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
		Name:    paramNameOutPath,
		Usage:   "output directory for generated orm query code",
		Default: "./internal/pkg/db/query/",
	}, {
		Name:    paramNameOutFile,
		Usage:   "output filename for generated orm query code",
		Default: "query.go",
	}, {
		Name:    paramNameModelPkgName,
		Usage:   "model package name imported by generated query code",
		Default: "model",
	}}

var Entrypoint = pclient.NewTool(pclient.ToolOption{
	ToolName:       "genGORM",
	Use:            "gorm",
	Aliases:        []string{"genGORM"},
	Short:          "基于数据库结构生成 GORM Query 代码",
	CacheKeyPrefix: cacheKeyPrefix,
	ParamList:      paramSettingList,
	Run:            Run,
})

// --------------------------------------------------
// 运行参数，定义的参数列表最终转换成当前程序使用的运行选项

type RunOptions struct {
	DB           string
	DSN          string
	OutPath      string
	OutFile      string
	ModelPkgName string
}

// cobra参数值转换为“当前程序的”运行选项
func convParamToRunOpt(values pclient.ParamMap) RunOptions {
	return RunOptions{
		DB:           values[paramNameDB],
		DSN:          values[paramNameDSN],
		OutPath:      values[paramNameOutPath],
		OutFile:      values[paramNameOutFile],
		ModelPkgName: values[paramNameModelPkgName],
	}
}

// --------------------------------------------------
func Run(values pclient.ParamMap) error {
	options := convParamToRunOpt(values)

	if options.DB == "" {
		options.DB = "mysql"
	}
	switch options.DB {
	case "mysql":
		if options.DSN == "" {
			return errors.New("dsn is empty")
		}
		err := pdb.InitMysqlByDsn(options.DSN)
		if err != nil {
			return fmt.Errorf("init mysql by dsn failed: %w", err)
		}
	case "sqlite3":
		if options.DSN == "" {
			return errors.New("dsn (sqlite file path) is empty")
		}
		err := pdb.InitSqlite(options.DSN)
		if err != nil {
			return fmt.Errorf("init sqlite failed: %w", err)
		}
	default:
		return fmt.Errorf("db %q is not supported, only mysql and sqlite3 are supported", options.DB)
	}
	if options.OutFile == "" {
		return errors.New("outFile is empty")
	}
	if options.ModelPkgName == "" {
		return errors.New("modelPkgName is empty")
	}

	// 初始化 GenTool
	g := gen.NewGenerator(gen.Config{
		OutPath:      options.OutPath,
		OutFile:      options.OutFile,
		ModelPkgPath: options.ModelPkgName,
	})
	g.WithImportPkgPath("github.com/shopspring/decimal")
	g.UseDB(pdb.GetGormDB())

	// 3. 核心：自定义类型映射（DECIMAL → decimal.Decimal）
	dataTypeMap := map[string]func(columnType gorm.ColumnType) (dataType string){
		"decimal": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal"
		},
	}
	g.WithDataTypeMap(dataTypeMap)

	// 4. 生成
	g.ApplyBasic(g.GenerateAllTable()...)
	g.Execute()
	return nil
}
