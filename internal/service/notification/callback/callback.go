package callback

import (
	"context"
	"fmt"
	notificationv1 "go-notification/api/proto/gen/api/proto/notification/v1"
	clientv1 "go-notification/api/proto/gen/client/v1"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	mygrpc "go-notification/internal/pkg/grpc"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/retry"
	"go-notification/internal/repository"
	configSvc "go-notification/internal/service/config"
	"google.golang.org/grpc"
	"sync"
	"time"
)

var _ Service = (*service)(nil)

type Service interface {
	SendCallback(ctx context.Context, startTime, batchSize int64) error
	SendCallbackByNotification(ctx context.Context, notification domain.Notification) error
	SendCallbackByNotifications(ctx context.Context, notifications []domain.Notification) error
}

type service struct {
	configSvc    configSvc.BusinessConfigService
	bizID2Config sync.Map
	clients      *mygrpc.Clients[clientv1.CallbackServiceClient]
	repo         repository.CallbackLogRepository
	logger       logger.Logger
}

func newService(configSvc configSvc.BusinessConfigService, repo repository.CallbackLogRepository, logger logger.Logger) Service {
	return &service{
		configSvc:    configSvc,
		bizID2Config: sync.Map{},
		clients: mygrpc.NewClients(func(conn *grpc.ClientConn) clientv1.CallbackServiceClient {
			return clientv1.NewCallbackServiceClient(conn)
		}),
		repo:   repo,
		logger: logger,
	}
}

func (s *service) SendCallback(ctx context.Context, startTime, batchSize int64) error {
	// 使用分页查询
	var nextStartID int64
	for {
		// 查询需要回调的通知
		logs, newNextStartID, err := s.repo.Find(ctx, startTime, batchSize, nextStartID)
		if err != nil {
			s.logger.Error("查询回调日志失败",
				logger.Int64("startTime", startTime),
				logger.Int64("batchSize", batchSize),
				logger.Int64("nextStartID", nextStartID),
				logger.Error(err))
			return err
		}

		if len(logs) == 0 {
			break
		}

		// 处理当前批次通知
		err = s.sendCallbackAndUpdateCallBackLogs(ctx, logs)
		if err != nil {
			return err
		}
		nextStartID = newNextStartID
	}
	return nil
}

func (s *service) SendCallbackByNotification(ctx context.Context, notification domain.Notification) error {
	logs, err := s.repo.FindByNotificationIDs(ctx, []int64{notification.ID})
	if err != nil {
		return err
	}
	return s.sendCallbackAndUpdateCallBackLogs(ctx, logs)
}

func (s *service) SendCallbackByNotifications(ctx context.Context, notifications []domain.Notification) error {
	notificationIDs := make([]int64, 0, len(notifications))
	mp := make(map[int64]domain.Notification, len(notifications))
	for i := range notifications {
		notificationIDs = append(notificationIDs, notifications[i].ID)
		mp[notifications[i].ID] = notifications[i]
	}

	logs, err := s.repo.FindByNotificationIDs(ctx, notificationIDs)
	if err != nil {
		return err
	}
	if len(logs) == len(notifications) {
		return s.sendCallbackAndUpdateCallBackLogs(ctx, logs)
	}

	for i := range logs {
		// 删除有回调记录的通知
		delete(mp, logs[i].Notification.ID)
	}

	var er error
	if len(logs) != 0 {
		// 部分有回调记录（调度器调度发送成功后触发）
		er = s.sendCallbackAndUpdateCallBackLogs(ctx, logs)
	}
	for k := range mp {
		// 全部没有回调记录（同步立刻批量发送，或者同步非立刻发送同时没有回调配置）
		// 部分没有回调记录（调度器调度发送成功后触发）
		// Client 上没有批量接口，这里可以考虑开协程
		_, er = s.sendCallback(ctx, mp[k])
	}
	return er
}

func (s *service) sendCallbackAndUpdateCallBackLogs(ctx context.Context, logs []domain.CallbackLog) error {
	needUpdate := make([]domain.CallbackLog, 0, len(logs))
	for i := range logs {
		changed, err := s.sendCallbackAndSetChangedFields(ctx, &logs[i])
		if err != nil {
			s.logger.Warn("业务方回调失败",
				logger.Int64("Callback.ID", logs[i].ID),
				logger.Error(err))
			continue
		}
		if changed {
			needUpdate = append(needUpdate, logs[i])
		}
	}
	return s.repo.Update(ctx, needUpdate)
}

