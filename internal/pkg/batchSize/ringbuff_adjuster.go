package batchSize

import (
	"context"
	"go-notification/internal/pkg/ringbuffer"
	"sync"
	"time"
)

type RingBufferAdjuster struct {
	mutex          *sync.Mutex
	timeBuffer     *ringbuffer.TimeDurationRingBuffer // 历史执行时间的环形缓冲区
	batchSize      int                                // 当前批次大小
	minBatchSize   int                                // 最小批次大小
	maxBatchSize   int                                // 最大批次大小
	adjustStep     int                                // 每次调整的步长（增加或减少）
	cooldownPeriod time.Duration                      // 调整后的冷却期
	lastAdjustTime time.Time                          // 上次调整时间
}

// NewRingBufferAdjuster 创建基于环形缓冲区的批大小调整器
// initialSize: 初始批大小（基准值）
func NewRingBufferAdjuster(initialSize int, minSize int, maxSize int, adjustStep int, cooldownPeriod time.Duration, bufferSize int) *RingBufferAdjuster {
	if bufferSize <= 0 {
		bufferSize = 128 // 默认维护128条历史记录
	}
	if initialSize < minSize {
		initialSize = minSize
	}
	if initialSize > maxSize {
		initialSize = maxSize
	}

	timeBuffer, _ := ringbuffer.NewTimeDurationRingBuffer(bufferSize)

	return &RingBufferAdjuster{
		mutex:          &sync.Mutex{},
		timeBuffer:     timeBuffer,
		batchSize:      initialSize,
		minBatchSize:   minSize,
		maxBatchSize:   maxSize,
		adjustStep:     adjustStep,
		cooldownPeriod: cooldownPeriod,
		lastAdjustTime: time.Time{}, // 零值时间，初始允许立即调整
	}
}

// Adjust 根据响应时间动态调整批次大小
// 1. 记录响应时间到环形缓冲区
// 2. 如果当前时间比平均时间长，且不在冷却期，则减少批大小
// 3. 如果响应时间比平均时间短，且不在冷切期，则增加批大小
func (r *RingBufferAdjuster) Adjust(ctx context.Context, responseTime time.Duration) (int, error) {
	r.mutex.Lock()
	defer r.mutex.Unlock()
	// 记录当前响应时间到环形缓冲区
	r.timeBuffer.Add(responseTime)

	// 至少需要收集一轮才能开始调整
	if r.timeBuffer.Len() < r.timeBuffer.Cap() {
		return r.batchSize, nil
	}

	// 如果处于冷却期内，不调整大小
	if !r.lastAdjustTime.IsZero() && time.Since(r.lastAdjustTime) < r.cooldownPeriod {
		return r.batchSize, nil
	}

	// 平均时间
	avgTime := r.timeBuffer.Avg()

	// 根据响应时间调整大小
	if responseTime > avgTime {
		// 响应时间高于平均时间，减少批次
		if r.batchSize > r.minBatchSize {
			r.batchSize = max(r.batchSize-r.adjustStep, r.minBatchSize)
			r.lastAdjustTime = time.Now() // 更新时间戳
		}
	} else if responseTime < avgTime {
		// 响应时间低于平均值，增加批大小
		if r.batchSize < r.maxBatchSize {
			r.batchSize = min(r.batchSize+r.adjustStep, r.maxBatchSize)
			r.lastAdjustTime = time.Now()
		}
	}
	// 响应时间等于平均，不调整
	// 可以新增一个阈值，不超过或不低于多少则不调整

	return r.batchSize, nil
}
