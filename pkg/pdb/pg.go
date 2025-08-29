package pdb

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

// 按固定的配置结构，初始化一个默认的DB单例

var gDB *gorm.DB

// 外部只读，只是方便外部获取当前连接的配置
var Host string
var Port int32
var User string
var Password string
var DbName string

// --------------------------------------------------
type pgConfig struct {
	Pg struct {
		Addr     string
		User     string
		Password string
		DbName   string
	}
}

func MustInitPGByConfig() {
	err := InitPGByConfig()
	if err != nil {
		panic(err)
	}
}

func InitPGByConfig() error {
	var conf pgConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		return err
	}
	plogger.Infof("load default pg with config: %+v", conf)

	strList := putil.StrToStrList(conf.Pg.Addr, ":")
	if len(strList) < 2 {
		return fmt.Errorf("invalid pg addr: %v", conf.Pg.Addr)
	}
	host, _port := strList[0], strList[1]
	port, err := putil.StrToInt32(_port)
	if err != nil {
		return fmt.Errorf("invalid pg port: %v, err: %v", _port, err)
	}

	return InitPG(host, conf.Pg.User, conf.Pg.Password, conf.Pg.DbName, port)
}
func InitPG(host, user, password, dbName string, port int32) (err error) {

	Host = host
	Port = port
	User = user
	Password = password
	DbName = dbName

	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v",
		host, user, password, dbName, port)
	dsn += " sslmode=disable TimeZone=Asia/Shanghai"

	gDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return err
	}
	return nil
}

// --------------------------------------------------
type myConfig struct {
	mysql struct {
		Addr     string
		User     string
		Password string
		DbName   string
	}
}

func MustInitMysqlByConfig() {
	err := InitMysqlByConfig()
	if err != nil {
		panic(err)
	}
}

func InitMysqlByConfig() error {
	var conf myConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		return err
	}
	plogger.Infof("load default mysql with config: %+v", conf)

	strList := putil.StrToStrList(conf.mysql.Addr, ":")
	if len(strList) < 2 {
		return fmt.Errorf("invalid mysql addr: %v", conf.mysql.Addr)
	}
	host, _port := strList[0], strList[1]
	port, err := putil.StrToInt32(_port)
	if err != nil {
		return fmt.Errorf("invalid mysql port: %v, err: %v", _port, err)
	}

	return InitMysql(host, conf.mysql.User, conf.mysql.Password, conf.mysql.DbName, port)
}
func InitMysql(host, user, password, dbName string, port int32) (err error) {

	Host = host
	Port = port
	User = user
	Password = password
	DbName = dbName

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbName)
	dsn += "?charset=utf8mb4&parseTime=True&loc=Local"

	gDB, err = gorm.Open(mysql.Open(dsn), &gorm.Config{
		Logger: dbLogger.New(
			Writer{},
			dbLogger.Config{
				SlowThreshold:             200 * time.Millisecond, // Slow SQL threshold
				LogLevel:                  dbLogger.Warn,          // Log level LogLevel 值为info打印sql
				IgnoreRecordNotFoundError: true,                   // Ignore ErrRecordNotFound error for logger
				Colorful:                  false,                  // Disable color
			},
		),
		SkipDefaultTransaction: true,
	})
	if err != nil {
		return err
	}
	return nil
}

type Writer struct{}

func (w Writer) Printf(format string, args ...any) {
	plogger.Infof(format, args...)
}

// --------------------------------------------------
func GetDB() (*sql.DB, error) {
	return gDB.DB()
}

func GetGormDB() *gorm.DB {
	return gDB
}

func GetTables() (ret []string, err error) {
	db, _ := GetDB()
	rows, err := db.Query(`show tables`)
	if err != nil {
		return ret, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return ret, err
		}
		ret = append(ret, tableName)
	}
	if err := rows.Err(); err != nil {
		return ret, err
	}
	return ret, err
}

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
