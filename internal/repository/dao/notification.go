package dao

import (
	"context"
	"errors"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

type Notification struct {
	ID                int64  `gorm:"primaryKey;comment:'雪花算法ID'"`
	BizID             int64  `gorm:"type:BIGINT;NOT NULL;index:idx_biz_id_status,proority:1;uniqueIndex:idx_biz_id_key,priority:1;comment:'业务方配表ID，业务方可能有多个业务每个业务配置不同'"`
	Key               string `gorm:"type:VARCHAR(256);NOT NULL;quiqueIndex:idx_biz_id_key,priority:2;comment:'业务内唯一标识'"`
	Receivers         string `gorm:"type:TEXT;NOT NULL;comment:'接收者(手机/邮箱/用户ID)，JSON数组'"`
	Channel           string `gorm:"type:ENUM('SMS', 'EMAIL', 'IN_APP');NOT NULL;comment:'发送渠道'"`
	TemplateID        int64  `gorm:"type:BIGINT;NOT NULL;comment:'关联的模版ID'"`
	TemplateVersionID int64  `gorm:"type:BIGINT;NOT NULL;comment:'关联的模版版本ID'"`
	Status            string `gorm:"type:ENUM('PREPARE', 'CANCELED', 'PENDING', 'SENDING', 'SUCCEEDED', 'FAILED');DEFAULT:'PENDING';index:idx_biz_id_status,priority:2;comment:'发送状态'"`
	ScheduledSTime    int64  `gorm:"column:scheduled_time;index:idx_scheuled,priority:1;comment:'计划发送开始时间'"`
	ScheduledETime    int64  `gorm:"column:scheduled_time;index:idx_scheuled,priority:2;comment:'计划发送结束时间'"`
	Version           int    `gorm:"type:INT;NOT NULL;DEFAULT:1;comment:'版本号'"`
	Ctime             int64
	Utime             int64
}

type NotificationDAO interface {
	Create(ctx context.Context, data Notification) (Notification, error)
	CreateWithCallbackLog(ctx context.Context, data Notification) (Notification, error)
	BatchCreate(ctx context.Context, dataList []Notification) ([]Notification, error)
	BatchCreateWithCallbackLog(ctx context.Context, dataList []Notification) ([]Notification, error)

	GetByID(ctx context.Context, id int64) (Notification, error)

	BatchGetByID(ctx context.Context, ids []int64) (map[int64]Notification, error)

	GetByKey(ctx context.Context, BizID int64, key string) (Notification, error)
	GetByKeys(ctx context.Context, BizID int64, keys ...string) ([]Notification, error)

	CASStatus(ctx context.Context, notification Notification) error
	UpdateStatus(ctx context.Context, notification Notification) error

	BatchUpdateStatusSucceedOrFailed(ctx context.Context, succededNotifications, failedNotifications []Notification)

	FindReadyNotifications(ctx context.Context, offset, limit int) ([]Notification, error)
	MarkSuccess(ctx context.Context, notification Notification) error
	MarkFailed(ctx context.Context, notification Notification) error
	MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error)
}

type notificationDAO struct {
	db *gorm.DB
}

func NewNotificationDAO(db *gorm.DB) NotificationDAO {
	return &notificationDAO{db: db}
}

func (n *notificationDAO) Create(ctx context.Context, data Notification) (Notification, error) {
	//now := time.Now().UnixMilli()
	//data.Ctime, data.Utime = now, now
	//data.Version = 1
	//
	//err := n.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
	//	if err := tx.Create(&data).Error; err != nil {
	//		if n.isUniqueConstraintError(err) {
	//			return fmt.Errorf("通知已存在: %w", err)
	//		}
	//		// 直接操作数据库，直接扣减，扣减1
	//		res := tx.Model(&Quota{}).
	//
	//	}
	//})
	panic("implement me")
}

func (n *notificationDAO) CreateWithCallbackLog(ctx context.Context, data Notification) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) BatchCreate(ctx context.Context, dataList []Notification) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) BatchCreateWithCallbackLog(ctx context.Context, dataList []Notification) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) GetByID(ctx context.Context, id int64) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) BatchGetByID(ctx context.Context, ids []int64) (map[int64]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) GetByKey(ctx context.Context, BizID int64, key string) (Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) GetByKeys(ctx context.Context, BizID int64, keys ...string) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) CASStatus(ctx context.Context, notification Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) UpdateStatus(ctx context.Context, notification Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) BatchUpdateStatusSucceedOrFailed(ctx context.Context, succededNotifications, failedNotifications []Notification) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) FindReadyNotifications(ctx context.Context, offset, limit int) ([]Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) MarkSuccess(ctx context.Context, notification Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) MarkFailed(ctx context.Context, notification Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n *notificationDAO) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	//TODO implement me
	panic("implement me")
}

// isUniqueConstraintError 检查是否是唯一约束错误
func (n *notificationDAO) isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	me := new(mysql.MySQLError)
	if ok := errors.As(err, &me); ok {
		const uniqueIndexErrorCode = 1062
		return me.Number == uniqueIndexErrorCode
	}
	return false
}
