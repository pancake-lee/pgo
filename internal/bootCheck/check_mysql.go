package main

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/internal/pkg/db/model"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	_ "gorm.io/driver/mysql" // for sql.Open("mysql", ...)
	dbLogger "gorm.io/gorm/logger"
)

func checkMysql() {
	plogger.Info("Checking MySQL...")

	// 1. Load Config
	var conf pdb.MysqlConfig
	if err := pconfig.Scan(&conf); err != nil {
		plogger.Fatalf("Failed to scan mysql config: %v", err)
	}

	if conf.Mysql.Addr == "" {
		plogger.Fatal("Mysql Addr is empty")
	}

	// Parse Host/Port
	parts := strings.Split(conf.Mysql.Addr, ":")
	if len(parts) != 2 {
		plogger.Fatalf("Invalid mysql addr: %s", conf.Mysql.Addr)
	}
	host := parts[0]
	portStr := parts[1]

	// 2. Connectivity Test & Ensure DB Exists
	ensureMysqlDBExists(conf, host, portStr)

	// 3. Schema Update
	updateMysqlSchema(conf, host, portStr)
}

func ensureMysqlDBExists(conf pdb.MysqlConfig, host, portStr string) {
	// Connect without DB name to check/create DB
	dsnNoDB := fmt.Sprintf("%s:%s@tcp(%s:%s)/", conf.Mysql.User, conf.Mysql.Password, host, portStr)

	rawDB, err := pdb.NewRawSql("mysql", dsnNoDB)
	if err != nil {
		plogger.Fatalf("Failed to open mysql connection: %v", err)
	}
	defer rawDB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rawDB.Ping(ctx); err != nil {
		plogger.Fatalf("Failed to ping mysql: %v", err)
	}

	plogger.Info("Mysql connection established.")

	// Check DB exists
	query := fmt.Sprintf("CREATE DATABASE IF NOT EXISTS `%s` DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci", conf.Mysql.DbName)
	if _, err := rawDB.Exec(ctx, query); err != nil {
		plogger.Fatalf("Failed to create database %s: %v", conf.Mysql.DbName, err)
	}
	plogger.Infof("Database %s checked/created.", conf.Mysql.DbName)
}

func updateMysqlSchema(conf pdb.MysqlConfig, host, portStr string) {
	// Init pdb (Global GORM instance)
	portInt, err := putil.StrToInt32(portStr)
	if err != nil {
		plogger.Fatalf("Invalid port: %v", err)
	}

	err = pdb.InitMysql(host, conf.Mysql.User, conf.Mysql.Password,
		conf.Mysql.DbName, portInt)
	if err != nil {
		plogger.Fatalf("pdb.InitMysql failed: %v", err)
	}

	db := pdb.GetGormDB()

	// Capture SQLs
	sb := &strings.Builder{}

	// Preserve original logger
	originalLogger := db.Logger
	defer func() { db.Logger = originalLogger }()

	// Use Info level base logger to ensure logs go to plogger (as configured in pdb)
	baseLogger := originalLogger.LogMode(dbLogger.Info)

	// Wrap with MigrationLogger to capture SQL commands
	db.Logger = &MigrationLogger{
		Interface: baseLogger,
		sb:        sb,
	}

	// AutoMigrate
	models := getAllModels()

	plogger.Info("Starting AutoMigrate...")
	err = db.AutoMigrate(models...)

	// Save SQL log regardless of success or failure
	if sb.Len() > 0 {
		folder := fmt.Sprintf("%v/record", putil.GetExecFolder())
		err = os.MkdirAll(folder, 0755)
		if err != nil {
			plogger.Errorf("Failed to create record folder: %v", err)
		} else {
			filename := fmt.Sprintf("%v/db_update_%s.sql",
				folder, time.Now().Format("20060102T150405"))
			err = os.WriteFile(filename, []byte(sb.String()), 0644)
			if err != nil {
				plogger.Errorf("Failed to write SQL log: %v", err)
			} else {
				plogger.Infof("SQL log written to %s", filename)
			}
		}
	}

	if err != nil {
		plogger.Fatalf("AutoMigrate failed: %v", err)
	}
	plogger.Info("AutoMigrate finished.")
}

// MigrationLogger wraps a GORM logger to capture SQL statements
type MigrationLogger struct {
	dbLogger.Interface
	sb *strings.Builder
}

func (l *MigrationLogger) LogMode(level dbLogger.LogLevel) dbLogger.Interface {
	return &MigrationLogger{
		Interface: l.Interface.LogMode(level),
		sb:        l.sb,
	}
}

func (l *MigrationLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	// 1. Log to standard output (plogger)
	l.Interface.Trace(ctx, begin, fc, err)

	// 2. Capture SQL for file
	sql, _ := fc()
	if sql == "" {
		return
	}

	// Filter: Only capture schema modification commands (DDL)
	// We trim space and check prefix to avoid SELECTs from being recorded in the update file
	upper := strings.ToUpper(strings.TrimSpace(sql))
	if strings.HasPrefix(upper, "CREATE") ||
		strings.HasPrefix(upper, "ALTER") ||
		strings.HasPrefix(upper, "DROP") ||
		strings.HasPrefix(upper, "RENAME") ||
		strings.HasPrefix(upper, "TRUNCATE") {
		l.sb.WriteString(sql + ";\n")
	}
}

func getAllModels() []interface{} {
	return []interface{}{
		&model.AbandonCode{},
		&model.CourseSwapRequest{},
		&model.Project{},
		&model.Task{},
		&model.User{},
		&model.UserDept{},
		&model.UserDeptAssoc{},
		&model.UserJob{},
		&model.UserProjectAssoc{},
		&model.UserRole{},
		&model.UserRoleAssoc{},
		&model.UserRolePermissionAssoc{},
	}
}
