package db

import (
	"github.com/pancake-lee/pgo/internal/pkg/db/query"
	"github.com/pancake-lee/pgo/pkg/pdb"
)

func GetPG() *query.Query {
	return query.Use(pdb.GetGormDB())
}
