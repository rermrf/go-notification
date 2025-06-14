package tracing

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/service/provider"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Provider 为供应商实现添加链路追踪的装饰器
type Provider struct {
	provider provider.Provider
	tracer   trace.Tracer
	name     string
}

// NewProvider 创建一个新的带有链路追踪的供应商
// name 应该传入类似于 tencent，ali 这种名字
func NewProvider(provider provider.Provider, name string) *Provider {
	return &Provider{
		provider: provider,
		name:     name,
		tracer:   otel.Tracer("go-notification/provider"),
	}
}

func (p Provider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	ctx, span := p.tracer.Start(ctx, "Provider.Send",
		trace.WithAttributes(
			attribute.String("provider.name", p.name),
			attribute.Int64("notification.id", notification.ID),
			attribute.Int64("notification.bizID", notification.BizID),
			attribute.String("notification.key", notification.Key),
			attribute.String("notification.channel", string(notification.Channel)),
		))
	defer span.End()

	// 调用底层供应商发送通知
	response, err := p.provider.Send(ctx, notification)
	if err != nil {
		// 记录错误
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		// 记录成功响应的属性
		span.SetAttributes(
			attribute.Int64("notification.id", response.NotificationID),
			attribute.String("notification.status", string(response.Status)),
		)
	}
	
	return response, err
}
