package dao

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-sql-driver/mysql"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"gorm.io/gorm"
	"time"
)

type Notification struct {
	ID                int64  `gorm:"primaryKey;comment:'雪花算法ID'"`
	BizID             int64  `gorm:"type:BIGINT;NOT NULL;index:idx_biz_id_status,proority:1;uniqueIndex:idx_biz_id_key,priority:1;comment:'业务方配表ID，业务方可能有多个业务每个业务配置不同'"`
	Key               string `gorm:"type:VARCHAR(256);NOT NULL;quiqueIndex:idx_biz_id_key,priority:2;comment:'业务内唯一标识'"`
	Receivers         string `gorm:"type:TEXT;NOT NULL;comment:'接收者(手机/邮箱/用户ID)，JSON数组'"`
	Channel           string `gorm:"type:ENUM('SMS', 'EMAIL', 'IN_APP');NOT NULL;comment:'发送渠道'"`
	TemplateID        int64  `gorm:"type:BIGINT;NOT NULL;comment:'关联的模版ID'"`
	TemplateVersionID int64  `gorm:"type:BIGINT;NOT NULL;comment:'关联的模版版本ID'"`
	TemplateParams    string `gorm:"NOT NULL;comment:'模板参数'"`
	Status            string `gorm:"type:ENUM('PREPARE', 'CANCELED', 'PENDING', 'SENDING', 'SUCCEEDED', 'FAILED');DEFAULT:'PENDING';index:idx_biz_id_status,priority:2;comment:'发送状态'"`
	ScheduledSTime    int64  `gorm:"column:scheduled_time;index:idx_scheuled,priority:1;comment:'计划发送开始时间'"`
	ScheduledETime    int64  `gorm:"column:scheduled_time;index:idx_scheuled,priority:2;comment:'计划发送结束时间'"`
	Version           int    `gorm:"type:INT;NOT NULL;DEFAULT:1;comment:'版本号'"`
	Ctime             int64
	Utime             int64
}

type NotificationDAO interface {
	Create(ctx context.Context, data Notification) (Notification, error)
	CreateWithCallbackLog(ctx context.Context, data Notification) (Notification, error)
	BatchCreate(ctx context.Context, dataList []Notification) ([]Notification, error)
	BatchCreateWithCallbackLog(ctx context.Context, dataList []Notification) ([]Notification, error)

	GetByID(ctx context.Context, id int64) (Notification, error)

	BatchGetByIDs(ctx context.Context, ids []int64) (map[int64]Notification, error)

	GetByKey(ctx context.Context, BizID int64, key string) (Notification, error)
	GetByKeys(ctx context.Context, BizID int64, keys ...string) ([]Notification, error)

	CASStatus(ctx context.Context, notification Notification) error
	UpdateStatus(ctx context.Context, notification Notification) error

	BatchUpdateStatusSucceedOrFailed(ctx context.Context, succededNotifications, failedNotifications []Notification) error

	FindReadyNotifications(ctx context.Context, offset, limit int) ([]Notification, error)
	MarkSuccess(ctx context.Context, notification Notification) error
	MarkFailed(ctx context.Context, notification Notification) error
	MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error)
}

type notificationDAO struct {
	db *gorm.DB
}

func NewNotificationDAO(db *gorm.DB) NotificationDAO {
	return &notificationDAO{db: db}
}

func (d *notificationDAO) Create(ctx context.Context, data Notification) (Notification, error) {
	return d.create(ctx, d.db, data, false)
}

func (d *notificationDAO) CreateWithCallbackLog(ctx context.Context, data Notification) (Notification, error) {
	return d.create(ctx, d.db, data, true)
}

func (d *notificationDAO) BatchCreate(ctx context.Context, dataList []Notification) ([]Notification, error) {
	return d.batchCreate(ctx, dataList, false)
}

func (d *notificationDAO) BatchCreateWithCallbackLog(ctx context.Context, dataList []Notification) ([]Notification, error) {
	return d.batchCreate(ctx, dataList, true)
}

func (d *notificationDAO) GetByID(ctx context.Context, id int64) (Notification, error) {
	var notification Notification
	err := d.db.WithContext(ctx).First(&notification, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Notification{}, fmt.Errorf("%w: id=%d", errs.ErrNotificationNotFound, id)
		}
		return Notification{}, err
	}
	return notification, nil
}

func (d *notificationDAO) BatchGetByIDs(ctx context.Context, ids []int64) (map[int64]Notification, error) {
	var notifications []Notification
	err := d.db.WithContext(ctx).Where("id in (?)", ids).Find(&notifications).Error
	notificationMap := make(map[int64]Notification, len(ids))
	for idx := range notifications {
		notification := notifications[idx]
		notificationMap[notification.ID] = notification
	}
	return notificationMap, err
}

