package notification

import (
	"context"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/pkg/id_generator"
	"go-notification/internal/service/sendstrategy"
	"go-notification/internal/service/template/manage"
)

// SendService 负责处理发送
//
//go:generate mockgen -source=./send_notification.go -destination=./mocks/send_notification.mock.go -package=notificationmocks -typed SendService
type SendService interface {
	// SendNotification 单条同步发送
	SendNotification(ctx context.Context, n domain.Notification) (domain.SendResponse, error)
	// SendNotificationAsync 单条异步发送
	SendNotificationAsync(ctx context.Context, n domain.Notification) (domain.SendResponse, error)
	// BatchSendNotifications 同步批量发送
	BatchSendNotifications(ctx context.Context, ns []domain.Notification) (domain.BatchSendResponse, error)
	// BatchSendNotificationsAsync 异步批量发送
	BatchSendNotificationsAsync(ctx context.Context, ns []domain.Notification) (domain.BatchSendAsyncResponse, error)
}

type sendService struct {
	notificationSvc Service
	templateSvc     manage.ChannelTemplateService
	idGenerator     *id_generator.Generator
	sendStrategy    sendstrategy.SendStrategy
}

// SendNotification 单条同步发送
func (s *sendService) SendNotification(ctx context.Context, n domain.Notification) (domain.SendResponse, error) {
	resp := domain.SendResponse{
		Status: domain.SendStatusFailed,
	}

	// 校验参数
	if err := n.Validate(); err != nil {
		return resp, err
	}

	// 生成通知ID，后续考虑分库分表
	id := s.idGenerator.GenerateID(n.BizID, n.Key)
	n.ID = id

	// 发送通知
	resp, err := s.sendStrategy.Send(ctx, n)
	if err != nil {
		return resp, fmt.Errorf("%w, 发送通知失败，原因：%w", errs.ErrSendNotificationFailed, err)
	}
	return resp, nil
}

// SendNotificationAsync 单条异步发送
func (s *sendService) SendNotificationAsync(ctx context.Context, n domain.Notification) (domain.SendResponse, error) {
	// 参数校验
	if err := n.Validate(); err != nil {
		return domain.SendResponse{}, err
	}
	// 生成通知ID
	id := s.idGenerator.GenerateID(n.BizID, n.Key)
	n.ID = id

	// 使用异步接口但要立即发送，修改为延迟发送
	// 本质上这是一个不怎么好的用法，但是业务方可能不清楚，所以我们兼容一下
	n.ReplaceAsyncImmediate()
	return s.sendStrategy.Send(ctx, n)
}

// BatchSendNotifications 批量同步发送
func (s *sendService) BatchSendNotifications(ctx context.Context, ns []domain.Notification) (domain.BatchSendResponse, error) {
	response := domain.BatchSendResponse{}

	if len(ns) == 0 {
		return response, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}

	// 校验并且生成 ID
	for i := range ns {
		n := ns[i]
		if err := n.Validate(); err != nil {
			return response, fmt.Errorf("参数非法 %w", err)
		}
		// 生成通知 ID
		id := s.idGenerator.GenerateID(n.BizID, n.Key)
		ns[i].ID = id
	}

	// 发送通知，这里有一个隐含的假设，就是发送策略必须是相同的
	results, err := s.sendStrategy.BatchSend(ctx, ns)
	response.Results = results
	if err != nil {
		return response, fmt.Errorf("%w, 发送失败 %w", errs.ErrSendNotificationFailed, err)
	}
	return response, nil
}

// BatchSendNotificationsAsync 批量异步发送
func (s *sendService) BatchSendNotificationsAsync(ctx context.Context, ns []domain.Notification) (domain.BatchSendAsyncResponse, error) {
	// 参数校验
	if len(ns) == 0 {
		return domain.BatchSendAsyncResponse{}, fmt.Errorf("%w: 通知列表不能为空", errs.ErrInvalidParameter)
	}

	ids := make([]int64, 0, len(ns))
	// 生成 ID，并且进行校验
	for i := range ns {
		if err := ns[i].Validate(); err != nil {
			return domain.BatchSendAsyncResponse{}, fmt.Errorf("参数非法 %w", err)
		}
		// 生成通知ID
		id := s.idGenerator.GenerateID(ns[i].BizID, ns[i].Key)
		ns[i].ID = id
		ids = append(ids, id)
		ns[i].ReplaceAsyncImmediate()
	}

	// 发送通知，隐含假设这一批的发送策略是一样的
	_, err := s.sendStrategy.BatchSend(ctx, ns)
	if err != nil {
		return domain.BatchSendAsyncResponse{}, fmt.Errorf("%w, 发送失败 %w", errs.ErrSendNotificationFailed, err)
	}
	return domain.BatchSendAsyncResponse{
		NotificationIDs: ids,
	}, nil
}
