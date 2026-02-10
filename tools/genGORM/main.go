package main

import (
	"flag"

	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"gorm.io/gen"
	"gorm.io/gorm"
)

func main() {
	l := flag.Bool("l", false, "log to console, default is false")

	// 暂只支持mysql，只是适配gentool的参数而已
	flag.String("db", "mysql", "数据库类型，默认为 mysql")

	dsn := flag.String("dsn", "", "数据库连接字符串")
	outPath := flag.String("outPath", "./internal/pkg/db/query/", "生成的ORM代码输出路径")
	outFile := flag.String("outFile", "query.go", "生成的ORM代码文件名")
	modelPkgName := flag.String("modelPkgName", "model", "生成的ORM代码中引用的模型包名")
	flag.Parse()

	// 1. 连接数据库
	plogger.InitFromConfig(*l)
	pdb.InitMysqlByDsn(*dsn)

	// 2. 初始化 GenTool
	g := gen.NewGenerator(gen.Config{
		OutPath:      *outPath,
		OutFile:      *outFile,
		ModelPkgPath: *modelPkgName,
		// Mode:         gen.WithDefaultQuery,
	})
	g.WithImportPkgPath("github.com/shopspring/decimal")
	g.UseDB(pdb.GetGormDB()) // 关联数据库

	// 3. 核心：自定义类型映射（DECIMAL → decimal.Decimal）
	// 覆盖默认的 DECIMAL → float64 映射
	dataTypeMap := map[string]func(columnType gorm.ColumnType) (dataType string){
		// 匹配数据库 DECIMAL 类型（包括 DECIMAL(15,2)、DECIMAL(10,2) 等）
		"decimal": func(columnType gorm.ColumnType) (dataType string) {
			return "decimal.Decimal"
		},
	}
	g.WithDataTypeMap(dataTypeMap)

	// 4. 生成
	g.ApplyBasic(g.GenerateAllTable()...)
	g.Execute()
}
