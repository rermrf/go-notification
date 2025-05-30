package retry

import (
	"fmt"
	"go-notification/internal/pkg/retry/strategy"
	"time"
)

func NewRetry(cfg Config) (strategy.Strategy, error) {
	// 根据 config 中的字段来检测
	switch cfg.Type {
	case "fixed":
		return strategy.NewFixedIntervalRetryStrategy(cfg.FixedInterval.MaxRetries, cfg.FixedInterval.Interval), nil
	case "exponential":
		return strategy.NewExponentialBackoffRetryStrategy(cfg.ExponentialBackoff.InitialInterval, cfg.ExponentialBackoff.MaxInterval, cfg.ExponentialBackoff.MaxRetries), nil
	default:
		return nil, fmt.Errorf("unsupported retry type: %s", cfg.Type)
	}
}

// Config 重试配置
type Config struct {
	Type string `json:"type"`
	// 固定间隔
	FixedInterval *FixedIntervalConfig `json:"fixedInterval"`
	// 指数退避
	ExponentialBackoff *ExponentialBackoffConfig `json:"exponentialBackoff"`
}

// FixedIntervalConfig 固定间隔配置
type FixedIntervalConfig struct {
	MaxRetries int32         `json:"maxRetries"`
	Interval   time.Duration `json:"interval"`
}

type ExponentialBackoffConfig struct {
	// 初始重试间隔 单位ms
	InitialInterval time.Duration `json:"initialInterval"`
	// 最大重试间隔
	MaxInterval time.Duration `json:"maxInterval"`
	// 最大重试次数
	MaxRetries int32 `json:"maxRetries"`
}
