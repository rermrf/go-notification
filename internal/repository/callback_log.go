package repository

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/repository/dao"
)

type CallbackLogRepository interface {
	Find(ctx context.Context, startTime, batchSize, startID int64) (logs []domain.CallbackLog, nextStartID int64, err error)
	Update(ctx context.Context, logs []domain.CallbackLog) error
	FindByNotificationIDs(ctx context.Context, notificationIDs []int64) ([]domain.CallbackLog, error)
}

type callbackLogRepository struct {
	notificationRepo NotificationRepository
	dao              dao.CallbackLogDAO
}

func NewCallbackLogRepository(notificationRepo NotificationRepository, dao dao.CallbackLogDAO) CallbackLogRepository {
	return &callbackLogRepository{notificationRepo: notificationRepo, dao: dao}
}

func (c callbackLogRepository) Find(ctx context.Context, startTime, batchSize, startID int64) (logs []domain.CallbackLog, nextStartID int64, err error) {
	entities, nextStartID, err := c.dao.Find(ctx, startTime, batchSize, startID)
	if err != nil {
		return nil, 0, err
	}

	if int64(len(entities)) < batchSize {
		nextStartID = 0
	}
	var result []domain.CallbackLog
	for _, entity := range entities {
		n, _ := c.notificationRepo.GetByID(ctx, entity.NotificationID)
		result = append(result, c.toDomain(entity, n))
	}
	return result, nextStartID, nil
}

func (c callbackLogRepository) Update(ctx context.Context, logs []domain.CallbackLog) error {
	entities := make([]dao.CallbackLog, 0, len(logs))
	for _, entity := range logs {
		entities = append(entities, c.toEntity(entity))
	}
	return c.dao.Update(ctx, entities)
}

func (c callbackLogRepository) FindByNotificationIDs(ctx context.Context, notificationIDs []int64) ([]domain.CallbackLog, error) {
	logs, err := c.dao.FindByNotificationIDs(ctx, notificationIDs)
	if err != nil {
		return nil, err
	}
	ns, err := c.notificationRepo.BatchGetByID(ctx, notificationIDs)
	if err != nil {
		return nil, err
	}
	result := make([]domain.CallbackLog, 0, len(logs))
	for _, entity := range logs {
		result = append(result, c.toDomain(entity, ns[entity.NotificationID]))
	}
	return result, nil
}

func (c callbackLogRepository) toDomain(log dao.CallbackLog, notification domain.Notification) domain.CallbackLog {
	return domain.CallbackLog{
		ID:            log.ID,
		Notification:  notification,
		RetryCount:    log.RetryCount,
		NextRetryTime: log.NextRetryTime,
		Status:        domain.CallbackLogStatus(log.Status),
	}
}

func (c callbackLogRepository) toEntity(log domain.CallbackLog) dao.CallbackLog {
	return dao.CallbackLog{
		ID:             log.ID,
		NotificationID: log.Notification.ID,
		RetryCount:     log.RetryCount,
		NextRetryTime:  log.NextRetryTime,
		Status:         log.Status.String(),
	}
}
