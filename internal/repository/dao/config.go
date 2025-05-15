package dao

import (
	"github.com/google/uuid"
	"go-notification/internal/domain"
	"go-notification/internal/pkg/sqlx"
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
