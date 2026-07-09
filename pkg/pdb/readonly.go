package pdb

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"time"

	glebarez "github.com/glebarez/sqlite"
	"github.com/go-sql-driver/mysql"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

// initReadOnlyDB 根据 gDriverType 分发到对应驱动的只读连接初始化。
// 调用方（GetGormDB_RO）已持有 roMu 锁。
func initReadOnlyDB() error {
	switch gDriverType {
	case DriverSQLite:
		return initSqliteReadOnly()
	case DriverMySQL:
		return initMysqlReadOnly()
	case DriverPostgres:
		return initPostgresReadOnly()
	default:
		return fmt.Errorf("unknown driver type: %s", gDriverType)
	}
}

func initSqliteReadOnly() error {
	dsn := "file:" + gSavedFilePath + "?mode=ro"
	var err error
	gReadOnlyDB, err = gorm.Open(glebarez.Open(dsn), &gorm.Config{
		Logger: dbLogger.New(
			Writer{},
			dbLogger.Config{
				SlowThreshold:             200 * time.Millisecond,
				LogLevel:                  dbLogger.Warn,
				IgnoreRecordNotFoundError: true,
				Colorful:                  false,
			},
		),
		SkipDefaultTransaction: true,
	})
	return err
}

func initPostgresReadOnly() error {
	dsn := gSavedDSN + " default_transaction_read_only=on"
	var err error
	gReadOnlyDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	return err
}

func initMysqlReadOnly() error {
	cfg, err := mysql.ParseDSN(gSavedDSN)
	if err != nil {
		return fmt.Errorf("parse mysql dsn: %w", err)
	}

	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return fmt.Errorf("create mysql connector: %w", err)
	}

	roSqlDB := sql.OpenDB(&roMySQLConnector{Connector: connector})

	gReadOnlyDB, err = gorm.Open(
		gormMysql.New(gormMysql.Config{
			Conn:                     roSqlDB,
			DisableDatetimePrecision: true,
		}),
		&gorm.Config{
			Logger: dbLogger.New(
				Writer{},
				dbLogger.Config{
					SlowThreshold:             200 * time.Millisecond,
					LogLevel:                  dbLogger.Warn,
					IgnoreRecordNotFoundError: true,
					Colorful:                  false,
				},
			),
			SkipDefaultTransaction: true,
		})
	if err != nil {
		roSqlDB.Close()
		return fmt.Errorf("open mysql ro gorm: %w", err)
	}
	return nil
}

// roMySQLConnector 包裹 mysql driver.Connector，在每次建立新连接时执行
// SET SESSION TRANSACTION READ ONLY，从 MySQL 引擎层面禁止写操作。
type roMySQLConnector struct {
	driver.Connector
}

func (c *roMySQLConnector) Connect(ctx context.Context) (driver.Conn, error) {
	conn, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	if execer, ok := conn.(driver.ExecerContext); ok {
		if _, err := execer.ExecContext(ctx, "SET SESSION TRANSACTION READ ONLY", nil); err != nil {
			conn.Close()
			return nil, fmt.Errorf("pdb: set session read only failed: %w", err)
		}
	}
	return conn, nil
}
