package repository

import (
	"context"
	"go-notification/internal/domain"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	BatchCreate(ctx context.Context, notifications []domain.Notification) (domain.Notification, error)
	BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) (domain.Notification, error)

	GetByID(ctx context.Context, id int64) (domain.Notification, error)
	BatchGetByID(ctx context.Context, ids []int64) (map[int64]domain.Notification, error)

	GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error)
	GetByKeys(ctx context.Context, keys []string) (map[int64]domain.Notification, error)

	CASStatus(ctx context.Context, notification domain.Notification) error
	UpdateStatus(ctx context.Context, notification domain.Notification) error

	BatchUpdateStatusSucceededOrFailed(ctx context.Context, succededNotifications, failedNotifications []domain.Notification) error

	FindReadNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error)
	MarkSuccess(ctx context.Context, notification domain.Notification) error
	MarkFailed(ctx context.Context, notification domain.Notification) error
	MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error)
}

type notificationRepository struct {
}

func (n notificationRepository) Create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) BatchCreate(ctx context.Context, notifications []domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) GetByID(ctx context.Context, id int64) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) BatchGetByID(ctx context.Context, ids []int64) (map[int64]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) GetByKeys(ctx context.Context, keys []string) (map[int64]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) CASStatus(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) UpdateStatus(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) BatchUpdateStatusSucceededOrFailed(ctx context.Context, succededNotifications, failedNotifications []domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) FindReadNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error) {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) MarkSuccess(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) MarkFailed(ctx context.Context, notification domain.Notification) error {
	//TODO implement me
	panic("implement me")
}

func (n notificationRepository) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	//TODO implement me
	panic("implement me")
}
