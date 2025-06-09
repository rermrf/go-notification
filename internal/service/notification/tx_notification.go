package notification

import (
	"context"
	"github.com/meoying/dlock-go"
	clientv1 "go-notification/api/proto/gen/client/v1"
	"go-notification/internal/domain"
	pgrpc "go-notification/internal/pkg/grpc"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository"
	"go-notification/internal/service/config"
	"go-notification/internal/service/sender"
	"google.golang.org/grpc"
	"time"
)

//go:generate mockgen -source=./tx_notification.go -destination=./mocks/tx_notification.mock.go -package=notificationmocks -typed TxNotificationService
type TxNotificationService interface {
	// Prepare 准备消息
	Prepare(ctx context.Context, notification domain.Notification) (int64, error)
	// Commit 提交
	Commit(ctx context.Context, bizID int64, key string) error
	// Cancel 取消
	Cancel(ctx context.Context, bizID int64, key string) error
}

type txNotificationService struct {
	repo      repository.TxNotificationRepository
	notiRepo  repository.NotificationRepository
	configSvc config.BusinessConfigService
	logger    logger.Logger
	lock      dlock.Client
	sender    sender.NotificationSender
}

const defaultBatchSize = 10

func (s *txNotificationService) StartTask(ctx context.Context) {
	clients := pgrpc.NewClients[clientv1.TransactionCheckServiceClient](func(conn *grpc.ClientConn) clientv1.TransactionCheckServiceClient {
		return clientv1.NewTransactionCheckServiceClient(conn)
	})
	task := &TxCheckTask{
		repo:      s.repo,
		configSvc: s.configSvc,
		logger:    s.logger,
		lock:      s.lock,
		batchSize: defaultBatchSize,
		clients:   clients,
	}
	go task.Start(ctx)
}

func (s *txNotificationService) Prepare(ctx context.Context, notification domain.Notification) (int64, error) {
	notification.Status = domain.SendStatusPrepare
	notification.SetSendTime()
	txn := domain.TxNotification{
		Notification: notification,
		Key:          notification.Key,
		BizID:        notification.BizID,
		Status:       domain.TxNotificationStatusPrepare,
	}

	cfg, err := s.configSvc.GetByID(ctx, notification.BizID)
	if err == nil {
		now := time.Now().UnixMilli()
		const second = 1000
		if cfg.TxnConfig != nil {
			txn.NextCheckTime = now + int64(cfg.TxnConfig.InitialDelay*second)
		}
	}
	return s.repo.Create(ctx, txn)
}

func (s *txNotificationService) Commit(ctx context.Context, bizID int64, key string) error {
	err := s.repo.UpdateStatus(ctx, bizID, key, domain.TxNotificationStatusCommit, domain.SendStatusSending)
	if err != nil {
		return err
	}
	notification, err := s.notiRepo.GetByKey(ctx, bizID, key)
	if err != nil {
		return err
	}
	if notification.IsImmediate() {
		_, err = s.sender.Send(ctx, notification)
	}
	return err
}

func (s *txNotificationService) Cancel(ctx context.Context, bizID int64, key string) error {
	return s.repo.UpdateStatus(ctx, bizID, key, domain.TxNotificationStatusCancel, domain.SendStatusCanceled)
}
