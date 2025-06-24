package predis

import (
	"github.com/go-redis/redis"
	"github.com/kataras/iris/v12/x/errors"
	"github.com/pancake-lee/pgo/pkg/config"
	"github.com/pancake-lee/pgo/pkg/logger"
)

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

func InitDefaultClient() error {
	var conf redisConfig
	err := config.Scan(&conf)
	if err != nil {
		return err
	}
	logger.Infof("load default redis with config: %+v", conf)

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
