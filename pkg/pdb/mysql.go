package pdb

import (
	"fmt"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

type MysqlConfig struct {
	Mysql SqlConfig
}

func MustInitMysqlByConfig() {
	err := InitMysqlByConfig()
	if err != nil {
		panic(err)
	}
}

func InitMysqlByConfig() error {
	var conf MysqlConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		return err
	}
	plogger.Infof("load default mysql with config: %+v", conf)

	strList := putil.StrToStrList(conf.Mysql.Addr, ":")
	if len(strList) < 2 {
		return fmt.Errorf("invalid mysql addr: %v", conf.Mysql.Addr)
	}
	host, _port := strList[0], strList[1]
	port, err := putil.StrToInt32(_port)
	if err != nil {
		return fmt.Errorf("invalid mysql port: %v, err: %v", _port, err)
	}

	return InitMysql(host, conf.Mysql.User, conf.Mysql.Password, conf.Mysql.DbName, port)
}
func InitMysql(host, user, password, dbName string, port int32) (err error) {

	gConf = &SqlConfig{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		User:     user,
		Password: password,
		DbName:   dbName,
		Host:     host,
		Port:     port,
	}

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
