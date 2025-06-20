package ioc

import (
	"github.com/patrickmn/go-cache"
	"github.com/spf13/viper"
	"time"
)

func InitGoCache() *cache.Cache {
	type Config struct {
		DefaultExpiration time.Duration `yaml:"defaultExpiration"`
		CleanupInterval   time.Duration `yaml:"cleanupInterval"`
	}
	var cfg Config
	err := viper.UnmarshalKey("cache.local", &cfg)
	if err != nil {
		panic(err)
	}
	c := cache.New(cfg.DefaultExpiration, cfg.CleanupInterval)
	return c
}
