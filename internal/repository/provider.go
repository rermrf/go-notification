package repository

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/repository/dao"
	"gorm.io/gorm"
)

// ProviderRepository 供应商存储接口
type ProviderRepository interface {
	// Create 创建供应商
	Create(ctx context.Context, provider domain.Provider) (domain.Provider, error)
	// Update 更新供应商
	Update(ctx context.Context, provider domain.Provider) error
	FindByID(ctx context.Context, id int64) (domain.Provider, error)
	FindByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error)
}

type providerRepository struct {
	dao dao.ProviderDAO
}

func NewProviderRepository(dao dao.ProviderDAO) ProviderRepository {
	return &providerRepository{dao: dao}
}

func (p *providerRepository) Create(ctx context.Context, provider domain.Provider) (domain.Provider, error) {
	created, err := p.dao.Create(ctx, p.toEntity(provider))
	if err != nil {
		return domain.Provider{}, err
	}
	return p.toDomain(created), nil
}

func (p *providerRepository) Update(ctx context.Context, provider domain.Provider) error {
	return p.dao.Update(ctx, p.toEntity(provider))
}

func (p *providerRepository) FindByID(ctx context.Context, id int64) (domain.Provider, error) {
	provider, err := p.dao.FindByID(ctx, id)
	if err != nil {
		// 未找到的情况，转换为领域错误
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Provider{}, fmt.Errorf("%w", errs.ErrProviderNotFound)
		}
		return domain.Provider{}, err
	}
	return p.toDomain(provider), nil
}

func (p *providerRepository) FindByChannel(ctx context.Context, channel domain.Channel) ([]domain.Provider, error) {
	providers, err := p.dao.FindByChannel(ctx, channel.String())
	if err != nil {
		return nil, err
	}
	result := make([]domain.Provider, 0, len(providers))
	for _, provider := range providers {
		result = append(result, p.toDomain(provider))
	}
	return result, nil
}

func (p *providerRepository) toEntity(provider domain.Provider) dao.Provider {
	return dao.Provider{
		ID:               provider.ID,
		Name:             provider.Name,
		Channel:          provider.Channel.String(),
		Endpoint:         provider.Endpoint,
		APIKey:           provider.APIKey,
		APISecret:        provider.APISecret,
		Weight:           provider.Weight,
		QPSLimit:         provider.QPSLimit,
		DailyLimit:       provider.DailyLimit,
		AuditCallbackURL: provider.AuditCallbackURL,
		Status:           provider.Status.String(),
	}
}

func (p *providerRepository) toDomain(provider dao.Provider) domain.Provider {
	return domain.Provider{
		ID:               provider.ID,
		Name:             provider.Name,
		Channel:          domain.Channel(provider.Channel),
		Endpoint:         provider.Endpoint,
		APIKey:           provider.APIKey,
		APISecret:        provider.APISecret,
		Weight:           provider.Weight,
		QPSLimit:         provider.QPSLimit,
		DailyLimit:       provider.DailyLimit,
		AuditCallbackURL: provider.AuditCallbackURL,
		Status:           domain.ProviderStatus(provider.Status),
	}
}
