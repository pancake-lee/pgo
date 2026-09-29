package papp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/pdb"
	"github.com/pancake-lee/pgo/pkg/plogger"
	"github.com/pancake-lee/pgo/pkg/pmq"
	"github.com/pancake-lee/pgo/pkg/predis"
	"github.com/pancake-lee/pgo/pkg/putil"
)

func CheckRabbitMQ() error {
	plogger.Info("Checking RabbitMQ...")
	err := checkDependencyWithRetry("RabbitMQ", func() error {
		if err := pmq.InitMQByConfig(); err != nil {
			return err
		}
		if pmq.DefaultClient == nil {
			return fmt.Errorf("RabbitMQ DefaultClient is nil")
		}
		return nil
	})
	if err != nil {
		return plogger.LogErr(err)
	}

	plogger.Info("RabbitMQ connected.")
	return nil
}

// --------------------------------------------------
func CheckRedis() error {
	plogger.Info("Checking Redis...")
	var pong string
	err := checkDependencyWithRetry("Redis", func() error {
		if err := predis.InitRedisByConfig(); err != nil {
			return fmt.Errorf("failed to init redis: %w", err)
		}
		if predis.DefaultClient == nil {
			return fmt.Errorf("redis client is nil")
		}
		var pingErr error
		pong, pingErr = predis.DefaultClient.Ping().Result()
		if pingErr != nil {
			predis.CloseDefaultClient()
			return fmt.Errorf("redis ping failed: %w", pingErr)
		}
		return nil
	})
	if err != nil {
		return plogger.LogErr(err)
	}
	plogger.Infof("Redis ping success: %s", pong)
	return nil
}

const (
	dependencyMaxAttempts   = 12
	dependencyRetryInterval = 5 * time.Second
)

func checkDependencyWithRetry(name string, check func() error) error {
	return checkDependencyWithRetryConfig(name, dependencyMaxAttempts, dependencyRetryInterval, check)
}

func checkDependencyWithRetryConfig(name string, maxAttempts int, interval time.Duration, check func() error) error {
	if maxAttempts <= 0 {
		return fmt.Errorf("%s check max attempts must be positive", name)
	}
	var attempts int
	err := NewRunner("check_"+strings.ToLower(name)).RunRetry(maxAttempts-1, interval, func() error {
		attempts++
		err := check()
		if err != nil && attempts < maxAttempts {
			plogger.Warnf("%s not ready, attempt %d/%d failed: %v", name, attempts, maxAttempts, err)
		}
		return err
	})
	if err != nil {
		return fmt.Errorf("%s unavailable after %d attempts: %w", name, attempts, err)
	}
	return nil
}

// --------------------------------------------------
func CheckMysql(models []any) error {
	plogger.Info("Checking MySQL...")
	_ = models

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

	if err := pdb.RunMysqlSchemaUpgrade(conf); err != nil {
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
