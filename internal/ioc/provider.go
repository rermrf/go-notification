package ioc

import "github.com/spf13/viper"

// InitProviderEncryptKey 提供provider服务所需的加密密钥
func InitProviderEncryptKey() string {
	type Config struct {
		Key string `yaml:"key"`
	}
	var cfg Config
	err := viper.UnmarshalKey("provider", &cfg)
	if err != nil {
		panic(err)
	}
	return cfg.Key
}
