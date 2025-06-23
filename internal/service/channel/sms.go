package channel

import (
	"context"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/service/provider"
)

type baseChannel struct {
	builder provider.SelectorBuilder
}

func (b *baseChannel) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	selector, err := b.builder.Build()
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, err)
	}

	for {
		// 获取供应商
		p, er := selector.Next(ctx, notification)
		if er != nil {
			return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, er)
		}

		// 使用当前供应商发送
		resp, er1 := p.Send(ctx, notification)
		if er1 == nil {
			return resp, nil
		}
	}
}

type smsChannel struct {
	baseChannel
}

func NewSmsChannel(builder provider.SelectorBuilder) Channel {
	return &smsChannel{baseChannel: baseChannel{builder: builder}}
}
