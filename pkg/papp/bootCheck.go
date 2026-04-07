package papp

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/predis"
	"github.com/pancake-lee/pgo/pkg/putil"
	_ "gorm.io/driver/mysql"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

func CheckRabbitMQ() error {
	plogger.Info("Checking RabbitMQ...")
	if err := pmq.InitMQByConfig(); err != nil {
		// Treat as warning/skip if config is missing or invalid,
		// assuming not all services need RabbitMQ.
		// If it's critical, user should see the log.
		return plogger.LogErr(err)
	}

	if pmq.DefaultClient == nil {
		return plogger.LogErr(fmt.Errorf("RabbitMQ DefaultClient is nil"))
	}

	plogger.Info("RabbitMQ connected.")
	return nil
}

// --------------------------------------------------
func CheckRedis() error {
	plogger.Info("Checking Redis...")

	if err := predis.InitRedisByConfig(); err != nil {
		return plogger.LogErr(fmt.Errorf("failed to init redis: %v", err))
	}
	if predis.DefaultClient == nil {
		return plogger.LogErr(fmt.Errorf("redis client is nil"))
	}

	pong, err := predis.DefaultClient.Ping().Result()
	if err != nil {
		return plogger.LogErr(fmt.Errorf("redis ping failed: %v", err))
	}
	plogger.Infof("Redis ping success: %s", pong)
	return nil
}

// --------------------------------------------------
func CheckMysql(models []any) error {
	plogger.Info("Checking MySQL...")

	var conf pdb.MysqlConfig
	if err := pconfig.Scan(&conf); err != nil {
		return plogger.LogErr(err)
	}
	if conf.Mysql.Addr == "" {
		return plogger.LogErr(fmt.Errorf("mysql addr is empty"))
	}

	parts := strings.Split(conf.Mysql.Addr, ":")
	if len(parts) != 2 {
		return plogger.LogErr(fmt.Errorf("invalid mysql addr: %s", conf.Mysql.Addr))
	}
	host := parts[0]
	portStr := parts[1]

	rawDB, err := openMysqlWithoutDB(conf, host, portStr)
	if err != nil {
		return err
	}
	defer rawDB.Close()

	if err := waitMysqlReady(rawDB); err != nil {
		return err
	}

	if err := ensureDatabaseExists(rawDB, conf); err != nil {
		return err
	}

	if err := initMysqlGorm(conf, portStr); err != nil {
		return err
	}

	if err := runAutoMigrate(pdb.GetGormDB(), models); err != nil {
		return err
	}
	return nil
}

func openMysqlWithoutDB(conf pdb.MysqlConfig, host, portStr string) (*pdb.RawSql, error) {
	dsnNoDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/", conf.Mysql.User, conf.Mysql.Password, host, portStr)
	rawDB, err := pdb.NewRawSql("mysql", dsnNoDB)
	if err != nil {
		return nil, plogger.LogErr(fmt.Errorf("failed to open mysql connection: %v", err))
	}
	return rawDB, nil
}

func waitMysqlReady(rawDB *pdb.RawSql) error {
	err := NewRunner("check_mysql").RunRetry(12, 15*time.Second, func() error {
		ctx, cancel := context.WithTimeout(context.Background(), pingTimeout)
		defer cancel()
		if err := rawDB.Ping(ctx); err != nil {
			plogger.Infof("Failed to ping mysql: %v, retrying...", err)
			return err
		}
		return nil
	})
	if err != nil {
		return plogger.LogErr(fmt.Errorf("failed to connect to mysql after retries: %v", err))
	}

	plogger.Info("Mysql connection established.")
	return nil
}

const pingTimeout = 3 * time.Second

func ensureDatabaseExists(rawDB *pdb.RawSql, conf pdb.MysqlConfig) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci", conf.Mysql.DbName)
	if _, err := rawDB.Exec(ctx, query); err != nil {
		return plogger.LogErr(fmt.Errorf("failed to create database %s: %v", conf.Mysql.DbName, err))
	}
	plogger.Infof("Database %s checked/created.", conf.Mysql.DbName)
	return nil
}

func initMysqlGorm(conf pdb.MysqlConfig, portStr string) error {
	parts := strings.Split(conf.Mysql.Addr, ":")
	host := parts[0]
	port, err := putil.StrToInt32(portStr)
	if err != nil {
		return plogger.LogErr(fmt.Errorf("invalid mysql port: %v", err))
	}
	if err := pdb.InitMysql(host, conf.Mysql.User, conf.Mysql.Password, conf.Mysql.DbName, port); err != nil {
		return plogger.LogErr(fmt.Errorf("pdb.InitMysql failed: %v", err))
	}
	return nil
}
func runAutoMigrate(db *gorm.DB, models []any) error {
	plogger.Info("Starting AutoMigrate...")

	plannedSQL, err := collectMigrationSQL(db, models, true)
	if err != nil {
		return plogger.LogErr(fmt.Errorf("failed to preview migration sql: %v", err))
	}
	if err := validateSafeMigrationSQL(plannedSQL); err != nil {
		plogger.Warnf("Skip AutoMigrate due to unsafe SQL plan: %v", err)
		plogger.Warn("MySQL connectivity and DB existence are checked; schema changes require manual SQL/generator alignment.")
		return nil
	}

	originalLogger := db.Logger
	sb := &strings.Builder{}
	defer func() { db.Logger = originalLogger }()

	baseLogger := originalLogger.LogMode(dbLogger.Info)
	db.Logger = &migrationSQLLogger{
		Interface: baseLogger,
		sb:        sb,
	}

	err = db.AutoMigrate(models...)
	persistMigrationSQL(sb)
	if err != nil {
		return plogger.LogErr(fmt.Errorf("auto migrate failed: %v", err))
	}
	plogger.Info("AutoMigrate finished.")
	return nil
}

