package pdb

import (
	"context"
	"database/sql"
)

type RawSql struct {
	db *sql.DB
}

// NewRawSql 创建原生SQL连接
func NewRawSql(driverName, dataSourceName string) (*RawSql, error) {
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &RawSql{db: db}, nil
}

func (r *RawSql) Close() error {
	return r.db.Close()
}

func (r *RawSql) Ping(ctx context.Context) error {
	return r.db.PingContext(ctx)
}

func (r *RawSql) Exec(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return r.db.ExecContext(ctx, query, args...)
}

func (r *RawSql) GetDB() *sql.DB {
	return r.db
}
