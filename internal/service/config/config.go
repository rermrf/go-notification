package config

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/repository"
	"gorm.io/gorm"
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

func NewBusinessConfigService(repo repository.BusinessConfigRepository) BusinessConfigService {
	return &businessConfigService{repo: repo}
}

func (b *businessConfigService) GetByIDs(ctx context.Context, ids []int64) (map[int64]domain.BusinessConfig, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	return b.repo.GetByIDs(ctx, ids)
}

func (b *businessConfigService) GetByID(ctx context.Context, id int64) (domain.BusinessConfig, error) {
	if id <= 0 {
		return domain.BusinessConfig{}, fmt.Errorf("%w", errs.ErrInvalidParameter)
	}

	config, err := b.repo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.BusinessConfig{}, fmt.Errorf("%w", errs.ErrConfigNotFound)
		}
		return domain.BusinessConfig{}, err
	}

	return config, nil
}

func (b *businessConfigService) DeleteByID(ctx context.Context, id int64) error {
	if id <= 0 {
		return fmt.Errorf("%w", errs.ErrInvalidParameter)
	}
	return b.repo.DeleteByID(ctx, id)
}

func (b *businessConfigService) SaveConfig(ctx context.Context, config domain.BusinessConfig) error {
	if config.ID <= 0 {
		return fmt.Errorf("%w", errs.ErrInvalidParameter)
	}
	return b.repo.SaveConfig(ctx, config)
}
