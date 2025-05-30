package domain

import (
	"go-notification/internal/pkg/retry"
	"time"
)

type TxNotification struct {
	// 事务id
	TxID int64

	// 创建的通知id
	Notification Notification
	// 业务方标识
	BizID int64
	// 业务内的唯一标识
	Key        string
	Status     TxNotificationStatus
	CheckCount int
	// NextCheckTime 根据重试策略得出的下一次的回查时间戳 0 表示不需要重试
	NextCheckTime int64
	Ctime         int64
	Utime         int64
}

func (t *TxNotification) SetSendTime() {
	t.Notification.SetSendTime()
}

func (t *TxNotification) SetNextCheckBackTimeAndStatus(txnCfg *TxnConfig) {
	nextTime, ok := t.NextCheckBackTime(txnCfg)
	// 可以重试
	if ok {
		t.NextCheckTime = time.Now().Add(nextTime).UnixMilli()
	} else {
		// 不能重试讲状态变为fail
		t.NextCheckTime = 0
		t.Status = TxNotificationStatusFail
	}
}

func (t *TxNotification) NextCheckBackTime(cfg *TxnConfig) (time.Duration, bool) {
	if cfg == nil || cfg.RetryPolicy == nil {
		return 0, false
	}
	s, err := retry.NewRetry(*cfg.RetryPolicy)
	if err != nil {
		return 0, false
	}
	return s.NextWithRetries(int32(t.CheckCount))
}

type TxNotificationStatus string

func (status TxNotificationStatus) String() string {
	return string(status)
}

const (
	TxNotificationStatusPrepare TxNotificationStatus = "PREPARE" // 准备发送
	TxNotificationStatusCommit  TxNotificationStatus = "COMMIT"  // 提交
	TxNotificationStatusCancel  TxNotificationStatus = "CANCEL"  // 用户主动取消
	TxNotificationStatusFail    TxNotificationStatus = "FAIL"    // 多次重试后失败
)
