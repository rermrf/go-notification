package sendstrategy

import (
	"context"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository"
	configsvc "go-notification/internal/service/config"
)

// DefaultSendStrategy 延迟发送策略
type DefaultSendStrategy struct {
	repo      repository.NotificationRepository
	configsvc configsvc.BusinessConfigService
	logger    logger.Logger
}

func NewDefaultSendStrategy(repo repository.NotificationRepository, configsvc configsvc.BusinessConfigService, logger logger.Logger) *DefaultSendStrategy {
	return &DefaultSendStrategy{repo: repo, configsvc: configsvc, logger: logger}
}

// Send 单条发送通知
func (d *DefaultSendStrategy) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	notification.SetSendTime()
	// 创建通知记录
	created, err := d.create(ctx, notification)
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("创建延迟通知失败: %w", err)
	}

	return domain.SendResponse{
		NotificationID: created.ID,
		Status:         created.Status,
	}, nil
}

// BatchSend 批量发送通知，其中每个通知的发送策略必须相同
func (d *DefaultSendStrategy) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}

	for i := range notifications {
		notifications[i].SetSendTime()
	}

	// 创建通知记录
	createdNotifications, err := d.batchCreate(ctx, notifications)
	if err != nil {
		return nil, fmt.Errorf("创建延迟通知失败: %w", err)
	}

	// 进创建通知记录，等待定时任务扫描发送
	responses := make([]domain.SendResponse, len(createdNotifications))
	for i := range createdNotifications {
		responses[i] = domain.SendResponse{
			NotificationID: createdNotifications[i].ID,
			Status:         createdNotifications[i].Status,
		}
	}
	return responses, nil
}

func (d *DefaultSendStrategy) create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	if !d.needCreateCallbackLog(ctx, notification) {
		return d.repo.CreateWithCallbackLog(ctx, notification)
	}
	return d.repo.Create(ctx, notification)
}

func (d *DefaultSendStrategy) needCreateCallbackLog(ctx context.Context, notification domain.Notification) bool {
	bizConfig, err := d.configsvc.GetByID(ctx, notification.BizID)
	if err != nil {
		d.logger.Error("查找 biz config 失败", logger.Error(err))
		return false
	}
	return bizConfig.CallbackConfig != nil
}

func (d *DefaultSendStrategy) batchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	const first = 0
	// 同一批肯定是同样的需要创建回调日志
	if d.needCreateCallbackLog(ctx, notifications[first]) {
		return d.repo.BatchCreateWithCallbackLog(ctx, notifications)
	}
	return d.repo.BatchCreate(ctx, notifications)
}
