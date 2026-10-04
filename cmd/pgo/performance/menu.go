package performance

import (
	"github.com/pancake-lee/pgo/cmd/pgo/performance/login"
	"github.com/pancake-lee/pgo/cmd/pgo/performance/permissions"
	"github.com/pancake-lee/pgo/pkg/pclient"
)

// RegisterTools 登记登录与权限的性能测试和独立数据清理入口。
func RegisterTools(group *pclient.CommandGroup) {
	group.RegisterTool("登录性能测试", login.Entrypoint)
	group.RegisterTool("登录数据清理", login.Entrypoint.CleanupEntrypoint())
	group.RegisterTool("权限性能测试", permissions.Entrypoint)
	group.RegisterTool("权限数据清理", permissions.Entrypoint.CleanupEntrypoint())
}
