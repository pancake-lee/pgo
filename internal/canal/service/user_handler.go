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
	for _, row := range e.Rows {
		var user data.UserDO
		if err := pdb.MapRowToStruct(e.Table.Columns, row, &user); err != nil {
			plogger.Errorf("SyncUser convert error: %v", err)
			continue
		}
		plogger.Infof("SyncUser processed event[%s] id[%v]", e.Action, user.ID)
		// plogger.Infof("SyncUser processed event[%s] user: %+v", e.Action, user)
	}

	return nil
}
