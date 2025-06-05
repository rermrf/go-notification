package dao

import (
	"context"
	"go-notification/internal/domain"
	"gorm.io/gorm"
	"time"
)

// CallbackLog 回调记录表
type CallbackLog struct {
	ID             int64  `gorm:"primaryKey;AUTO_INCREMENT;comment:'回调记录ID'"`
	NotificationID int64  `gorm:"column:notification_id;NOT NULL;quiqueIndex:idx_notification_id;comment:'待回调通知ID'"`
	RetryCount     int32  `gorm:"type:TINYINT;NOT NULL;default:0;comment:'重试次数'"`
	NextRetryTime  int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:0;comment:'下次重试时间戳'"`
	Status         string `gorm:"type:ENUM('INIT','PENFING','SUCCEEDED','FAILED');NOT NULL;default:'INIT';index:idx_status;comment:'回调状态'"`
	Ctime          int64
	Utime          int64
}

func (CallbackLog) TableName() string {
	return "callback_logs"
}

type CallbackLogDAO interface {
	Find(ctx context.Context, startTime, batchSize, startID int64) (logs []CallbackLog, nextStartID int64, err error)
	FindByNotificationIDs(ctx context.Context, notificationIDs []int64) (logs []CallbackLog, err error)
	Update(ctx context.Context, logs []CallbackLog) error
}

type callbackLogDAO struct {
	db *gorm.DB
}

func NewCallbackLogDAO(db *gorm.DB) CallbackLogDAO {
	return &callbackLogDAO{db: db}
}

func (c *callbackLogDAO) Find(ctx context.Context, startTime, batchSize, startID int64) (logs []CallbackLog, nextStartID int64, err error) {
	nextStartID = 0

	res := c.db.WithContext(ctx).Model(&CallbackLog{}).
		Where("next_retry_time <= ?", startTime).
		Where("status = ?", domain.CallbackLogStatusPending).
		Where("id > ?", startID).
		Order("id ASC").
		Limit(int(batchSize)).
		Find(&logs)

	if res.Error != nil {
		return logs, nextStartID, res.Error
	}

	if len(logs) > 0 {
		nextStartID = logs[len(logs)-1].ID
	}

	return logs, nextStartID, nil
}

func (c *callbackLogDAO) FindByNotificationIDs(ctx context.Context, notificationIDs []int64) (logs []CallbackLog, err error) {
	err = c.db.WithContext(ctx).Where("notification_id IN (?)", notificationIDs).Find(&logs).Error
	return logs, err
}

func (c *callbackLogDAO) Update(ctx context.Context, logs []CallbackLog) error {
	if len(logs) == 0 {
		return nil
	}
	utime := time.Now().UnixMilli()
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, log := range logs {
			res := tx.Model(&CallbackLog{ID: log.ID}).Updates(map[string]interface{}{
				"retry_count":     log.RetryCount,
				"next_retry_time": log.NextRetryTime,
				"status":          log.Status,
				"utime":           utime,
			})
			if res.Error != nil {
				return res.Error
			}
		}
		return nil
	})
}
