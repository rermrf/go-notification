package sender

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/repository"
)

type NotificationSender interface {
	// Send 单条发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
	// BatchSend 发送批量通知
	BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error)
}

type sender struct {
	repo repository.NotificationRepository
	configSvc
}
