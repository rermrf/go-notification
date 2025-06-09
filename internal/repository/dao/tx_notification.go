package dao

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/domain"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"strings"
	"time"
)

var (
	ErrDuplicatedTx       = errors.New("duplicated tx")
	ErrUpdateStatusFailed = errors.New("没有更新")
)

type TxNotification struct {
	// 事务id
	TxID int64  `gorm:"column:tx_id;autoIncrement;primaryKey'"`
	Key  string `gorm:"type:VARCHAR(256);NOT NULL;uniqueIndex:idx_biz_id_key,priority:2;comment:'业务内唯一标识，区分同一个业务内的不同通知'"`
	// 创建的通知id
	NotificationID int64 `gorm:"column:notification_id"`
	// 业务方唯一标识
	BizID int64 `gorm:"column:biz_id;type:bigint;not null; uniqueIndex:idx_biz_id_key"`
	// 通知状态
	Status string `gorm:"column:status;type:varchar(20);not null;default:'PREPARE';index:idx_next_check_time_status"`
	// 第几次检查，从1开始
	CheckCount int `gorm:"column:check_count;type:int;not null;default:1"`
	// 下一次的回查时间戳
	NextCheckTime int64 `gorm:"column:next_check_time;type:bigint;not null;default:0;index:idx_next_check_time_status"`
	Ctime         int64 `gorm:"column:ctime;type:bigint;not null"`
	Utime         int64 `gorm:"column:utime;type:bigint;not null"`
}

func (TxNotification) TableName() string {
	return "tx_notification"
}

type TxNotificationDAO interface {
	// FindCheckBack 查找需要回查的事物通知，筛选条件是status为PREPARE，并且下一次回查时间小于当前时间
	FindCheckBack(ctx context.Context, offset, limit int) ([]TxNotification, error)

	//CASStatus(ctx context.Context, txID int64, status string) error

	// UpdateCheckStatus 更新回查状态用于回查任务，回查次数+1 更新下一次的回查时间戳，状态通知，utime 要求都是同一状态的
	UpdateCheckStatus(ctx context.Context, txNotifications []TxNotification, status domain.SendStatus) error
	// First 通过事物id查找对应的事务
	First(ctx context.Context, txID int64) (TxNotification, error)
	// BatchGetTXNotification 批量获取事务消息
	BatchGetTXNotification(ctx context.Context, txIDs []int64) (map[int64]TxNotification, error)

	GetByBizIDKey(ctx context.Context, bizID int64, key string) (TxNotification, error)
	UpdateNotificationID(ctx context.Context, bizID int64, key string, notificationID int64) error

	Prepare(ctx context.Context, txNotification TxNotification, notification Notification) (int64, error)
	// UpdateStatus 提供给用户使用
	UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error
}

type txNotificationDAO struct {
	db *gorm.DB
}

func newTxNotificationDAO(db *gorm.DB) TxNotificationDAO {
	return &txNotificationDAO{db: db}
}

func (t *txNotificationDAO) FindCheckBack(ctx context.Context, offset, limit int) ([]TxNotification, error) {
	var notifications []TxNotification
	currentTime := time.Now().UnixMilli()

	err := t.db.WithContext(ctx).
		Where("status = ? AND next_check_time <= ? AND next_check_time > 0", domain.TxNotificationStatusPrepare, currentTime).Find(&notifications).
		Offset(offset).
		Limit(limit).
		Order("next_check_time").
		Find(&notifications).Error

	return notifications, err
}

func (t *txNotificationDAO) UpdateCheckStatus(ctx context.Context, txNotifications []TxNotification, status domain.SendStatus) error {
	sqls := make([]string, 0, len(txNotifications))
	now := time.Now().UnixMilli()
	notificationIDs := make([]int64, 0, len(txNotifications))
	for _, txNotification := range txNotifications {
		updateSQL := fmt.Sprintf("UPDATE `tx_notifications` set `status` = %s, `utime` = %d, `next_check_time` = %d, `check_count` = %d WHERE `key` =%s AND `biz_id` = %d AND `status` = 'PREPARE'", txNotification.Status, now, txNotification.NextCheckTime, txNotification.CheckCount, txNotification.Key, txNotification.BizID)
		sqls = append(sqls, updateSQL)
		notificationIDs = append(notificationIDs, txNotification.NotificationID)
	}
	// 拼接所有sql并执行
	// UPDATE xxx; UPDATE xxx; UPDATE xxx;
	if len(sqls) > 0 {
		return t.db.Transaction(func(tx *gorm.DB) error {
			combinedSQL := strings.Join(sqls, "; ")
			err := tx.WithContext(ctx).Exec(combinedSQL).Error
			if err != nil {
				return err
			}
			if status != domain.SendStatusPrepare {
				return tx.WithContext(ctx).Model(&Notification{}).Where("id in ?", notificationIDs).
					Update("status", status).Error
			}
			return nil
		})
	}
	return nil
}

func (t *txNotificationDAO) First(ctx context.Context, txID int64) (TxNotification, error) {
	var notification TxNotification
	err := t.db.WithContext(ctx).Where("tx_id = ?", txID).First(&notification).Error
	return notification, err
}

func (t *txNotificationDAO) BatchGetTXNotification(ctx context.Context, txIDs []int64) (map[int64]TxNotification, error) {
	var txns []TxNotification
	err := t.db.WithContext(ctx).Where("tx_id in (?)", txIDs).Find(&txns).Error
	if err != nil {
		return nil, err
	}
	res := make(map[int64]TxNotification, len(txns))
	for idx := range txns {
		txn := txns[idx]
		res[txn.TxID] = txn
	}
	return res, nil
}

func (t *txNotificationDAO) GetByBizIDKey(ctx context.Context, bizID int64, key string) (TxNotification, error) {
	var txn TxNotification
	err := t.db.WithContext(ctx).
		Model(&TxNotification{}).
		Where("biz_id = ? AND `key` = ?", bizID, key).First(&txn).Error
	return txn, err
}

func (t *txNotificationDAO) UpdateNotificationID(ctx context.Context, bizID int64, key string, notificationID int64) error {
	err := t.db.WithContext(ctx).
		Model(&TxNotification{}).
		Where("biz_id = ? AND `key` = ?", bizID, key).
		Update("notification_id", notificationID).Error
	return err
}

func (t *txNotificationDAO) Prepare(ctx context.Context, txNotification TxNotification, notification Notification) (int64, error) {
	var notificationID int64
	now := time.Now().UnixMilli()
	txNotification.Ctime = now
	txNotification.Utime = now
	notification.Ctime = now
	notification.Utime = now
	err := t.db.Transaction(func(tx *gorm.DB) error {
		res := tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&txNotification)
		if res.Error != nil {
			return res.Error
		}
		notificationID = notification.ID
		if res.RowsAffected == 0 {
			return nil
		}
		txNotification.NotificationID = notificationID
		return tx.WithContext(ctx).Clauses(clause.OnConflict{
			DoNothing: true,
		}).Create(&txNotification).Error
	})
	return notificationID, err
}

func (t *txNotificationDAO) UpdateStatus(ctx context.Context, bizID int64, key string, status domain.TxNotificationStatus, notificationStatus domain.SendStatus) error {
	return t.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.WithContext(ctx).
			Model(&TxNotification{}).
			Where("biz_id = ? AND `key` = ? AND status = 'PREPARE'", bizID, key).
			Update("status", status.String())
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return ErrUpdateStatusFailed
		}
		return tx.WithContext(ctx).Model(&Notification{}).
			Where("biz_id = ? AND `key` = ?", bizID, key).
			Update("status", notificationStatus).Error
	})
}
