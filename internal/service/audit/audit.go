package audit

import (
	"context"
	"go-notification/internal/domain"
)

type Service interface {
	CreateAudit(ctx context.Context, req domain.Audit) (int64, error)
}

type service struct{}

func (s service) CreateAudit(ctx context.Context, req domain.Audit) (int64, error) {
	//TODO implement me
	panic("implement me")
}