func (s *service) sendCallbackAndSetChangedFields(ctx context.Context, log *domain.CallbackLog) (changed bool, err error) {
	resp, err := s.sendCallback(ctx, log.Notification)
	if err != nil {
		return false, err
	}

	// 拿到业务方对回调处理的结果
	if resp.Success {
		log.Status = domain.CallbackLogStatusSuccess
		return true, nil
	}

	// 业务方对回调的处理失败，需要重试，此时业务方必定有配置
	cfg, _ := s.getConfig(ctx, log.Notification.BizID)
	retryStrategy, _ := retry.NewRetry(*cfg.RetryPolicy)
	interval, ok := retryStrategy.NextWithRetries(log.RetryCount)
	if ok {
		// 为达到最大重试次数，状态不变但要更新下一次重试时间和重试次数
		log.NextRetryTime = time.Now().Add(interval).UnixMilli()
		log.RetryCount++
	} else {
		// 达到最大重试次数，不再重试，更新状态为失败
		log.Status = domain.CallbackLogStatusFailed
	}
	return true, nil
}

func (s *service) sendCallback(ctx context.Context, notification domain.Notification) (*clientv1.HandleNotificationResultResponse, error) {
	cfg, err := s.getConfig(ctx, notification.BizID)
	if err != nil {
		s.logger.Warn("获取业务配置失败",
			logger.String("key", "BizID"),
			logger.Int64("bizID", notification.BizID),
			logger.Error(err))
		return nil, err
	}
	if cfg == nil {
		return nil, fmt.Errorf("%w", errs.ErrConfigNotFound)
	}
	return s.clients.Get(cfg.ServiceName).HandleNotificationResult(ctx, s.buildRequest(notification))
}

func (s *service) getConfig(ctx context.Context, bizId int64) (*domain.CallbackConfig, error) {
	cfg, ok := s.bizID2Config.Load(bizId)
	if ok {
		return cfg.(*domain.CallbackConfig), nil
	}
	bizConfig, err := s.configSvc.GetByID(ctx, bizId)
	if err != nil {
		return nil, err
	}
	if bizConfig.CallbackConfig != nil {
		s.bizID2Config.Store(bizId, bizConfig.CallbackConfig)
	}
	return bizConfig.CallbackConfig, nil
}

func (s *service) buildRequest(notification domain.Notification) *clientv1.HandleNotificationResultRequest {
	templateParams := make(map[string]string)
	if notification.Template.Params != nil {
		templateParams = notification.Template.Params
	}
	return &clientv1.HandleNotificationResultRequest{
		NotificationId: notification.ID,
		OriginalRequest: &notificationv1.SendNotificationRequest{
			Notification: &notificationv1.Notification{
				Key:            notification.Key,
				Receivers:      notification.Receivers,
				Channel:        s.getChannel(notification),
				TemplateId:     fmt.Sprintf("%d", notification.Template.ID),
				TemplateParams: templateParams,
			},
		},
		Result: &notificationv1.SendNotificationResponse{
			NotificationId: uint64(notification.ID),
			Status:         s.getStatus(notification),
		},
	}
}

func (s *service) getChannel(notification domain.Notification) notificationv1.Channel {
	var channel notificationv1.Channel
	switch notification.Channel {
	case domain.ChannelSMS:
		channel = notificationv1.Channel_SMS
	case domain.ChannelEmail:
		channel = notificationv1.Channel_EMAIL
	case domain.ChannelInApp:
		channel = notificationv1.Channel_IN_APP
	default:
		channel = notificationv1.Channel_CHANNEL_UNSPECIFIED
	}
	return channel
}

func (s *service) getStatus(notification domain.Notification) notificationv1.SendStatus {
	var status notificationv1.SendStatus
	switch notification.Status {
	case domain.SendStatusSucceeded:
		status = notificationv1.SendStatus_SUCCEEDED
	case domain.SendStatusFailed:
		status = notificationv1.SendStatus_FAILED
	case domain.SendStatusCanceled:
		status = notificationv1.SendStatus_CANCELED
	case domain.SendStatusPending:
		status = notificationv1.SendStatus_PENDING
	case domain.SendStatusPrepare:
		status = notificationv1.SendStatus_PREPARE
	case domain.SendStatusSending:
		status = notificationv1.SendStatus_SEND_STATUS_UNSPECIFIED
	default:
		status = notificationv1.SendStatus_SEND_STATUS_UNSPECIFIED
	}
	return status
}
