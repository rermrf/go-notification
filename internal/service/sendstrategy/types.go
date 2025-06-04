package sendstrategy

import (
	"context"
	"go-notification/internal/domain"
)

// SendStrategy 发送策略接口
//
//go:generate mockgen -source=./types.go -destination=./mocks/send_strategy.mock.go -package=sendstrategymocks -types Sendstrategy
type SendStrategy interface {
	// Send 单条发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
	// BatchSend 批量发送通知，其中每个通知的发送策略必须相同
	BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error)
}

// Dispatcher 通知发送分发器
// 根据通知的策略类型选择合适的发送策略
type Dispatcher struct {
	immediate *
}
