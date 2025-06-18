package loopjob

import (
	"context"
	"go-notification/internal/errs"
	"sync"
)

// ResourceSemaphore 信号量，控制抢占资源的最大信号量
type ResourceSemaphore interface {
	Acquire(ctx context.Context) error
	Release(ctx context.Context) error
}

type MaxCntResourceSemaphore struct {
	maxCount     int
	currentCount int
	mu           *sync.Mutex
}

func NewMaxCntResourceSemaphore(maxCount int) *MaxCntResourceSemaphore {
	return &MaxCntResourceSemaphore{
		maxCount:     maxCount,
		currentCount: 0,
		mu:           new(sync.Mutex),
	}
}

func (m *MaxCntResourceSemaphore) Acquire(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.currentCount >= m.maxCount {
		return errs.ErrExceedLimit
	}
	m.currentCount++
	return nil
}

func (m *MaxCntResourceSemaphore) Release(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.currentCount--
	return nil
}

func (m *MaxCntResourceSemaphore) UpdateMaxCount(maxCount int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.maxCount = maxCount
}
