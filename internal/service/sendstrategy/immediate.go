package sendstrategy

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/repository"
	"go-notification/internal/service/sender"
)

// ImmediateSendStrategy 立即发送策略
// 同步立刻发送，异步接口选择了这个立即发送策略也不会生效
type ImmediateSendStrategy struct {
	repo   repository.NotificationRepository
	sender sender.NotificationSender
}

func NewImmediateSendStrategy(repo repository.NotificationRepository, sender sender.NotificationSender) *ImmediateSendStrategy {
	return &ImmediateSendStrategy{repo: repo, sender: sender}
}

func (i *ImmediateSendStrategy) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	notification.SetSendTime()
	created, err := i.repo.Create(ctx, notification)

	if err == nil {
		return i.sender.Send(ctx, created)
	}

	// 非唯一索引冲突直接返回错误
	if !errors.Is(err, errs.ErrNotificationDuplicate) {
		return domain.SendResponse{}, fmt.Errorf("创建通知失败: %w", err)
	}

	// 唯一索引冲突表示业务方重试
	found, err := i.repo.GetByKey(ctx, created.BizID, created.Key)
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("获取通知失败: %w", err)
	}

	// 已存在的通知为发送成功的则返回通知id和状态
	if found.Status == domain.SendStatusSucceeded {
		return domain.SendResponse{
			NotificationID: found.ID,
			Status:         found.Status,
		}, nil
	}

	// 已存在的通知状态为发送发送中的则直接返回错误
	if found.Status == domain.SendStatusSending {
		return domain.SendResponse{}, fmt.Errorf("发送失败 %w", errs.ErrSendNotificationFailed)
	}

	// 更新状态为SENDING同时获取乐观锁（版本号）
	found.Status = domain.SendStatusSending
	err = i.repo.CASStatus(ctx, found)
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("并发竞争失败: %w", err)
	}
	found.Version++
	// 再次立即发送
	return i.sender.Send(ctx, found)

}

// BatchSend 批量发送通知，其中每个通知的发送策略必须相同
func (i *ImmediateSendStrategy) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, fmt.Errorf("%w: 通知列表不能为空", errs.ErrSendNotificationFailed)
	}

	for i := range notifications {
		notifications[i].SetSendTime()
	}

	// 创建通知记录
	createdNotifications, err := i.repo.BatchCreate(ctx, notifications)
	if err != nil {
		return nil, fmt.Errorf("通知创建失败: %w", err)
	}
	// 立即发送
	return i.sender.BatchSend(ctx, createdNotifications)
}
