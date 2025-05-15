package domain

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
	chennels    []ChannelItem
	RetryPolicy *retry.Config
}
