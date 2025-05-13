package notification

import (
	"context"
	"go-notification/internal/domain"
)

//go:generate mockgen -source=./notification.go -destination=./mocks/notification.mock.go -package=notificationmocks -typed Service
type Service interface {
	FindReadyNotifications(ctx context.Context, offiset, limit int) ([]domain.Notification, error)
	GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error)
}

type notificationService struct {
}

func (n notificationService) FindReadyNotifications(ctx context.Context, offiset, limit int) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationService) GetByKeys(ctx context.Context, bizID int64, keys ...string) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}
