package notification

import (
	"context"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/repository"
)

//go:generate mockgen -source=./notification.go -destination=./mocks/notification.mock.go -package=notificationmocks -typed Service
type Service interface {
	// FindReadyNotifications 准备好调度发送的通知
	FindReadyNotifications(ctx context.Context, offiset, limit int) ([]domain.Notification, error)
	// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
	GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error)
}

type notificationService struct {
	repo repository.NotificationRepository
}

func newNotificationService(repo repository.NotificationRepository) Service {
	return &notificationService{repo: repo}
}

// NewNotificationService 创建通知服务实例
func NewNotificationService(repo repository.NotificationRepository) Service {
	return &notificationService{
		repo: repo,
	}
}

// FindReadyNotifications 准备好调度发送的通知
func (n *notificationService) FindReadyNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error) {
	return n.repo.FindReadNotifications(ctx, offset, limit)
}

// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
func (n *notificationService) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error) {
	if len(keys) == 0 {
		return nil, fmt.Errorf("%w: 业务内唯一标识列表不能为空", errs.ErrInvalidParameter)
	}

	notifications, err := n.repo.GetByKeys(ctx, bizID, keys...)
	if err != nil {
		return nil, fmt.Errorf("获取通知列表失败: %w", err)
	}
	return notifications, nil
}
