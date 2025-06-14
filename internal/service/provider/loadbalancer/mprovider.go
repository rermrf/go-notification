package loadbalancer

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/service/provider"
	"math/bits"
	"sync"
	"sync/atomic"
	"time"
)

const (
	defaultNumberLen   = 64
	defaultFailPercent = 0.1

	bitsPeruint6       = 64
	bitsPerUint64Shift = 6
	btiMask            = 64
	initialHealth      = true
	recoverSecond      = 3
)

type mprovider struct {
	provider.Provider
	healthy       *atomic.Bool
	ringBuffer    []uint64 // 比特环（滑动窗口存储）
	reqCount      uint64   // 请求数量
	bufferLen     int      // 滑动窗口长度
	bitCnt        uint64   // 比特位总数
	failThreshold int
	mu            *sync.RWMutex
}

func newMprovider(provider provider.Provider, bufferLen int) *mprovider {
	health := &atomic.Bool{}
	health.Store(initialHealth)
	bitCnt := uint64(bufferLen) * uint64(defaultNumberLen)
	return &mprovider{
		Provider:      provider,
		healthy:       health,
		bufferLen:     bufferLen,
		ringBuffer:    make([]uint64, bufferLen),
		bitCnt:        bitCnt,
		mu:            &sync.RWMutex{},
		failThreshold: int(float64(bitCnt) * defaultFailPercent),
	}
}

func (m *mprovider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	res, err := m.Provider.Send(ctx, notification)
	if err != nil {
		m.markFailed()
		v := m.getFailed()
		if v > m.failThreshold {
			if m.healthy.CompareAndSwap(true, false) {
				const waitTime = time.Minute
				time.AfterFunc(waitTime, func() {
					m.healthy.Store(true)
					m.mu.Lock()
					m.ringBuffer = make([]uint64, m.bufferLen)
					m.mu.Unlock()
				})
			}
		}
	} else {
		m.markSuccess()
	}
	return res, err
}

func (m *mprovider) markFailed() {
	count := atomic.AddUint64(&m.reqCount, 1)
	count %= m.bitCnt
	idx := count >> bitsPerUint64Shift
	bitPos := count & btiMask
	old := atomic.LoadUint64(&m.ringBuffer[idx])
	// (uint64(1)<<bitPos) 将目标位设置为1
	atomic.StoreUint64(&m.ringBuffer[idx], old|(uint64(1)<<bitPos))
}

func (m *mprovider) getFailed() int {
	var failCount int
	for i := 0; i < len(m.ringBuffer); i++ {
		v := atomic.LoadUint64(&m.ringBuffer[i])
		failCount += bits.OnesCount64(v)
	}
	return failCount
}

func (m *mprovider) markSuccess() {
	count := atomic.AddUint64(&m.reqCount, 1)
	count %= m.bitCnt
	idx := count >> bitsPerUint64Shift
	bitPos := count & btiMask
	old := atomic.LoadUint64(&m.ringBuffer[idx])
	atomic.StoreUint64(&m.ringBuffer[idx], old|(uint64(1)<<bitPos))
}

func (m *mprovider) isHealthy() bool {
	return m.healthy.Load()
}
