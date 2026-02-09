package pdb

import (
	"database/sql"
	"os"
	"strings"

	"gorm.io/gorm"
)

// 按固定的配置结构，初始化一个默认的DB单例

var gDB *gorm.DB
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

func GetGormDB() *gorm.DB {
	return gDB
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
