package scheduler

import (
	"context"
	"github.com/meoying/dlock-go"
	"go-notification/internal/errs"
	"go-notification/internal/pkg/batchSize"
	"go-notification/internal/pkg/bitring"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/loopjob"
	"go-notification/internal/pkg/sharding"
	"go-notification/internal/repository"
	"go-notification/internal/service/sender"
	"sync/atomic"
	"time"
)

// ShardingScheduler 通知调度服务实现
type ShardingScheduler struct {
	repo              repository.NotificationRepository
	sender            sender.NotificationSender
	minLoopDuration   time.Duration
	batchSize         atomic.Uint64
	batchSizeAdjuster batchSize.Adjuster

	errorEvents *bitring.BitRing
	job         *loopjob.ShardingLoopJob
}

// NewShardingScheduler 创建通知调度服务
func NewShardingScheduler(
	repo repository.NotificationRepository,
	sender sender.NotificationSender,
	dclient dlock.Client,
	shardingStrategy sharding.ShardingStrategy,
	sem loopjob.ResourceSemaphore,
	minLoopDuration time.Duration,
	batchSize int64,
	batchSizeAdjuster batchSize.Adjuster,
	errorEvents *bitring.BitRing,
	log logger.Logger,
) *ShardingScheduler {
	const key = "go_notification_platform_async_sharding_scheduler"
	s := &ShardingScheduler{
		repo:              repo,
		sender:            sender,
		minLoopDuration:   minLoopDuration,
		batchSizeAdjuster: batchSizeAdjuster,
		errorEvents:       errorEvents,
	}
	s.job = loopjob.NewShardingLoopJob(dclient, key, s.loop, shardingStrategy, sem, log)
	s.batchSize.Store(uint64(batchSize))
	return s
}

// Start 启动调度服务
// 当 ctx 被取消或者关闭的时候，就会结束循环
func (s *ShardingScheduler) Start(ctx context.Context) {
	go s.job.Run(ctx)
}

func (s *ShardingScheduler) loop(ctx context.Context) error {
	for {
		// 纪录开始执行时间
		start := time.Now()

		// 批量发送已就绪的通知
		cnt, err := s.batchSendReadyNotifications(ctx)

		// 记录响应时间
		responseTime := time.Since(start)

		// 记录错误事件
		s.errorEvents.Add(err != nil)
		// 判断错误事件是否已达到预设的条件 - 连续出现三次错误，或者错误率达到阈值
		if s.errorEvents.IsConditionMet() {
			return errs.ErrErrorConditionIsMet
		}

		// 根据响应时间调整 batchSize
		newBatchSize, err1 := s.batchSizeAdjuster.Adjust(ctx, responseTime)
		if err1 == nil {
			s.batchSize.Store(uint64(newBatchSize))
		}

		// 没有数据时，响应非常快，需要等待一段时间
		if cnt == 0 {
			time.Sleep(s.minLoopDuration - responseTime)
			continue
		}
	}
}

// batchSendReadyNotifications 批量发送已就绪的通知
func (s *ShardingScheduler) batchSendReadyNotifications(ctx context.Context) (int, error) {
	const defaultTimeout = 3 * time.Second

	loopCtx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()

	const offset = 0
	notifications, err := s.repo.FindReadNotifications(loopCtx, offset, int(s.batchSize.Load()))
	if err != nil {
		return 0, err
	}

	if len(notifications) == 0 {
		return 0, nil
	}

	_, err = s.sender.BatchSend(ctx, notifications)
	return len(notifications), err
}
