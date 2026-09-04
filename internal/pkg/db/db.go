package db

import (
	"github.com/pancake-lee/pgo/internal/pkg/db/query"
	"github.com/pancake-lee/pgo/pkg/pdb"
)

// 可以考虑生成gorm时加入gen.WithDefaultQuery，生成SetDefault就不用每次都Use
func GetQuery() *query.Query {
	return query.Use(pdb.GetGormDB())
}

// GetQueryTx returns the query bound to transactionID's active transaction.
// Existing callers keep using GetQuery; transaction-aware workflows opt in.
func GetQueryTx(transactionID int32) *query.Query {
	return query.Use(pdb.GetGormDB(transactionID))
}
