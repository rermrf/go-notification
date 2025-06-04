package repository

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
)

type BusinessConfigRepository interface {
	LoadCache(ctx context.Context) error
	GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error)
	DeleteByID(ctx context.Context, id int64) error
	SaveConfig(ctx context.Context, config domain.BusinessConfig) error
	Find(ctx context.Context, offset, limit int) ([]domain.BusinessConfig, error)
}

type businessConfigRepository struct {
	dao        dao.BusinessConfigDAO
	localCache cache.ConfigCache
	redisCache cache.ConfigCache
	logger     logger.Logger
}
