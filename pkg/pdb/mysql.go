package pdb

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	gormMysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	dbLogger "gorm.io/gorm/logger"
)

// defaultMysqlMaxOpenConns 限制每个 MySQL 连接池的最大连接数。
const defaultMysqlMaxOpenConns = 64

// defaultMysqlMaxIdleConns 指定每个 MySQL 连接池保留的空闲连接数。一般和Open一致，至少是一半。
const defaultMysqlMaxIdleConns = 64

const DefaultConfigGroup = "Mysql"

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

	return InitMysqlWithConfig(conf.Mysql)
}

// InitMysqlWithConfig 使用连接配置和可选连接池参数初始化 MySQL。
func InitMysqlWithConfig(conf SqlConfig) error {
	strList := putil.StrToStrList(conf.Addr, ":")
	if len(strList) < 2 {
		return fmt.Errorf("invalid mysql addr: %v", conf.Addr)
	}
	host, _port := strList[0], strList[1]
	port, err := putil.StrToInt32(_port)
	if err != nil {
		return fmt.Errorf("invalid mysql port: %v, err: %v", _port, err)
	}

	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s",
		conf.User, conf.Password, host, port, conf.DbName)
	dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
	return initMysqlByDsn(dsn, conf)
}
func InitMysql(host, user, password, dbName string, port int32) (err error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s", user, password, host, port, dbName)
	dsn += "?charset=utf8mb4&parseTime=True&loc=Local"
	return InitMysqlByDsn(dsn)
}

// InitMysqlByDsn 使用 DSN 与默认连接池参数初始化 MySQL。
func InitMysqlByDsn(dsn string) error {
	return initMysqlByDsn(dsn, SqlConfig{})
}

// initMysqlByDsn 校验参数并将连接池设置应用于共享 MySQL 实例。
func initMysqlByDsn(dsn string, conf SqlConfig) (err error) {
	conf, err = resolveMysqlPoolConfig(conf)
	if err != nil {
		return err
	}
	cfg, err := mysql.ParseDSN(dsn)
	if err != nil {
		return err
	}
	host, port, _ := strings.Cut(cfg.Addr, ":")
	p, _ := putil.StrToInt32(port)

	gDriverType = DriverMySQL
	gSavedDSN = dsn

	gConf = &SqlConfig{
		Addr:         cfg.Addr,
		User:         cfg.User,
		Password:     cfg.Passwd,
		DbName:       cfg.DBName,
		Host:         host,
		Port:         p,
		MaxOpenConns: conf.MaxOpenConns,
		MaxIdleConns: conf.MaxIdleConns,
	}

	gDB, err = gorm.Open(
		gormMysql.New(gormMysql.Config{
			DSN:                      dsn,
			DisableDatetimePrecision: true, // false 时 AutoMigrate 会失败
		}),
		&gorm.Config{
			Logger: newGormLogger(
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
	sqlDB, err := gDB.DB()
	if err != nil {
		return err
	}
	setMysqlPool(sqlDB, conf)
	return registerQueryObservability(gDB)
}

type Writer struct{}

func (w Writer) Printf(format string, args ...any) {
	plogger.Infof(format, args...)
}

// resolveMysqlPoolConfig 补全默认值并在创建连接前校验连接池参数。
func resolveMysqlPoolConfig(conf SqlConfig) (SqlConfig, error) {
	if conf.MaxOpenConns == nil {
		value := defaultMysqlMaxOpenConns
		conf.MaxOpenConns = &value
	}
	if *conf.MaxOpenConns <= 0 {
		return conf, fmt.Errorf("Mysql.MaxOpenConns must be greater than 0")
	}
	if conf.MaxIdleConns == nil {
		value := min(defaultMysqlMaxIdleConns, *conf.MaxOpenConns)
		conf.MaxIdleConns = &value
	}
	if *conf.MaxIdleConns < 0 || *conf.MaxIdleConns > *conf.MaxOpenConns {
		return conf, fmt.Errorf("Mysql.MaxIdleConns must be between 0 and MaxOpenConns")
	}
	return conf, nil
}

// setMysqlPool 将已校验的最大连接与空闲连接数量应用于 SQL 连接池。
func setMysqlPool(db *sql.DB, conf SqlConfig) {
	db.SetMaxOpenConns(*conf.MaxOpenConns)
	db.SetMaxIdleConns(*conf.MaxIdleConns)
}
