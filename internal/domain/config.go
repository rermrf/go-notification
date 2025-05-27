package domain

import "go-notification/internal/pkg/retry"

type BusinessConfig struct {
	ID             int64           // 业务标识
	OwnerId        int64           // 业务方 ID
	OwnerType      string          // 业务方类型：person - 个人，organization - 组织
	ChannelConfig  *ChannelConfig  // 渠道配置，json格式
	TxnConfig      *TxnConfig      // 事务配置，json格式
	RateLimit      int             // 速率限制
	Quota          *QuotaConfig    // 配额配置，json格式
	CallbackConfig *CallbackConfig // 回调配置，json格式
	Ctime          int64
	Utime          int64
}

type ChannelConfig struct {
	Channels    []ChannelItem `json:"channels"`
	RetryPolicy *retry.Config `json:"retryPolicy"`
}

type ChannelItem struct {
	Channel  string `json:"channel"`
	Priority int    `json:"priority"`
	Enabled  bool   `json:"enabled"`
}

type TxnConfig struct {
	// 回查方法名
	ServiceName string `json:"serviceName"`
	// 期望事物在 initialDelay 秒后完成
	InitialDelay int `json:"initialDelay"`
	// 回查的重试策略
	RetryPolicy *retry.Config `json:"retryPolicy"`
}

// QuotaConfig 配额配置
type QuotaConfig struct {
	Monthly MonthlyConfig `json:"monthly"`
}

// MonthlyConfig 每月配额配置
type MonthlyConfig struct {
	SMS   int `json:"sms"`
	EMAIL int `json:"email"`
}

// CallbackConfig 回调配置
type CallbackConfig struct {
	ServiceName string        `json:"serviceName"`
	RetryPolicy *retry.Config `json:"retryPolicy"`
}
