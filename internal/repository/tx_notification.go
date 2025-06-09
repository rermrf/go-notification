package repository

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/repository/dao"
)

type TxNotificationRepository interface {
	Create(ctx context.Context, txNotification domain.TxNotification) (int64, error)
	FindCheckBack(ctx context.Context, offset, limit int) ([]domain.TxNotification, error)
	UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error
	UpdateCheckStatus(ctx context.Context, txNotifications []domain.TxNotification, notificationStatus domain.SendStatus) error
}
type txNotificationRepository struct {
	txdao dao.TxNotificationDAO
}

func NewTxNotificationRepository(txdao dao.TxNotificationDAO) TxNotificationRepository {
	return &txNotificationRepository{txdao: txdao}
}

func (t *txNotificationRepository) Create(ctx context.Context, txNotification domain.TxNotification) (int64, error) {
	txnEntity := t.toDao(txNotification)
	notificationEntity := t.toEntity(txNotification.Notification)
	return t.txdao.Prepare(ctx, txnEntity, notificationEntity)
}

func (t *txNotificationRepository) FindCheckBack(ctx context.Context, offset, limit int) ([]domain.TxNotification, error) {
	daoNotifications, err := t.txdao.FindCheckBack(ctx, offset, limit)
	if err != nil {
		return nil, err
	}
	res := make([]domain.TxNotification, 0, len(daoNotifications))
	for _, daoNotification := range daoNotifications {
		res = append(res, t.toDomain(daoNotification))
	}
	return res, nil
}

func (t *txNotificationRepository) UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error {
	return t.txdao.UpdateStatus(ctx, bizID, key, status, notificationStatus)
}

func (t *txNotificationRepository) UpdateCheckStatus(ctx context.Context, txNotifications []domain.TxNotification, notificationStatus domain.SendStatus) error {
	daoNotifications := make([]dao.TxNotification, 0, len(txNotifications))
	for idx := range txNotifications {
		txNotification := t.toDao(txNotifications[idx])
		daoNotifications = append(daoNotifications, txNotification)
	}

	return t.txdao.UpdateCheckStatus(ctx, daoNotifications, notificationStatus)
}

// toDao 将领域模型转换为DAO对象
func (t *txNotificationRepository) toDao(notification domain.TxNotification) dao.TxNotification {
	return dao.TxNotification{
		TxID:           notification.TxID,
		Key:            notification.Key,
		NotificationID: notification.Notification.ID,
		BizID:          notification.BizID,
		Status:         string(notification.Status),
		CheckCount:     notification.CheckCount,
		NextCheckTime:  notification.NextCheckTime,
		Ctime:          notification.Ctime,
		Utime:          notification.Utime,
	}
}

func (t *txNotificationRepository) toEntity(notification domain.Notification) dao.Notification {
	templateParams, _ := notification.MarshalTemplateParms()
	receivers, _ := notification.MarshalReceivers()
	return dao.Notification{
		ID:             notification.ID,
		BizID:          notification.BizID,
		Key:            notification.Key,
		Receivers:      receivers,
		Channel:        string(notification.Channel),
		TemplateID:     notification.Template.ID,
		TemplateParams: templateParams,
		Status:         string(notification.Status),
		ScheduledSTime: notification.ScheduledSTime.UnixMilli(),
		ScheduledETime: notification.ScheduledETime.UnixMilli(),
		Version:        notification.Version,
	}
}

func (t *txNotificationRepository) toDomain(txn dao.TxNotification) domain.TxNotification {
	return domain.TxNotification{
		TxID: txn.TxID,
		Notification: domain.Notification{
			ID: txn.NotificationID,
		},
		BizID:         txn.BizID,
		Key:           txn.Key,
		Status:        domain.TxNotificationStatus(txn.Status),
		CheckCount:    txn.CheckCount,
		NextCheckTime: txn.NextCheckTime,
		Ctime:         txn.Ctime,
		Utime:         txn.Utime,
	}
}
