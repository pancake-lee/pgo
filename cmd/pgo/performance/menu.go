package performance

import (
	"github.com/pancake-lee/pgo/cmd/pgo/performance/login"
	"github.com/pancake-lee/pgo/cmd/pgo/performance/permissions"
	"github.com/pancake-lee/pgo/pkg/pclient"
)

// RegisterTools registers the concrete performance tools.
func RegisterTools(group *pclient.CommandGroup) {
	group.RegisterTool("用户登录压测 (User Login)", login.Entrypoint)
	group.RegisterTool("共享角色权限读写压测", permissions.Entrypoint)
}
