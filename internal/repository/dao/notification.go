package dao

import (
	"context"
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
