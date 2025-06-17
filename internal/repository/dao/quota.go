package dao

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/errs"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Quota struct {
	ID int64 `gorm:"primaryKey;comment:'雪花算法ID'"`
	// 构成一个唯一索引
	BizID   int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:biz_id_channel,priority:1;comment:'业务配置表ID，业务方可能有多个业务每个业务配置不同'"`
	Channel string `gorm:"type:ENUM('SMS', 'EMAIL', 'IN_APP');NOT NULL;uniqueIndex:biz_id_channel,priority:2;comment:'发送渠道'"`
	// 每个月的 quota
	// 如果你要分开控制不同渠道的 Quota，那么就加一个 Channel 列
	// 确保不同 Channel 使用不同的 Quota 来规避更新的锁竞争（CAS 等）
	Quota int32

	// 版本号，用于 CAS，你可以考虑使用 CAS 来更新
	//Version int `gorm:"type:INT;NOT NULL;DEFAULT:1;comment:'版本号'"`
	Utime int64
	Ctime int64
}

type QuotaDAO interface {
	CreateOrUpdate(ctx context.Context, quota ...Quota) error
	Find(ctx context.Context, bizID int64, channel string) (Quota, error)
}

type quotaDAO struct {
	db *gorm.DB
}

func NewQuotaDAO(db *gorm.DB) QuotaDAO {
	return &quotaDAO{db: db}
}

func (d *quotaDAO) CreateOrUpdate(ctx context.Context, quota ...Quota) error {
	now := time.Now().UnixMilli()
	for i := range quota {
		quota[i].Ctime = now
		quota[i].Utime = now
	}
	return d.db.WithContext(ctx).Clauses(clause.OnConflict{
		DoUpdates: clause.AssignmentColumns([]string{"quota", "utime"}),
	}).Create(&quota).Error
}

func (d *quotaDAO) Find(ctx context.Context, bizID int64, channel string) (Quota, error) {
	var q Quota
	err := d.db.WithContext(ctx).Where("biz_id = ? AND channel = ?", bizID, channel).First(&q).Error
	if err != nil && errors.Is(err, gorm.ErrRecordNotFound) {
		return q, fmt.Errorf("%w", errs.ErrQuotaNotFound)
	}
	return q, nil
}
