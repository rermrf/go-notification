package sendstrategy

import (
	"context"
	"fmt"
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
	immediate       *ImmediateSendStrategy
	defaultStrategy *DefaultSendStrategy
}

func NewDispatcher(immediate *ImmediateSendStrategy, defaultStrategy *DefaultSendStrategy) *Dispatcher {
	return &Dispatcher{immediate: immediate, defaultStrategy: defaultStrategy}
}

// Send 发送通知
func (d *Dispatcher) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	// 执行发送
	return d.selectStrategy(notification).Send(ctx, notification)
}

// BatchSend 批量发送通知
func (d *Dispatcher) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, fmt.Errorf("%w: 通知列表不能为空")
	}
	// 同一批发送策略是一致的
	return d.selectStrategy(notifications[0]).BatchSend(ctx, notifications)
}

func (d *Dispatcher) selectStrategy(notification domain.Notification) SendStrategy {
	if notification.SendStrategyConfig.Type == domain.SendStrategyImmediate {
		return d.immediate
	}
	return d.defaultStrategy
}
