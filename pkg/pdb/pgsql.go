package pdb

import (
	"fmt"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/putil"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PgsqlConfig struct {
	Pgsql SqlConfig
}

func MustInitPGByConfig() {
	err := InitPGByConfig()
	if err != nil {
		panic(err)
	}
}

func InitPGByConfig() error {
	var conf PgsqlConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		return err
	}
	plogger.Infof("load default pg with config: %+v", conf)

	strList := putil.StrToStrList(conf.Pgsql.Addr, ":")
	if len(strList) < 2 {
		return fmt.Errorf("invalid pg addr: %v", conf.Pgsql.Addr)
	}
	host, _port := strList[0], strList[1]
	port, err := putil.StrToInt32(_port)
	if err != nil {
		return fmt.Errorf("invalid pg port: %v, err: %v", _port, err)
	}

	return InitPG(host, conf.Pgsql.User, conf.Pgsql.Password, conf.Pgsql.DbName, port)
}
func InitPG(host, user, password, dbName string, port int32) (err error) {

	gConf = &SqlConfig{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		User:     user,
		Password: password,
		DbName:   dbName,
		Host:     host,
		Port:     port,
	}

	dsn := fmt.Sprintf("host=%v user=%v password=%v dbname=%v port=%v",
		host, user, password, dbName, port)
	dsn += " sslmode=disable TimeZone=Asia/Shanghai"

	gDB, err = gorm.Open(postgres.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return err
	}
	return nil
}
