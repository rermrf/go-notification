package retry

import "time"

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
