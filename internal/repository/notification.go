package repository

import (
	"context"
	"encoding/json"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/repository/cache"
	"go-notification/internal/repository/dao"
	"time"
)

type NotificationRepository interface {
	Create(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error)
	BatchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error)
	BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error)

	GetByID(ctx context.Context, id int64) (domain.Notification, error)
	BatchGetByID(ctx context.Context, ids []int64) (map[int64]domain.Notification, error)

	GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error)
	GetByKeys(ctx context.Context, bizId int64, keys ...string) ([]domain.Notification, error)

	CASStatus(ctx context.Context, notification domain.Notification) error
	UpdateStatus(ctx context.Context, notification domain.Notification) error

	BatchUpdateStatusSucceededOrFailed(ctx context.Context, succeededNotifications, failedNotifications []domain.Notification) error

	FindReadNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error)
	MarkSuccess(ctx context.Context, notification domain.Notification) error
	MarkFailed(ctx context.Context, notification domain.Notification) error
	MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error)
}

const (
	defaultQuotaNumber int32 = 1
)

type notificationRepository struct {
	dao        dao.NotificationDAO
	quotaCache cache.QuotaCache
	logger     logger.Logger
}

func NewNotificationRepository(dao dao.NotificationDAO, quotaCache cache.QuotaCache, logger logger.Logger) NotificationRepository {
	return &notificationRepository{dao: dao, quotaCache: quotaCache, logger: logger}
}

// Create 创建单条通知记录，但不创建对应的回调记录
func (r *notificationRepository) Create(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	// 扣减额度
	err := r.quotaCache.Decr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
	if err != nil {
		return domain.Notification{}, err
	}
	ds, err := r.dao.Create(ctx, r.toEntity(notification))
	if err != nil {
		// 未创建成功归还额度
		qerr := r.quotaCache.Incr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
		if qerr != nil {
			r.logger.Error("额度归还失败", logger.Error(err), logger.Int64("biz_id", notification.BizID), logger.String("channel", notification.Channel.String()))
		}
		return domain.Notification{}, err
	}
	return r.toDomain(ds), nil
}

// CreateWithCallbackLog 创建单条通知记录，同时创建对应的回调记录
func (r *notificationRepository) CreateWithCallbackLog(ctx context.Context, notification domain.Notification) (domain.Notification, error) {
	// 扣减额度
	err := r.quotaCache.Decr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
	if err != nil {
		return domain.Notification{}, err
	}
	ds, err := r.dao.CreateWithCallbackLog(ctx, r.toEntity(notification))
	if err != nil {
		// 未创建成功归还额度
		qerr := r.quotaCache.Incr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
		if qerr != nil {
			r.logger.Error("额度归还失败", logger.Error(err), logger.Int64("biz_id", notification.BizID), logger.String("channel", notification.Channel.String()))
		}
		return domain.Notification{}, err
	}
	return r.toDomain(ds), nil
}

func (r *notificationRepository) BatchCreate(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	return r.batchCreate(ctx, notifications, false)
}

func (r *notificationRepository) batchCreate(ctx context.Context, notifications []domain.Notification, createCallbackLog bool) ([]domain.Notification, error) {
	if len(notifications) == 0 {
		return nil, nil
	}
	daoNotifications := make([]dao.Notification, 0, len(notifications))
	for _, notification := range notifications {
		daoNotifications = append(daoNotifications, r.toEntity(notification))
	}

	var createdNotifications []dao.Notification
	var err error
	// 扣件额度
	err = r.mutiDecr(ctx, notifications)
	if err != nil {
		return nil, err
	}
	if createCallbackLog {
		createdNotifications, err = r.dao.BatchCreateWithCallbackLog(ctx, daoNotifications)
		if err != nil {
			eerr := r.mutiIncr(ctx, notifications)
			if eerr != nil {
				r.logger.Error("发送失败，归还额度失败", logger.Error(eerr))
			}
			return nil, err
		}
	} else {
		createdNotifications, err = r.dao.BatchCreate(ctx, daoNotifications)
		if err != nil {
			eerr := r.mutiIncr(ctx, notifications)
			if eerr != nil {
				r.logger.Error("发送失败，归还额度失败", logger.Error(eerr))
				return nil, err
			}
		}
	}

	var result []domain.Notification
	for _, notification := range createdNotifications {
		result = append(result, r.toDomain(notification))
	}
	return result, nil
}

func (r *notificationRepository) BatchCreateWithCallbackLog(ctx context.Context, notifications []domain.Notification) ([]domain.Notification, error) {
	return r.batchCreate(ctx, notifications, true)
}

func (r *notificationRepository) GetByID(ctx context.Context, id int64) (domain.Notification, error) {
	n, err := r.dao.GetByID(ctx, id)
	if err != nil {
		return domain.Notification{}, err
	}
	return r.toDomain(n), nil
}

func (r *notificationRepository) BatchGetByID(ctx context.Context, ids []int64) (map[int64]domain.Notification, error) {
	notificationMap, err := r.dao.BatchGetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	domainNotificationMap := make(map[int64]domain.Notification, len(notificationMap))
	for id := range notificationMap {
		notification := notificationMap[id]
		domainNotificationMap[id] = r.toDomain(notification)
	}
	return domainNotificationMap, nil
}

func (r *notificationRepository) GetByKey(ctx context.Context, bizID int64, key string) (domain.Notification, error) {
	not, err := r.dao.GetByKey(ctx, bizID, key)
	return r.toDomain(not), err
}

// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
func (r *notificationRepository) GetByKeys(ctx context.Context, bizId int64, keys ...string) ([]domain.Notification, error) {
	notifications, err := r.dao.GetByKeys(ctx, bizId, keys...)
	if err != nil {
		return nil, fmt.Errorf("查询通知列表失败：%w", err)
	}
	result := make([]domain.Notification, len(notifications))
	for i := range notifications {
		result[i] = r.toDomain(notifications[i])
	}
	return result, nil
}

// CASStatus 更新通知状态
func (r *notificationRepository) CASStatus(ctx context.Context, notification domain.Notification) error {
	return r.dao.CASStatus(ctx, r.toEntity(notification))
}

func (r *notificationRepository) UpdateStatus(ctx context.Context, notification domain.Notification) error {
	return r.dao.UpdateStatus(ctx, r.toEntity(notification))
}

// BatchUpdateStatusSucceededOrFailed 批量更新通知状态为成功或失败
func (r *notificationRepository) BatchUpdateStatusSucceededOrFailed(ctx context.Context, succeededNotifications, failedNotifications []domain.Notification) error {
	successItems := make([]dao.Notification, 0, len(succeededNotifications))
	for _, notification := range succeededNotifications {
		successItems = append(successItems, r.toEntity(notification))
	}

	failedItems := make([]dao.Notification, 0, len(failedNotifications))
	for _, notification := range failedNotifications {
		failedItems = append(failedItems, r.toEntity(notification))
	}

	err := r.dao.BatchUpdateStatusSucceedOrFailed(ctx, successItems, failedItems)
	if err != nil {
		return err
	}

	items := r.getItems(failedNotifications)
	eerr := r.quotaCache.MutiIncr(ctx, items)
	if eerr != nil {
		r.logger.Error("发送失败，归还额度失败", logger.Error(eerr))
	}
	return nil
}

func (r *notificationRepository) FindReadNotifications(ctx context.Context, offset, limit int) ([]domain.Notification, error) {
	nos, err := r.dao.FindReadyNotifications(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	result := make([]domain.Notification, 0, len(nos))
	for i := range nos {
		result = append(result, r.toDomain(nos[i]))
	}
	return result, nil
}

func (r *notificationRepository) MarkSuccess(ctx context.Context, notification domain.Notification) error {
	return r.dao.MarkSuccess(ctx, r.toEntity(notification))
}

func (r *notificationRepository) MarkFailed(ctx context.Context, notification domain.Notification) error {
	err := r.dao.MarkFailed(ctx, r.toEntity(notification))
	if err != nil {
		return err
	}
	return r.quotaCache.Incr(ctx, notification.BizID, notification.Channel, defaultQuotaNumber)
}

func (r *notificationRepository) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	return r.dao.MarkTimeoutSendingAsFailed(ctx, batchSize)
}

func (r *notificationRepository) toEntity(notification domain.Notification) dao.Notification {
	templateParams, _ := notification.MarshalTemplateParms()
	receivers, _ := notification.MarshalReceivers()
	return dao.Notification{
		ID:                notification.ID,
		BizID:             notification.BizID,
		Key:               notification.Key,
		Receivers:         receivers,
		Channel:           notification.Channel.String(),
		TemplateID:        notification.Template.ID,
		TemplateVersionID: notification.Template.VersionID,
		TemplateParams:    templateParams,
		Status:            notification.Status.String(),
		ScheduledSTime:    notification.ScheduledSTime.UnixMilli(),
		ScheduledETime:    notification.ScheduledETime.UnixMilli(),
		Version:           notification.Version,
	}
}

func (r *notificationRepository) toDomain(n dao.Notification) domain.Notification {
	var templateParams map[string]string
	_ = json.Unmarshal([]byte(n.TemplateParams), &templateParams)

	var receivers []string
	_ = json.Unmarshal([]byte(n.Receivers), &receivers)

	return domain.Notification{
		ID:        n.ID,
		BizID:     n.BizID,
		Key:       n.Key,
		Receivers: receivers,
		Channel:   domain.Channel(n.Channel),
		Template: domain.Template{
			ID:        n.TemplateID,
			VersionID: n.TemplateVersionID,
			Params:    templateParams,
		},
		Status:         domain.SendStatus(n.Status),
		ScheduledSTime: time.UnixMilli(n.ScheduledSTime),
		ScheduledETime: time.UnixMilli(n.ScheduledETime),
		Version:        n.Version,
	}
}

func (r *notificationRepository) mutiDecr(ctx context.Context, notifications []domain.Notification) error {
	return r.quotaCache.MutiDecr(ctx, r.getItems(notifications))
}

func (r *notificationRepository) mutiIncr(ctx context.Context, notifications []domain.Notification) error {
	return r.quotaCache.MutiDecr(ctx, r.getItems(notifications))
}

func (r *notificationRepository) getItems(notifications []domain.Notification) []cache.IncrItem {
	notiMap := make(map[string]cache.IncrItem)
	for idx := range notifications {
		d := notifications[idx]
		key := fmt.Sprintf("%d-%s", d.BizID, d.Channel.String())
		item, ok := notiMap[key]
		if !ok {
			item = cache.IncrItem{
				BizID:   d.BizID,
				Channel: d.Channel,
			}
		}
		item.Val++
		notiMap[key] = item
	}
	items := make([]cache.IncrItem, 0, len(notiMap))
	for key := range notiMap {
		items = append(items, notiMap[key])
	}
	return items
}
