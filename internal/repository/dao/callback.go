package dao

// CallbackLog 回调记录表
type CallbackLog struct {
	ID             int64  `gorm:"primaryKey;AUTO_INCREMENT;comment:'回调记录ID'"`
	NotificationID int64  `gorm:"column:notification_id;NOT NULL;quiqueIndex:idx_notification_id;comment:'待回调通知ID'"`
	RetryCount     int8   `gorm:"type:TINYINT;NOT NULL;default:0;comment:'重试次数'"`
	NextRetryTime  int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:0;comment:'下次重试时间戳'"`
	Status         string `gorm:"type:ENUM('INIT','PENFING','SUCCEEDED','FAILED');NOT NULL;default:'INIT';index:idx_status;comment:'回调状态'"`
	Ctime          int64
	Utime          int64
}

func (CallbackLog) TableName() string {
	return "callback_logs"
}
