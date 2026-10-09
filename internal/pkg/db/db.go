package db

import (
	"github.com/pancake-lee/pgo/internal/pkg/db/query"
	"github.com/pancake-lee/pgo/pkg/pdb"
)

// InitQuery 在数据库初始化完成后绑定默认查询，须在处理请求之前调用。
func InitQuery() {
	query.SetDefault(pdb.GetGormDB())
}

// GetQuery 返回已初始化的默认查询，各请求通过 WithContext 创建查询链。
func GetQuery() *query.Query {
	return query.Q
}

// GetQueryTx 复用默认字段定义创建查询副本，绑定指定事务或回退到默认库。
func GetQueryTx(transactionID int32) *query.Query {
	return query.Q.ReplaceDB(pdb.GetGormDB(transactionID))
}
