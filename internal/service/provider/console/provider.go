package console

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
)

// 提供一个用于测试使用的 provider 输出到控制台

type Provider struct {
	log logger.Logger
}

func (p *Provider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	p.log.Info("Sending notification", logger.Any("notification", notification))
	return domain.SendResponse{Status: domain.SendStatusSucceeded}, nil
}
