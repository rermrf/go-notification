package repository

import (
	"context"
	"go-notification/internal/domain"
)

// ChannelTemplateRepository 提供模板数据存储的仓库接口
type ChannelTemplateRepository interface {
	// 模版相关方法

	// GetTemplateByOwner 获取指定所有者的模板列表
	GetTemplateByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) (domain.ChannelTemplate, error)

	// GetTemplateByID 根据ID获取模版
	GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error)

	// CreateTemplate 创建模版
	CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error)

	// UpdateTemplate 更新模版
	UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error

	// SetTemplateActiveVersion 谁模板的活跃版本
	SetTemplateActiveVersion(ctx context.Context, templateID, versionID int64) error

	// 模版版本相关方法

	// GetTemplateVersionByID 根据ID获取模板版本
	GetTemplateVersionByID(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error)

	// CreateTemplateVersion 创建模板版本
	CreateTemplateVersion(ctx context.Context, templateVersion domain.ChannelTemplateVersion) (domain.ChannelTemplateVersion, error)

	// ForkTemplateVersion 基于已有版本创建新版本
	ForkTemplateVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error)

	// 供应商相关方法

	// GetProviderByNameAndChannel 根据名称和渠道获取供应商
	GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName string, channel domain.Channel) ([]domain.ChannelTemplateProvider, error)

	// BatchCreateTemplateProviders 批量创建模板供应商关联
	BatchCreateTemplateProviders(ctx context.Context, providers []domain.ChannelTemplateProvider) ([]domain.ChannelTemplateProvider, error)

	// GetApprovedProvidersByTemplateIDAndVersionID 获取已审核通过的供应商列表
	GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error)

	// GetProvidersByTemplateIDAndVersionID 获取模版和版本管理的所有供应商
	GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error)

	// UpdateTemplateVersion 更新模板版本
	UpdateTemplateVersion(ctx context.Context, templateVersion domain.ChannelTemplateVersion) error

	// BatchUpdateTemplateVersionAuditInfo 批量更新模板版本审核信息
	BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []domain.ChannelTemplateVersion) error

	// UpdateTemplateProviderAuditInfo 更新模板供应商审核信息
	UpdateTemplateProviderAuditInfo(ctx context.Context, provider domain.ChannelTemplateProvider) error

	// BatchUpdateTemplateProvidersAuditInfo 批量更新模板供应商审核信息
	BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error

	// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, ctime int64) (providers []domain.ChannelTemplateProvider, total int64, err error)
}

type channelTemplateRepository struct {
	dao dao.C
}