func persistMigrationSQL(sb *strings.Builder) {
	if sb.Len() == 0 {
		return
	}

	folder := fmt.Sprintf("%v/record", putil.GetExecFolder())

	if err := os.MkdirAll(folder, 0755); err != nil {
		plogger.Errorf("Failed to create record folder: %v", err)
		return
	}

	filename := fmt.Sprintf("%v/db_update_%s.sql", folder, time.Now().Format("20060102T150405"))
	if err := os.WriteFile(filename, []byte(sb.String()), 0644); err != nil {
		plogger.Errorf("Failed to write SQL log: %v", err)
		return
	}
	plogger.Infof("SQL log written to %s", filename)
}

type migrationSQLLogger struct {
	dbLogger.Interface
	sb *strings.Builder
}

func (l *migrationSQLLogger) LogMode(level dbLogger.LogLevel) dbLogger.Interface {
	return &migrationSQLLogger{
		Interface: l.Interface.LogMode(level),
		sb:        l.sb,
	}
}

func (l *migrationSQLLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.Interface.Trace(ctx, begin, fc, err)

	sql, _ := fc()
	if sql == "" {
		return
	}

	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "CREATE") ||
		strings.HasPrefix(upper, "ALTER") ||
		strings.HasPrefix(upper, "DROP") ||
		strings.HasPrefix(upper, "RENAME") ||
		strings.HasPrefix(upper, "TRUNCATE") {
		l.sb.WriteString(sql + ";\n")
	}
}

func collectMigrationSQL(db *gorm.DB, models []any, dryRun bool) ([]string, error) {
	originalLogger := db.Logger
	statements := make([]string, 0)
	defer func() { db.Logger = originalLogger }()

	baseLogger := originalLogger.LogMode(dbLogger.Info)
	db.Logger = &migrationSQLCollectorLogger{
		Interface:  baseLogger,
		statements: &statements,
	}

	migrationDB := db
	if dryRun {
		migrationDB = db.Session(&gorm.Session{DryRun: true})
	}

	if err := migrationDB.AutoMigrate(models...); err != nil {
		return nil, err
	}

	return statements, nil
}

func validateSafeMigrationSQL(sqlList []string) error {
	for _, raw := range sqlList {
		stmt := strings.ToUpper(strings.TrimSpace(raw))
		if stmt == "" {
			continue
		}

		if strings.HasPrefix(stmt, "DROP ") || strings.HasPrefix(stmt, "TRUNCATE ") {
			return fmt.Errorf("detected destructive migration SQL, please let ops handle manually: %s", raw)
		}

		if strings.HasPrefix(stmt, "ALTER ") {
			if !isSafeAlterAddOnly(stmt) {
				return fmt.Errorf("detected non-additive ALTER SQL, please let ops handle manually: %s", raw)
			}
		}
	}
	return nil
}

func isSafeAlterAddOnly(stmt string) bool {
	if !strings.Contains(stmt, " ADD ") {
		return false
	}

	for _, keyword := range []string{
		" DROP ",
		" MODIFY ",
		" CHANGE ",
		" RENAME ",
		" ALTER COLUMN ",
		" DROP COLUMN ",
		" DROP INDEX ",
		" DROP PRIMARY KEY ",
	} {
		if strings.Contains(stmt, keyword) {
			return false
		}
	}

	return true
}

type migrationSQLCollectorLogger struct {
	dbLogger.Interface
	statements *[]string
}

func (l *migrationSQLCollectorLogger) LogMode(level dbLogger.LogLevel) dbLogger.Interface {
	return &migrationSQLCollectorLogger{
		Interface:  l.Interface.LogMode(level),
		statements: l.statements,
	}
}

func (l *migrationSQLCollectorLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	l.Interface.Trace(ctx, begin, fc, err)

	sql, _ := fc()
	if sql == "" {
		return
	}

	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "CREATE") ||
		strings.HasPrefix(upper, "ALTER") ||
		strings.HasPrefix(upper, "DROP") ||
		strings.HasPrefix(upper, "RENAME") ||
		strings.HasPrefix(upper, "TRUNCATE") {
		*l.statements = append(*l.statements, sql)
	}
}
