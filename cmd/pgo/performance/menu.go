package performance

import (
	"github.com/pancake-lee/pgo/cmd/pgo/performance/login"
	"github.com/pancake-lee/pgo/cmd/pgo/performance/permissions"
	"github.com/pancake-lee/pgo/pkg/pclient"
)

// RegisterTools 登记登录与权限合并性能场景。
func RegisterTools(group *pclient.CommandGroup) {
	group.RegisterTool("用户登录压测 (User Login)", login.Entrypoint)
	group.RegisterTool("权限合并压测 (Permissions)", permissions.Entrypoint)
}
