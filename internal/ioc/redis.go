package ioc

import (
	"github.com/redis/go-redis/v9"
	"github.com/spf13/viper"
	"go-notification/internal/pkg/redis/metrics"
	"go-notification/internal/pkg/redis/tracing"
)

func InitRedisClient() *redis.Client {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(err)
	}
	cmd := redis.NewClient(&redis.Options{Addr: cfg.Addr})
	cmd = tracing.Withtracing(cmd)
	cmd = metrics.WithMetrics(cmd)
	return cmd
}

func InitRedisCmdable() redis.Cmdable {
	type Config struct {
		Addr string `yaml:"addr"`
	}
	var cfg Config
	err := viper.UnmarshalKey("redis", &cfg)
	if err != nil {
		panic(err)
	}
	cmd := redis.NewClient(&redis.Options{Addr: cfg.Addr})
	cmd = tracing.Withtracing(cmd)
	cmd = metrics.WithMetrics(cmd)
	return cmd
}
