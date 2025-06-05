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
	//found, err :=

}

func (i *ImmediateSendStrategy) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	//TODO implement me
	panic("implement me")
}