func (d *notificationDAO) GetByKey(ctx context.Context, BizID int64, key string) (Notification, error) {
	var notification Notification
	err := d.db.WithContext(ctx).Where("biz_id = ? AND key = ?", BizID, key).First(&notification).Error
	if err != nil {
		return Notification{}, fmt.Errorf("查询通知列表失败：bizID: %d, key %s %w", BizID, key, err)
	}
	return notification, nil
}

// GetByKeys 根据业务ID和业务内唯一标识获取通知列表
func (d *notificationDAO) GetByKeys(ctx context.Context, BizID int64, keys ...string) ([]Notification, error) {
	var notifications []Notification
	err := d.db.WithContext(ctx).Where("biz_id = ? AND key in (?)", BizID, keys).Find(&notifications).Error
	if err != nil {
		return notifications, fmt.Errorf("查询通知列表失败: %w", err)
	}
	return notifications, nil
}

// CASStatus 更新通知状态
func (d *notificationDAO) CASStatus(ctx context.Context, notification Notification) error {
	updates := map[string]interface{}{
		"status":  notification.Status,
		"version": gorm.Expr("version + 1"),
		"utime":   time.Now().UnixMilli(),
	}

	result := d.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ? AND version = ?", notification.ID, notification.Version).
		Updates(updates)

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected < 1 {
		return fmt.Errorf("并发竞争失败 %w, id %d", errs.ErrNotificationVersionMismatch, notification.ID)
	}
	return nil
}

func (d *notificationDAO) UpdateStatus(ctx context.Context, notification Notification) error {
	return d.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ?", notification.ID).
		Updates(map[string]interface{}{
			"status":  notification.Status,
			"version": gorm.Expr("version + 1"),
			"utime":   time.Now().UnixMilli(),
		}).Error
}

// BatchUpdateStatusSucceedOrFailed 批量更新通知状态为成功或失败，使用乐观锁控制并发
// successNotifications：更新为成功状态的通知列表，包含ID、Version和重试次数
// failedNotifications：更新为失败状态的通知列表，包含ID、Version和重试次数
func (d *notificationDAO) BatchUpdateStatusSucceedOrFailed(ctx context.Context, succededNotifications, failedNotifications []Notification) error {
	if len(succededNotifications) == 0 && len(failedNotifications) == 0 {
		return nil
	}

	successIDs := make([]int64, 0, len(succededNotifications))
	for _, notification := range succededNotifications {
		successIDs = append(successIDs, notification.ID)
	}
	failedIDs := make([]int64, 0, len(failedNotifications))
	for _, notification := range failedNotifications {
		failedIDs = append(failedIDs, notification.ID)
	}

	// 开启事务
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if len(successIDs) != 0 {
			err := d.batchMarkSuccess(tx, successIDs)
			if err != nil {
				return err
			}
		}
		if len(failedIDs) != 0 {
			now := time.Now().UnixMilli()
			return tx.Model(&Notification{}).
				Where("id in (?)", successIDs).
				Updates(map[string]interface{}{
					"status":  domain.SendStatusFailed.String(),
					"version": gorm.Expr("version + 1"),
					"utime":   now,
				}).Error
		}
		return nil
	})
}

func (d *notificationDAO) FindReadyNotifications(ctx context.Context, offset, limit int) ([]Notification, error) {
	var result []Notification
	now := time.Now().UnixMilli()
	err := d.db.WithContext(ctx).
		Where("scheduled_stime <= ? AND scheduled_etime >= ? AND status = ?", now, now, domain.SendStatusPending.String()).
		Limit(limit).Offset(offset).Find(&result).Error
	return result, err
}

func (d *notificationDAO) MarkSuccess(ctx context.Context, notification Notification) error {
	now := time.Now().UnixMilli()
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		err := tx.Model(&Notification{}).
			Where("id = ?", notification.ID).
			Updates(map[string]interface{}{
				"status":  notification.Status,
				"version": gorm.Expr("version + 1"),
				"utime":   now,
			}).Error
		if err != nil {
			return err
		}
		return tx.Model(&CallbackLog{}).Where("notification_id = ?", notification.ID).Updates(map[string]interface{}{
			// 标记为可以发送回调
			"status": domain.CallbackLogStatusPending,
			"utime":  now,
		}).Error
	})
}

