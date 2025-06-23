package db

import (
	"github.com/pancake-lee/pgo/internal/pkg/db/query"
	"github.com/pancake-lee/pgo/pkg/db"
)

func GetPG() *query.Query {
	return query.Use(db.GetGormDB())
}
