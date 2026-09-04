package pdb

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"sync"

	"gorm.io/gorm"
)

// --------------------------------------------------
// 按固定的配置结构，初始化一个默认的DB单例
var gDB *gorm.DB

// --------------------------------------------------
// 只读连接单例 + 懒初始化锁
var gReadOnlyDB *gorm.DB
var roMu sync.Mutex

// --------------------------------------------------
// DriverType 数据库驱动类型，Init* 时设置
type DriverType string

const (
	DriverSQLite   DriverType = "sqlite"
	DriverMySQL    DriverType = "mysql"
	DriverPostgres DriverType = "postgres"
)

// Init* 时保存，供只读连接懒初始化使用
var gDriverType DriverType
var gSavedDSN string      // MySQL / PostgreSQL 的 DSN
var gSavedFilePath string // SQLite 的文件路径
var gConf *SqlConfig

type SqlConfig struct {
	Addr     string
	User     string
	Password string
	DbName   string

	Host string // 从addr解析
	Port int32  // 从addr解析
}

// --------------------------------------------------
func GetSqlConfig() *SqlConfig {
	return gConf
}

func GetDB() (*sql.DB, error) {
	return gDB.DB()
}

// GetGormDB_RO 返回只读 *gorm.DB，首次调用时自动懒初始化。
// 初始化失败返回 nil，后续调用会重试。
func GetGormDB_RO() *gorm.DB {
	if gReadOnlyDB != nil {
		return gReadOnlyDB
	}
	roMu.Lock()
	defer roMu.Unlock()
	if gReadOnlyDB != nil {
		return gReadOnlyDB
	}
	if err := initReadOnlyDB(); err != nil {
		fmt.Printf("[pdb] init read only db failed: %v\n", err)
		return nil
	}
	return gReadOnlyDB
}

// GetDB_RO 返回只读 *sql.DB，首次调用时自动懒初始化。
// 初始化失败返回 error，后续调用会重试。
func GetDB_RO() (*sql.DB, error) {
	gormDB := GetGormDB_RO()
	if gormDB == nil {
		return nil, fmt.Errorf("read only db not initialized")
	}
	return gormDB.DB()
}

// --------------------------------------------------
func Exec(sql string) (sqlResult sql.Result, err error) {
	db, err := GetDB()
	if err != nil {
		return nil, err
	}
	sqlResult, err = db.Exec(sql)
	if err != nil {
		return nil, err
	}
	return sqlResult, nil
}

func ExecFile(path string) (err error) {
	sqlContent, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	// 将SQL文件内容拆分成单个语句
	statements := strings.Split(string(sqlContent), ";")
	for _, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement != "" {
			_, err = Exec(statement)
			if err != nil {
				return err
			}
		}
	}
	return nil
}
