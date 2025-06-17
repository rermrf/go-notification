package scheduler

import (
	"go-notification/internal/pkg/batchSize"
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
	job         *loopjob.S
}
