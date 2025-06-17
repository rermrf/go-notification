package quota

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/repository"
)

type Service interface {
	ResetQuota(ctx context.Context, biz domain.BusinessConfig) error
}

type service struct {
	repo repository.QuotaRepository
}

func NewService(repo repository.QuotaRepository) Service {
	return &service{repo: repo}
}

func (s service) ResetQuota(ctx context.Context, biz domain.BusinessConfig) error {
	if biz.Quota == nil {
		return errs.ErrNoQuota
	}
	sms := domain.Quota{
		BizID:   biz.ID,
		Quota:   int32(biz.Quota.Monthly.SMS),
		Channel: domain.ChannelSMS,
	}
	email := domain.Quota{
		BizID:   biz.ID,
		Quota:   int32(biz.Quota.Monthly.EMAIL),
		Channel: domain.ChannelEmail,
	}
	return s.repo.CreateOrUpdate(ctx, sms, email)
}
