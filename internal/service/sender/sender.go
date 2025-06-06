package sender

import (
	"context"
	"fmt"
	"github.com/ecodeclub/ekit/pool"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository"
	"go-notification/internal/service/channel"
	configSvc "go-notification/internal/service/config"
	"go-notification/internal/service/notification/callback"
	"sync"
)

// NotificationSender 通知发送接口
//
//go:generate mockgen -source=./sender.go -destination=./mocks/sender.mock.go -package=sendermocks -type NotificationSender
type NotificationSender interface {
	// Send 单条发送通知
	Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error)
	// BatchSend 发送批量通知
	BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error)
}

type sender struct {
	repo        repository.NotificationRepository
	configSvc   configSvc.BusinessConfigService
	callbackSvc callback.Service
	channel     channel.Channel
	taskPool    pool.TaskPool
	logger      logger.Logger
}

// NewSender 创建通知发送器
func NewSender(
	repo repository.NotificationRepository,
	configSvc configSvc.BusinessConfigService,
	callbackSvc callback.Service,
	channel channel.Channel,
	taskPool pool.TaskPool,
	logger logger.Logger,
) NotificationSender {
	return &sender{repo: repo, configSvc: configSvc, callbackSvc: callbackSvc, channel: channel, taskPool: taskPool, logger: logger}
}

func (s *sender) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	resp := domain.SendResponse{
		NotificationID: notification.ID,
	}
	_, err := s.channel.Send(ctx, notification)
	if err != nil {
		s.logger.Error("发送失败 %w", logger.Error(err))
		resp.Status = domain.SendStatusFailed
		notification.Status = domain.SendStatusFailed
		// 如果是 FAILED，你需要把 quota 加回去
		err = s.repo.MarkFailed(ctx, notification)
	} else {
		resp.Status = domain.SendStatusSucceeded
		notification.Status = domain.SendStatusSucceeded
		err = s.repo.MarkSuccess(ctx, notification)
	}

	// 更新发送状态
	if err != nil {
		return domain.SendResponse{}, err
	}

	// 得到准确的发送结果，发起回调，发送成功和失败都应该回调

	_ = s.callbackSvc.SendCallbackByNotification(ctx, notification)
	return resp, nil
}

func (s *sender) BatchSend(ctx context.Context, notifications []domain.Notification) ([]domain.SendResponse, error) {
	if len(notifications) == 0 {
		return nil, nil
	}

	// 并发发送通知
	var succeedMu, failedMu sync.Mutex
	var succeeded, failed []domain.SendResponse

	var wg sync.WaitGroup
	wg.Add(len(notifications))
	for i := range notifications {
		n := notifications[i]
		err := s.taskPool.Submit(ctx, pool.TaskFunc(func(ctx context.Context) error {
			defer wg.Done()
			_, err := s.channel.Send(ctx, n)
			if err != nil {
				resp := domain.SendResponse{
					NotificationID: n.ID,
					Status:         domain.SendStatusFailed,
				}
				failedMu.Lock()
				failed = append(failed, resp)
				failedMu.Unlock()
			} else {
				resp := domain.SendResponse{
					NotificationID: n.ID,
					Status:         domain.SendStatusSucceeded,
				}
				succeedMu.Lock()
				succeeded = append(succeeded, resp)
				succeedMu.Unlock()
			}
			s.logger.Info(fmt.Sprintf("submit notification[%d] = %#v\n", i, n))
			return nil
		}))
		if err != nil {
			s.logger.Warn("提交任务到任务池失败", logger.Error(err), logger.Any("notification", n))
			return nil, fmt.Errorf("提交任务到任务池失败：%w", err)
		}
	}
	wg.Wait()

	// 获取通知信息，以便于获取版本号
	allNotificationIDs := make([]int64, 0, len(failed)+len(succeeded))
	for _, succ := range succeeded {
		allNotificationIDs = append(allNotificationIDs, succ.NotificationID)
	}
	for _, fail := range failed {
		allNotificationIDs = append(allNotificationIDs, fail.NotificationID)
	}

	// 获取所有通知的详细信息，包括版本号
	notificationsMap, err := s.repo.BatchGetByID(ctx, allNotificationIDs)
	if err != nil {
		s.logger.Warn("批量获取通知详情失败", logger.Error(err), logger.Any("notificationIDs", allNotificationIDs))
		return nil, fmt.Errorf("批量获取通知详情失败 %w", err)
	}

	succeedNotifications := s.getUpdatedNotifications(succeeded, notificationsMap)
	failedNotifications := s.getUpdatedNotifications(failed, notificationsMap)

	// 更新发送状态
	err = s.batchUpdateStatus(ctx, succeedNotifications, failedNotifications)
	if err != nil {
		return nil, err
	}
	// 得到准确的发送结果，发起调用，发送成功和发送失败都应该回调
	_ = s.callbackSvc.SendCallbackByNotifications(ctx, append(succeedNotifications, failedNotifications...))

	// 合并结果并返回
	return append(succeeded, failed...), nil
}

// getUpdatedNotifications 获取更新字段后的实体
func (s *sender) getUpdatedNotifications(responses []domain.SendResponse, notificationsMap map[int64]domain.Notification) []domain.Notification {
	notifications := make([]domain.Notification, 0, len(responses))
	for i := range responses {
		if n, ok := notificationsMap[responses[i].NotificationID]; ok {
			n.Status = responses[i].Status
			notifications = append(notifications, n)
		}
	}
	return notifications
}

func (s *sender) batchUpdateStatus(ctx context.Context, succeedNotifications []domain.Notification, failedNotifications []domain.Notification) error {
	if len(succeedNotifications) > 0 || len(succeedNotifications) > 0 {
		err := s.repo.BatchUpdateStatusSucceededOrFailed(ctx, succeedNotifications, failedNotifications)
		if err != nil {
			s.logger.Warn("批量更新通知状态失败",
				logger.Error(err),
				logger.Any("succeddNotifications", succeedNotifications),
				logger.Any("failedNotifications", failedNotifications),
			)
			return fmt.Errorf("批量更新通知状态失败：%w", err)
		}
	}
	return nil
}
