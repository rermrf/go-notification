package notification

import (
	"context"
	"go-notification/internal/domain"
)

//go:generate mockgen -source=./tx_notification.go -destination=./mocks/tx_notification.mock.go -package=notificationmocks -typed TxNotificationService
type TxNotificationService interface {
	// Prepare 准备消息
	Prepare(ctx context.Context, notification domain.Notification) (int64, error)
	// Commit 提交
	Commit(ctx context.Context, bizID int64, key string) error
	// Cancel 取消
	Cancel(ctx context.Context, bizID int64, key string) error
}

type txNotificationService struct {
}
