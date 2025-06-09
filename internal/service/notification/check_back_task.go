package notification

import (
	"context"
	"errors"
	"fmt"
	"github.com/ecodeclub/ekit/list"
	"github.com/meoying/dlock-go"
	clientv1 "go-notification/api/proto/gen/client/v1"
	"go-notification/internal/domain"
	pgrpc "go-notification/internal/pkg/grpc"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/loopjob"
	"go-notification/internal/repository"
	"go-notification/internal/service/config"
	"go.uber.org/multierr"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"time"
)

type TxCheckTask struct {
	repo      repository.TxNotificationRepository
	configSvc config.BusinessConfigService
	logger    logger.Logger
	lock      dlock.Client
	batchSize int
	clients   *pgrpc.Clients[clientv1.TransactionCheckServiceClient]
}

func NewTxCheckTask(repo repository.TxNotificationRepository, configSvc config.BusinessConfigService, lock dlock.Client, logger logger.Logger) *TxCheckTask {
	return &TxCheckTask{
		repo:      repo,
		configSvc: configSvc,
		lock:      lock,
		logger:    logger,
		clients: pgrpc.NewClients[clientv1.TransactionCheckServiceClient](func(conn *grpc.ClientConn) clientv1.TransactionCheckServiceClient {
			return clientv1.NewTransactionCheckServiceClient(conn)
		}),
	}
}

const (
	TxCheckTaskKey  = "check_back_job"
	defaultTimeout  = time.Second * 5
	committedStatus = 1
	unknownStatus   = 0
	cancelStatus    = 2
)

func (task *TxCheckTask) Start(ctx context.Context) {
	job := loopjob.NewInfiniteLoop(task.lock, task.logger, task.oneLoop, TxCheckTaskKey)
	job.Run(ctx)
}

// 为了性能，使用了批量操作，针对的是数据库的批量操作
func (task *TxCheckTask) oneLoop(ctx context.Context) error {
	loopCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	txNotifications, err := task.repo.FindCheckBack(loopCtx, 0, task.batchSize)
	if err != nil {
		return err
	}

	if len(txNotifications) == 0 {
		// 避免立刻又调度
		time.Sleep(time.Second)
		return nil
	}

	bizIds := make([]int64, 0, len(txNotifications))
	for _, txNotification := range txNotifications {
		bizIds = append(bizIds, txNotification.BizID)
	}
	configMap, err := task.configSvc.GetByIDs(ctx, bizIds)
	if err != nil {
		return err
	}
	length := len(txNotifications)
	// 这一次回查没有拿到明确结果的
	retryTxns := &list.ConcurrentList[domain.TxNotification]{
		List: list.NewArrayList[domain.TxNotification](length),
	}

	// 要回滚的
	failTxns := &list.ConcurrentList[domain.TxNotification]{
		List: list.NewArrayList[domain.TxNotification](length),
	}

	// 要提交的
	commitTxns := &list.ConcurrentList[domain.TxNotification]{
		List: list.NewArrayList[domain.TxNotification](length),
	}

	// 分别处理
	var eg errgroup.Group
	for idx := range txNotifications {
		eg.Go(func() error {
			// 并发回查
			txNotification := txNotifications[idx]
			// 在此处发起回查，拿到结果
			txn := task.oneBackCheck(ctx, configMap, txNotification)
			switch txn.Status {
			case domain.TxNotificationStatusPrepare:
				// 查到还是 Prepare 状态
				_ = retryTxns.Append(txn)
			case domain.TxNotificationStatusCommit:
				_ = commitTxns.Append(txn)
			case domain.TxNotificationStatusCancel, domain.TxNotificationStatusFail:
				_ = failTxns.Append(txn)
			default:
				return errors.New("不合法的回查状态")
			}
			return nil
		})
	}

	err = eg.Wait()
	if err != nil {
		return err
	}
	// 分别处理，更新数据库状态
	// 数据库就可以一次性执行完，避免频繁更新数据库
	err = task.updateStatus(ctx, retryTxns, domain.SendStatusPrepare)
	err = multierr.Append(err, task.updateStatus(ctx, failTxns, domain.SendStatusFailed))
	// 转为 PENDING，后续 Scheduler 会调度执行
	err = multierr.Append(err, task.updateStatus(ctx, commitTxns, domain.SendStatusSending))
	return err
}

func (task *TxCheckTask) oneBackCheck(ctx context.Context, configMap map[int64]domain.BusinessConfig, txNotification domain.TxNotification) domain.TxNotification {
	bizConfig, ok := configMap[txNotification.BizID]
	if !ok || bizConfig.TxnConfig == nil {
		// 没设置，不需要回查
		txNotification.NextCheckTime = 0
		txNotification.Status = domain.TxNotificationStatusFail
		return txNotification
	}

	txConfig := bizConfig.TxnConfig
	// 发起回查
	res, err := task.getCheckBackRes(ctx, *txConfig, txNotification)
	// 执行了一次回查，要 +1
	txNotification.CheckCount++
	// 回查失败了
	if err != nil || res == unknownStatus {
		// 重新计算下一次的回查时间
		txNotification.SetNextCheckBackTimeAndStatus(txConfig)
		return txNotification
	}
	switch res {
	case cancelStatus:
		txNotification.NextCheckTime = 0
		txNotification.Status = domain.TxNotificationStatusCancel
	case committedStatus:
		txNotification.NextCheckTime = 0
		txNotification.Status = domain.TxNotificationStatusCommit
	}
	return txNotification
}

func (task *TxCheckTask) getCheckBackRes(ctx context.Context, txnConfig domain.TxnConfig, txn domain.TxNotification) (status int, err error) {
	defer func() {
		if r := recover(); r != nil {
			if str, ok := r.(string); ok {
				err = errors.New(str)
			} else {
				err = fmt.Errorf("未知panic类型: %v", r)
			}
		}
	}()
	// 借助服务发现来回查
	client := task.clients.Get(txnConfig.ServiceName)

	req := &clientv1.TransactionCheckServiceCheckRequest{Key: txn.Key}
	resp, err := client.Check(ctx, req)
	if err != nil {
		return unknownStatus, err
	}
	return int(resp.Status), nil
}

func (task *TxCheckTask) updateStatus(ctx context.Context, txns *list.ConcurrentList[domain.TxNotification], status domain.SendStatus) error {
	if txns.Len() == 0 {
		return nil
	}
	res := txns.AsSlice()
	return task.repo.UpdateCheckStatus(ctx, res, status)
}
