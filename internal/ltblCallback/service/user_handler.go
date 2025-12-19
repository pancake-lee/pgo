package service

import (
	"context"

	"github.com/go-mysql-org/go-mysql/schema"
	"github.com/pancake-lee/pgo/internal/userService/data"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

func init() {
	// 注册对 user 表的监控
	pdb.RegisterInsertCallback("user", SyncUserInsert)
	pdb.RegisterUpdateCallback("user", SyncUserUpdate)
	pdb.RegisterDeleteCallback("user", SyncUserDelete)
}

// SyncUserInsert 处理 user 表的插入事件
func SyncUserInsert(ctx context.Context, columns []schema.TableColumn, row []interface{}) error {
	var user data.UserDO
	if err := pdb.MapRowToStruct(columns, row, &user); err != nil {
		plogger.Errorf("SyncUserInsert convert error: %v", err)
		return err
	}
	plogger.Infof("SyncUserInsert processed id[%v]", user.ID)
	return nil
}

// SyncUserUpdate 处理 user 表的更新事件
func SyncUserUpdate(ctx context.Context, columns []schema.TableColumn, oldRow, newRow []interface{}) error {
	var user data.UserDO
	if err := pdb.MapRowToStruct(columns, newRow, &user); err != nil {
		plogger.Errorf("SyncUserUpdate convert error: %v", err)
		return err
	}
	plogger.Infof("SyncUserUpdate processed id[%v]", user.ID)
	return nil
}

// SyncUserDelete 处理 user 表的删除事件
func SyncUserDelete(ctx context.Context, columns []schema.TableColumn, row []interface{}) error {
	var user data.UserDO
	if err := pdb.MapRowToStruct(columns, row, &user); err != nil {
		plogger.Errorf("SyncUserDelete convert error: %v", err)
		return err
	}
	plogger.Infof("SyncUserDelete processed id[%v]", user.ID)
	return nil
}
