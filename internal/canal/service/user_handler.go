package service

import (
	"context"

	"github.com/go-mysql-org/go-mysql/canal"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func init() {
	// 注册对 user 表的监控
	pdb.RegisterCallback("user", SyncUser)
}

// SyncUser 处理 user 表的变更事件
func SyncUser(ctx context.Context, e *canal.RowsEvent) error {
	plogger.Infof("SyncUser receive event: %s", e.Action)

	for _, row := range e.Rows {
		var user data.UserDO
		// 使用通用转换方法将行数据转换为 UserDO
		if err := pdb.MapRowToStruct(e.Table.Columns, row, &user); err != nil {
			plogger.Errorf("SyncUser convert error: %v", err)
			continue
		}

		plogger.Infof("SyncUser processed user: %+v", user)
	}

	return nil
}
