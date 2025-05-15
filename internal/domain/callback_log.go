package domain

type CallbackLogStatus string

const (
	CallbackStatusInit    CallbackLogStatus = "INIT"      // 初始化
	CallbackStatusPending CallbackLogStatus = "PENDING"   // 待处理
	CallbackStatusSuccess CallbackLogStatus = "SUCCEEDED" // 成功
	CallbackStatusFailed  CallbackLogStatus = "FAILED"    // 失败
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
