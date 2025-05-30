package notification

import (
	"context"
	"go-notification/internal/domain"
)

// SendService 复杂处理发送
//
//go:generate mockgen -source=./send_notification.go -destination=./mocks/send_notification.mock.go -package=notificationmocks -typed SendService
type SendService interface {
	// SendNotification 单条同步发送
	SendNotification(ctx context.Context, n domain.Notification) (domain.SendResponse, error)
	// SendNotificationAsync 单条异步发送
	SendNotificationAsync(ctx context.Context, n domain.Notification) (domain.SendResponse, error)
	// BatchSendNotifications 同步批量发送
	BatchSendNotifications(ctx context.Context, ns []domain.Notification) (domain.SendResponse, error)
	// BatchSendNotificationsAsync 异步批量发送
	BatchSendNotificationsAsync(ctx context.Context, ns []domain.Notification) (domain.SendResponse, error)
}
