package db

import (
	"pgo/internal/pkg/db/query"
	"pgo/pkg/db"
)

func GetPG() *query.Query {
	return query.Use(db.GetGormDB())
}
