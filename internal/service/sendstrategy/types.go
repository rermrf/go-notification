package sendstrategy

import (
	"context"
	"go-notification/internal/domain"
)

type SendStrategy interface {
	// Send 单条发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
	// BatchSend 批量发送通知，其中每个通知的发送策略必须相同
	BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error)
}
