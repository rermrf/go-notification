package monitor

import (
	"context"
	"database/sql"
	"go-notification/internal/pkg/logger"
	"sync/atomic"
	"time"
)

const (
	timeout             = 5 * time.Second
	defaultFailCount    = 3
	defaultSuccessCount = 3
)

//go:generate mockgen -source=./monitor.go -package=monitormocks -destination=./mocks/monitor.mock.go DBMonitor
type DBMonitor interface {
	Health() bool
	// Report 上报数据库调用时的error，来收集调用时的错误
	Report(err error)
}

type Heartbeat struct {
	db             *sql.DB
	log            logger.Logger
	health         *atomic.Bool
	failCounter    *atomic.Int32 // 连续失败计数器
	successCounter *atomic.Int32 // 连续成功计数器（用于恢复）
}

func NewHeartbeatDBMonitor(db *sql.DB, log logger.Logger) *Heartbeat {
	he := &atomic.Bool{}
	he.Store(true)

	h := &Heartbeat{
		db:             db,
		log:            log,
		health:         he,
		failCounter:    &atomic.Int32{},
		successCounter: &atomic.Int32{},
	}
	go h.HealthCheck(context.Background())
	return h
}

func (h *Heartbeat) Health() bool {
	return h.health.Load()
}

func (h *Heartbeat) Report(err error) {

}

func (h *Heartbeat) HealthCheck(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			// 如果超时就返回
			if ctx.Err() != nil {
				h.log.Error("ctx 超时退出", logger.Error(ctx.Err()))
				return
			}
			return
		case <-ticker.C:
			// 执行健康检查
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			err := h.healthOneLoop(timeoutCtx)
			cancel()
			if err != nil {
				h.log.Error("ConnPool健康检查失败", logger.Error(err))
			}
		}
	}
}

func (h *Heartbeat) healthOneLoop(ctx context.Context) error {
	err := h.db.PingContext(ctx)
	if err != nil {
		// 失败时递增失败计数器，重置成功计数器
		h.successCounter.Store(0)
		if h.failCounter.Add(1) >= defaultFailCount {
			h.health.Store(false)
			h.failCounter.Store(0) // 重置计数器
		}
		return err
	}
	// 成功时递增成功计数器，重置失败计数器
	h.failCounter.Store(0)
	if h.successCounter.Add(1) >= defaultSuccessCount {
		h.health.Store(true)
		h.successCounter.Store(0) // 重置计数器
	}
	return nil
}
