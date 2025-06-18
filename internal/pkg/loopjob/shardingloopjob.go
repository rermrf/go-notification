package loopjob

import (
	"context"
	"errors"
	"fmt"
	"github.com/meoying/dlock-go"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/sharding"
	"time"
)

type CtxKey string

type ShardingLoopJob struct {
	shardingStrategy  sharding.ShardingStrategy
	baseKey           string // 业务标识
	dclient           dlock.Client
	log               logger.Logger
	biz               func(ctx context.Context) error
	retryInterval     time.Duration
	defaultTimeout    time.Duration
	resourceSemaphore ResourceSemaphore
}

type ShardingLoopJobOption func(*ShardingLoopJob)

func NewShardingLoopJob(dclient dlock.Client, baseKey string, biz func(ctx context.Context) error, shardingStrategy sharding.ShardingStrategy, resourceSemaphore ResourceSemaphore, log logger.Logger) *ShardingLoopJob {
	const defaultTimeout = 3 * time.Second
	return newShardingLoopJobLoop(dclient, baseKey, biz, shardingStrategy, time.Minute, defaultTimeout, resourceSemaphore, log)
}

// newShardingLoopJobLoop 用于创建一个ShardingLoopJobLoop 实例，允许指定重试间隔，便于测试
func newShardingLoopJobLoop(
	dclient dlock.Client,
	baseKey string,
	biz func(ctx context.Context) error,
	shardingStrategy sharding.ShardingStrategy,
	retryInterval time.Duration,
	defaultTimeout time.Duration,
	resourceSemaphore ResourceSemaphore,
	log logger.Logger,
) *ShardingLoopJob {
	return &ShardingLoopJob{
		dclient:           dclient,
		baseKey:           baseKey,
		shardingStrategy:  shardingStrategy,
		log:               log,
		biz:               biz,
		retryInterval:     retryInterval,
		defaultTimeout:    defaultTimeout,
		resourceSemaphore: resourceSemaphore,
	}
}

func (l *ShardingLoopJob) generateKey(db, table string) string {
	return fmt.Sprintf("%s:%s:%s", l.baseKey, db, table)
}

func (l *ShardingLoopJob) Run(ctx context.Context) {
	for {
		for _, dst := range l.shardingStrategy.Broadcast() {
			// 超过允许强占的上限了
			err := l.resourceSemaphore.Acquire(ctx)
			if err != nil {
				time.Sleep(l.retryInterval)
				continue
			}

			key := l.generateKey(dst.DB, dst.Table)
			// 抢锁
			lock, err := l.dclient.NewLock(ctx, key, l.retryInterval)
			if err != nil {
				l.log.Error("初始化分布式锁失败，重试", logger.Error(err))
				err = l.resourceSemaphore.Release(ctx)
				if err != nil {
					l.log.Error("释放表的信号量失败", logger.Error(err))
					continue
				}

				lockCtx, cancel := context.WithTimeout(ctx, l.defaultTimeout)
				// 还没拿到锁，不管是系统错误，还是锁被人持有，都没关系
				// 暂停一段时间之后继续
				err = lock.Lock(lockCtx)
				cancel()
				if err != nil {
					l.log.Error("没有抢到分布式锁，系统出现问题", logger.Error(err))
					err = l.resourceSemaphore.Release(ctx)
					if err != nil {
						l.log.Error("释放表的信号量失败", logger.Error(err))
					}
					continue
				}
				// 抢占成功
				go l.tableLoop(sharding.CtxWithDst(ctx, dst), lock)
			}
		}
	}
}

func (l *ShardingLoopJob) tableLoop(ctx context.Context, lock dlock.Lock) {
	defer func() {
		_ = l.resourceSemaphore.Release(ctx)
	}()
	// 在这里执行业务
	err := l.bizLoop(ctx, lock)
	// 要么续约失败，要么是 ctx 本身已经过期了
	if err != nil {
		l.log.Error("执行业务失败，将执行重试", logger.Error(err))
	}
	// 不管什么原因，都要考虑释放分布式锁了
	// 要稍微摆脱 ctx 的控制，因为此时 ctx 可能被取消了
	unCtx, cancel := context.WithTimeout(context.Background(), l.defaultTimeout)
	unErr := lock.Unlock(unCtx)
	cancel()
	if unErr != nil {
		l.log.Error("释放分布式锁失败", logger.Error(unErr))
	}
	err = ctx.Err()
	switch {
	case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded):
		// 被取消，那么就要跳出循环
		l.log.Info("任务被取消，推出任务循环")
		return
	default:
		l.log.Error("执行任务失败，将执行重试", logger.Error(err))
		time.Sleep(l.retryInterval)
	}
}

func (l *ShardingLoopJob) bizLoop(ctx context.Context, lock dlock.Lock) error {
	const bizTimeout = 50 * time.Second
	for {
		// 可以确保业务在分布式锁过期之前结束
		bizCtx, cancel := context.WithTimeout(ctx, bizTimeout)
		err := l.biz(bizCtx)
		cancel()
		if err != nil {
			l.log.Error("业务执行失败", logger.Error(err))
		}
		if ctx.Err() != nil {
			return ctx.Err()
		}
		redCtx, cancel := context.WithTimeout(ctx, l.defaultTimeout)
		err = lock.Refresh(redCtx)
		cancel()
		if err != nil {
			l.log.Error("分布式锁续约失败", logger.Error(err))
			return fmt.Errorf("分布式锁续约失败 %w", err)
		}
	}
}
