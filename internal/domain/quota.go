package domain

type Quota struct {
	BizID   int64   // 业务ID
	Quota   int32   // 配额数量
	Channel Channel // 渠道类型
}
