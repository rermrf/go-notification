package strategy

import (
	"math"
	"sync/atomic"
	"time"
)

var _ Strategy = (*ExponentialBackoffRetryStrategy)(nil)

// ExponentialBackoffRetryStrategy 指数退避重试策略
type ExponentialBackoffRetryStrategy struct {
	// 初始重试间隔
	initialInterval time.Duration
	// 最大重试间隔
	maxInterval time.Duration
	// 最大重试次数
	maxRetries int32
	// 当前重试次数
	retries int32
	// 是否已经达到最大重试间隔
	maxIntervalReached atomic.Value
}

func NewExponentialBackoffRetryStrategy(initialInterval time.Duration, maxInterval time.Duration, maxRetries int32) *ExponentialBackoffRetryStrategy {
	return &ExponentialBackoffRetryStrategy{initialInterval: initialInterval, maxInterval: maxInterval, maxRetries: maxRetries}
}

func (e *ExponentialBackoffRetryStrategy) NextWithRetries(retries int32) (time.Duration, bool) {
	return e.nextWithRetries(retries)
}

func (e *ExponentialBackoffRetryStrategy) nextWithRetries(retries int32) (time.Duration, bool) {
	if e.maxRetries <= 0 || retries <= e.maxRetries {
		if reached, ok := e.maxIntervalReached.Load().(bool); ok && reached {
			return e.maxInterval, true
		}
		const two = 2
		interval := e.initialInterval * time.Duration(math.Pow(two, float64(retries-1)))
		// 溢出或当前重试间隔大于最大重试间隔
		if interval <= 0 || interval > e.maxInterval {
			e.maxIntervalReached.Store(true)
			return e.maxInterval, true
		}
		return interval, true
	}
	return 0, false
}

// Next 返回下一次重试的间隔，如果不需要继续重试，那么第二参数返回 false
func (e *ExponentialBackoffRetryStrategy) Next() (time.Duration, bool) {
	retries := atomic.AddInt32(&e.retries, 1)
	return e.nextWithRetries(retries)
}

func (e *ExponentialBackoffRetryStrategy) Report(_ error) Strategy {
	return e
}
