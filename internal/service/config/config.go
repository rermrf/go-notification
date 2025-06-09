package config

import (
	"context"
	"errors"
	"go-notification/internal/domain"
	"go-notification/internal/repository"
)

var ErrIDNotSet = errors.New("业务id没有设置")

type BusinessConfigService interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error)
	DeleteByID(ctx context.Context, id int64) error
	// SaveConfig 保存非零字段
	SaveConfig(ctx context.Context, config domain.BusinessConfig) error
}

type businessConfigService struct {
	repo repository.BusinessConfigRepository
}
