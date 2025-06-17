package scheduler

import (
	"context"
	"github.com/meoying/dlock-go"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/loopjob"
	"go-notification/internal/service/notification"
	"go-notification/internal/service/sender"
	"time"
)

// NotificationScheduler 通知调服服务接口
type NotificationScheduler interface {
	Start(ctx context.Context)
}

// staticScheduler 通知调度服务实现
type staticScheduler struct {
	notificationSvc notification.Service
	sender          sender.NotificationSender
	dclient         dlock.Client
	log             logger.Logger

	batchSize int
}

func NewStaticScheduler(notificationSvc notification.Service, sender sender.NotificationSender, dclient dlock.Client, log logger.Logger) NotificationScheduler {
	const defaultBatchSize = 10
	return &staticScheduler{
		notificationSvc: notificationSvc,
		sender:          sender,
		dclient:         dclient,
		batchSize:       defaultBatchSize,
	}
}

// Start 启动调度服务
// 当 ctx 被取消的或者关闭的时候，就会结束循环
func (s *staticScheduler) Start(ctx context.Context) {
	const key = "notificatipn_async_scheduler"
	lj := loopjob.NewInfiniteLoop(s.dclient, s.log, s.processPendingNotifications, key)
	lj.Run(ctx)
}

// processPendingNotifications 处理待发送的通知
func (s *staticScheduler) processPendingNotifications(ctx context.Context) error {
	const defaultTimeout = 3 * time.Second
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	const offset = 0
	notifications, err := s.notificationSvc.FindReadyNotifications(ctx, offset, s.batchSize)
	if err != nil {
		return err
	}
	if len(notifications) == 0 {
		time.Sleep(time.Second)
		return nil
	}
	_, err = s.sender.BatchSend(ctx, notifications)
	return err
}
