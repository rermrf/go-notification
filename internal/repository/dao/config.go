package dao

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/sqlx"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type BusinessConfig struct {
	ID             int64                                  `gorm:"primaryKey;type:BIGINT;comment:'业务方标识'"`
	OwnerID        int64                                  `gorm:"type:BIGINT;comment:'业务方ID'"`
	OwnerType      string                                 `gorm:"type:ENUM('person', 'organization');comment:'业务方类型: 个人/组织'"`
	ChannelConfig  sqlx.JsonColumn[domain.ChannelConfig]  `gorm:"type:JSON;comment:'{\"channels\":[{\"channel\":\"SMS\", \"priority\":\"1\",\"enabled\":\"true\"},{\"channel\":\"EMAIL\", \"priority\":\"2\",\"enabled\":\"true\"}]}'"`
	TxnConfig      sqlx.JsonColumn[domain.TxnConfig]      `gorm:"type:JSON;comment:'事务配置'"`
	RateLimit      int                                    `gorm:"type:INT;DEFAULT:1000;comment:'速率限制'"`
	Quota          sqlx.JsonColumn[domain.QuotaConfig]    `gorm:"type:JSON;comment:'配额配置'"`
	CallbackConfig sqlx.JsonColumn[domain.CallbackConfig] `gorm:"type:JSON;comment:'回调配置，通知平台回调业务通知异步请求结果'"`
	Ctime          int64
	Utime          int64
}

func (BusinessConfig) TableName() string {
	return "business_configs"
}

type BusinessConfigDAO interface {
	GetByIDs(ctx context.Context, ids []int64) (map[int64]BusinessConfig, error)
	GetByID(ctx context.Context, id int64) (BusinessConfig, error)
	DeleteByID(ctx context.Context, id int64) error
	SaveConfig(ctx context.Context, config BusinessConfig) (BusinessConfig, error)
	Find(ctx context.Context, offset, limit int) ([]BusinessConfig, error)
}

type businessConfigDAO struct {
	db *gorm.DB
}

func NewBusinessConfigDAO(db *gorm.DB) BusinessConfigDAO {
	return &businessConfigDAO{db: db}
}

// GetByIDs 批量获取config
func (b businessConfigDAO) GetByIDs(ctx context.Context, ids []int64) (map[int64]BusinessConfig, error) {
	var businessConfigs []BusinessConfig
	if err := b.db.WithContext(ctx).Where("id in (?)", ids).Find(&businessConfigs).Error; err != nil {
		return nil, err
	}
	configMap := make(map[int64]BusinessConfig, len(ids))
	for idx := range businessConfigs {
		config := businessConfigs[idx]
		configMap[config.ID] = config
	}
	return configMap, nil
}

// GetByID 通过id查询config
func (b businessConfigDAO) GetByID(ctx context.Context, id int64) (BusinessConfig, error) {
	var config BusinessConfig

	// 根据ID查询业务配置
	if err := b.db.WithContext(ctx).Where("id = ?", id).First(&config).Error; err != nil {
		return BusinessConfig{}, err
	}
	return config, nil
}

// DeleteByID 根据ID删除config
func (b businessConfigDAO) DeleteByID(ctx context.Context, id int64) error {
	// 执行删除操作
	err := b.db.WithContext(ctx).Where("id = ?", id).Delete(&BusinessConfig{}).Error
	if err != nil {
		return err
	}
	return nil
}

// SaveConfig 保存业务配置
func (b businessConfigDAO) SaveConfig(ctx context.Context, config BusinessConfig) (BusinessConfig, error) {
	now := time.Now().UnixMilli()
	config.Ctime = now
	config.Utime = now
	// 使用 upsert 语句，如果记录存在则更新，不存在则插入
	db := b.db.WithContext(ctx)
	res := db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "id"}}, // 根据ID判断冲突
		DoUpdates: clause.AssignmentColumns([]string{
			"owner_id",
			"owner_type",
			"channel_config",
			"txn_config",
			"rate_limit",
			"quota",
			"callback_config",
			"utime",
		}), // 只更新制定的非空列
	}).Create(&config)
	if res.Error != nil {
		return BusinessConfig{}, res.Error
	}
	return config, nil
}

func (b businessConfigDAO) Find(ctx context.Context, offset, limit int) ([]BusinessConfig, error) {
	var result []BusinessConfig
	err := b.db.WithContext(ctx).Limit(limit).Offset(offset).Find(&result).Error
	return result, err
}
