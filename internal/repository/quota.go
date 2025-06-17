package repository

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/repository/dao"
)

type QuotaRepository interface {
	CreateOrUpdate(ctx context.Context, quota ...domain.Quota) error
	Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error)
}

type quotaRepository struct {
	dao dao.QuotaDAO
}

func NewQuotaRepository(dao dao.QuotaDAO) QuotaRepository {
	return &quotaRepository{dao: dao}
}

func (r *quotaRepository) CreateOrUpdate(ctx context.Context, quota ...domain.Quota) error {
	qs := make([]dao.Quota, 0, len(quota))
	for _, q := range quota {
		qs = append(qs, dao.Quota{
			Quota:   q.Quota,
			BizID:   q.BizID,
			Channel: q.Channel.String(),
		})
	}
	return r.dao.CreateOrUpdate(ctx, qs...)
}

func (r *quotaRepository) Find(ctx context.Context, bizID int64, channel domain.Channel) (domain.Quota, error) {
	found, err := r.dao.Find(ctx, bizID, channel.String())
	if err != nil {
		return domain.Quota{}, err
	}
	return domain.Quota{
		BizID:   found.BizID,
		Quota:   found.Quota,
		Channel: domain.Channel(found.Channel),
	}, nil
}
