package predis

import (
	"context"
	"fmt"
	"github.com/go-redis/redis"
	"github.com/kataras/iris/v12/x/errors"
	"github.com/pancake-lee/pgo/pkg/pconfig"
	"github.com/pancake-lee/pgo/pkg/plogger"
)

// 按固定的配置结构，初始化一个默认的Redis客户端单例

const DefaultConfigGroup = "Redis"

var DefaultClient *redisClient

type redisClient struct {
	*redis.Client
	Addr     string
	DB       int
	Password string
}

type redisConfig struct {
	Redis struct {
		Addr     string
		DB       int    `default:"0"` // Redis database index
		Password string `default:""`  // Redis password
	}
}

func MustInitRedisByConfig() {
	err := InitRedisByConfig()
	if err != nil {
		panic(err)
	}
}

func InitRedisByConfig() error {
	var conf redisConfig
	err := pconfig.Scan(&conf)
	if err != nil {
		return err
	}
	plogger.Infof("load default redis with config: %+v", conf)

	cli := redis.NewClient(&redis.Options{
		Addr:     conf.Redis.Addr,
		Password: conf.Redis.Password,
		DB:       conf.Redis.DB,
	})

	if cli == nil {
		return errors.New("redis client init failed")
	}

	DefaultClient = &redisClient{
		Client:   cli,
		Addr:     conf.Redis.Addr,
		DB:       conf.Redis.DB,
		Password: conf.Redis.Password,
	}

	return nil
}

func CloseDefaultClient() {
	if DefaultClient != nil {
		DefaultClient.Close()
	}
}

// IsInitialized reports whether the default Redis client has been configured.
func IsInitialized() bool {
	return DefaultClient != nil && DefaultClient.Client != nil
}

// Ping verifies that the initialized default Redis client is reachable.
func Ping(ctx context.Context) error {
	if DefaultClient == nil || DefaultClient.Client == nil {
		return fmt.Errorf("redis client is not initialized")
	}
	return DefaultClient.Client.WithContext(ctx).Ping().Err()
}
