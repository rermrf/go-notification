package quota

import (
	"context"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository"
	"time"
)

// MonthlyResetCron 每月重置配额
type MonthlyResetCron struct {
	bizRepo   repository.BusinessConfigRepository
	svc       Service
	batchSize int
	log       logger.Logger
}

func NewMonthlyResetCron(bizRepo repository.BusinessConfigRepository, svc Service, log logger.Logger) *MonthlyResetCron {
	const batchSize = 10
	return &MonthlyResetCron{
		bizRepo:   bizRepo,
		svc:       svc,
		log:       log,
		batchSize: batchSize,
	}
}

func (m *MonthlyResetCron) Start(ctx context.Context) error {
	offset := 0
	for {
		const loopTimeOut = time.Second * 15
		ctx, cancel := context.WithTimeout(ctx, loopTimeOut)
		cnt, err := m.oneLoop(ctx, offset)
		cancel()
		if err != nil {
			m.log.Error("查找 Biz 配置失败", logger.Error(err))
			// 继续尝试下一批
			offset += m.batchSize
			continue
		}

		if cnt < m.batchSize {
			return nil
		}
		offset += cnt
	}
}

func (m *MonthlyResetCron) oneLoop(ctx context.Context, offset int) (int, error) {
	const findTimeout = time.Second * 3
	ctx, cancel := context.WithTimeout(ctx, findTimeout)
	defer cancel()
	bizs, err := m.bizRepo.Find(ctx, offset, 0)
	if err != nil {
		return 0, err
	}
	const resetTimeout = time.Second * 10
	ctx, cancel = context.WithTimeout(ctx, resetTimeout)
	defer cancel()
	for _, cfg := range bizs {
		err = m.svc.ResetQuota(ctx, cfg)
		if err != nil {
			continue
		}
	}
	return len(bizs), nil
}