func (d *notificationDAO) MarkFailed(ctx context.Context, notification Notification) error {
	now := time.Now().UnixMilli()
	return d.db.WithContext(ctx).Model(&Notification{}).
		Where("id = ?", notification.ID).
		Updates(map[string]interface{}{
			"status":  notification.Status,
			"version": gorm.Expr("version + 1"),
			"utime":   now,
		}).Error
}

func (d *notificationDAO) MarkTimeoutSendingAsFailed(ctx context.Context, batchSize int) (int64, error) {
	now := time.Now()
	ddl := now.Add(-time.Minute).UnixMilli()
	var rowsAffected int64

	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var idsToUpdate []int64

		// 查询需要更新的 ID
		err := tx.Model(&Notification{}).
			Select("id").
			Where("status = ? AND utime <= ?", domain.SendStatusPending.String(), ddl).
			Limit(batchSize).
			Find(&idsToUpdate).Error

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}

		// 没有找到需要更新的记录，直接成功返回（事务将提交）
		if len(idsToUpdate) == 0 {
			rowsAffected = 0
			return nil
		}

		// 根据查询到的 ID 集合更新记录
		res := tx.Model(&CallbackLog{}).
			Where("id IN (?)", idsToUpdate).
			Updates(map[string]interface{}{
				"status":  domain.CallbackLogStatusFailed.String(),
				"version": gorm.Expr("version + 1"),
				"utime":   now.UnixMilli(),
			})

		rowsAffected = res.RowsAffected
		return res.Error
	})

	return rowsAffected, err
}

// isUniqueConstraintError 检查是否是唯一约束错误
func (d *notificationDAO) isUniqueConstraintError(err error) bool {
	if err == nil {
		return false
	}
	me := new(mysql.MySQLError)
	if ok := errors.As(err, &me); ok {
		const uniqueIndexErrorCode = 1062
		return me.Number == uniqueIndexErrorCode
	}
	return false
}

func (d *notificationDAO) create(ctx context.Context, db *gorm.DB, data Notification, createCallbackLog bool) (Notification, error) {
	now := time.Now().UnixMilli()
	data.Ctime, data.Utime = now, now
	data.Version = 1

	err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&data).Error; err != nil {
			if d.isUniqueConstraintError(err) {
				return fmt.Errorf("%w", errs.ErrNotificationDuplicate)
			}
			return err
		}
		if createCallbackLog {
			if err := tx.Create(&CallbackLog{
				NotificationID: data.ID,
				Status:         domain.CallbackLogStatusInit.String(),
				NextRetryTime:  now,
			}).Error; err != nil {
				return fmt.Errorf("%w", errs.ErrCreateCallbackLogFailed)
			}
		}
		return nil
	})

	return data, err
}

func (d *notificationDAO) batchCreate(ctx context.Context, dataList []Notification, createCallbackLog bool) ([]Notification, error) {
	if len(dataList) == 0 {
		return dataList, nil
	}

	const batchSize = 100
	now := time.Now().UnixMilli()
	for i := range dataList {
		dataList[i].Ctime = now
		dataList[i].Utime = now
		dataList[i].Version = 1
	}

	// 使用事务执行批量插入
	err := d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 创建通知记录
		if err := tx.CreateInBatches(dataList, batchSize).Error; err != nil {
			if d.isUniqueConstraintError(err) {
				return fmt.Errorf("%w", errs.ErrNotificationDuplicate)
			}
			return err
		}

		if createCallbackLog {
			// 创建回调记录
			var callbackLogs []CallbackLog
			for i := range dataList {
				callbackLogs = append(callbackLogs, CallbackLog{
					NotificationID: dataList[i].ID,
					NextRetryTime:  now,
					Ctime:          now,
					Utime:          now,
				})
			}
			if err := tx.Create(&callbackLogs).Error; err != nil {
				return fmt.Errorf("%w", errs.ErrCreateCallbackLogFailed)
			}
		}
		return nil
	})

	return dataList, err
}

func (d *notificationDAO) batchMarkSuccess(tx *gorm.DB, successIDs []int64) error {
	now := time.Now().UnixMilli()
	err := tx.Model(&Notification{}).
		Where("id in (?)", successIDs).
		Updates(map[string]interface{}{
			"version": gorm.Expr("version + 1"),
			"utime":   now,
			"status":  domain.SendStatusSucceeded.String(),
		}).Error
	if err != nil {
		return err
	}

	// 更新 callback log
	return tx.Model(&Notification{}).
		Where("notification_id in (?)", successIDs).
		Updates(map[string]interface{}{
			"status": domain.CallbackLogStatusPending.String(),
			"utime":  now,
		}).Error
}
