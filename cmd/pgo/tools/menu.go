package tools

import (
	"github.com/pancake-lee/pgo/cmd/pgo/tools/diagnostics"
	"github.com/pancake-lee/pgo/cmd/pgo/tools/genCURD"
	"github.com/pancake-lee/pgo/cmd/pgo/tools/genGORM"
	"github.com/pancake-lee/pgo/cmd/pgo/tools/interaction"
	"github.com/pancake-lee/pgo/cmd/pgo/tools/prettyCode"
	"github.com/pancake-lee/pgo/cmd/pgo/tools/psql"
	"github.com/pancake-lee/pgo/cmd/pgo/tools/sheet2mysql"
	"github.com/pancake-lee/pgo/pkg/pclient"
)

// RegisterTools registers concrete development and diagnostic tools.
func RegisterTools(group *pclient.CommandGroup) {
	group.RegisterTool("美化代码 (Pretty Code)", prettyCode.Entrypoint)
	group.RegisterTool("执行 PostgreSQL", psql.Entrypoint)
	group.RegisterTool("生成 CURD 代码", genCURD.Entrypoint)
	group.RegisterTool("生成 GORM 代码", genGORM.Entrypoint)
	group.RegisterTool("多维表格转 MySQL 建表 SQL", sheet2mysql.Entrypoint)
	group.RegisterTool("诊断健康状态", diagnostics.HealthEntrypoint)
	group.RegisterTool("诊断指标地址", diagnostics.MetricsURLEntrypoint)
	group.RegisterTool("下载运行时 Profile", diagnostics.ProfileEntrypoint)
	group.RegisterTool("测试交互组件", interaction.Entrypoint)
}
