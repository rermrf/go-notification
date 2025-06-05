package domain

type CallbackLogStatus string

const (
	CallbackLogStatusInit    CallbackLogStatus = "INIT"      // 初始化
	CallbackLogStatusPending CallbackLogStatus = "PENDING"   // 待处理
	CallbackLogStatusSuccess CallbackLogStatus = "SUCCEEDED" // 成功
	CallbackLogStatusFailed  CallbackLogStatus = "FAILED"    // 失败
)

func (cs CallbackLogStatus) String() string {
	return string(cs)
}

type CallbackLog struct {
	ID            int64
	Notification  Notification
	RetryCount    int32
	NextRetryTime int64
	Status        CallbackLogStatus
}
