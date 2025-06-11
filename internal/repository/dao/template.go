package dao

import (
	"context"
	"errors"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"gorm.io/gorm"
	"time"
)

// ChannelTemplate 渠道模板表
type ChannelTemplate struct {
	ID              int64  `gorm:"primaryKey;autoIncrement;comment:'渠道模版ID'"`
	OwnerID         int64  `gorm:"type:BIGINT;NOT NULL;comment:'用户ID或部门ID'"`
	OwnerType       string `gorm:"type:ENUM('person', 'organization');NOT NULL;comment:'业务方类型：person-个人，organization-组织'"`
	Name            string `gorm:"type:VARCHAR(128);NOT NULL;comment:'模板名称'"`
	Description     string `gorm:"type:VARCHAR(512);NOT NULL;comment:'模版描述'"`
	Channel         string `gorm:"type:ENUM('SMS', 'EMAIL', 'IN_APP');NOT NULL;comment:'渠道类型'"`
	BusinessType    int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:1;comment:'业务类型：1-推广营销、2-通知、3-验证码等'"`
	ActiveVersionID int64  `gorm:"type:BIGINT;DEFAULT:0;index:idx_active_version;comment:'当前启用的版本ID，0表示无活跃版本'"`
	Ctime           int64
	Utime           int64
}

func (ChannelTemplate) TableName() string {
	return "channel_templates"
}

// ChannelTemplateVersion 渠道模板版本表
type ChannelTemplateVersion struct {
	ID                int64  `gorm:"primaryKey;autoIncrement;comment:'渠道模板版本ID'"`
	ChannelTemplateID int64  `gorm:"type:BIGINT;NOT NULL;index:idx_channel_template_id;comment:'关联渠道模板ID'"`
	Name              string `gorm:"type:VARCHAR(32);NOT NULL;comment:'版本名称，如v1.0.1'"`
	Signature         string `gorm:"type:VARCHAR(64);comment:'已通过所有供应商审核的短信签名/邮件发件人'"`
	Content           string `gorm:"type:TEXT;NOT NULL;comment:'原始模版内容，使用平台统一变量格式，如${bane}'"`
	Remark            string `gorm:"type:TEXT;NOT NULL;comment:'申请说明，描述使用短信的业务场景，并提供短信完整示例（填入变量内容），短信完整有助于提高模版审核通过率'"`
	// 审核相关信息，AuditID之后的为冗余的信息
	AuditID                   int64  `gorm:"type:BIGINT;NOT NULL;DEFAULT:0;comment:'审核表ID，0表示尚未提交审核或者未拿到审核结果'"`
	AuditorID                 int64  `gorm:"type:BIGINT;comment:'审核人ID'"`
	AuditTime                 int64  `gorm:"comment:'审核时间'"`
	AuditStatus               string `gorm:"type:ENUM('PENDING', 'IN_REVIEW', 'REJECTED', 'APPROVED');NOT NULL; DEFAULT:'PENDING';comment:'内部审核状态，PENDING表示未提交审核；IN_REVIEW表示已提交审核；APPROVED表示审核通过；REJECTED表示审核未通过'"`
	RejectReason              string `gorm:"type:VARCHAR(512);comment:'拒绝原因'"`
	LastREeviewSubmissionTime int64  `gorm:"comment:'上次提交审核时间'"`
	Ctime                     int64
	Utime                     int64
}

func (ChannelTemplateVersion) TableName() string {
	return "channel_template_versions"
}

type ChannelTemplateProvider struct {
	ID                        int64  `gorm:"primaryKey;autoIncrement;comment:'渠道模板-供应商关联ID'"`
	TemplateID                int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:idx_template_version_provider,priority:1;unqiueIndex:idx_temp_ver_name_chan,priority:1;comment:'渠道模板ID'"`
	TemplateVersionID         int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:idx_template_version_provider,priority:2;unqiueIndex:idx_temp_ver_name_chan,priority:2;comment:'渠道模板版本ID'"`
	ProviderID                int64  `gorm:"type:BIGINT;NOT NULL;uniqueIndex:idx_template_version_provider,priority:3;comment:'供应商ID'"`
	ProviderName              string `gorm:"type:VARCHAR(64);NOT NULL;unqiueIndex:idx_temp_ver_name_chan,priority:3;comment:'供应商名称'"`
	ProviderChannel           string `gorm:"type:ENUM('SMS', 'EMAIL', 'IN_APP');NOT NULL;unqiueIndex:idx_temp_ver_name_chan,priority:4;comment:'渠道类型')"`
	RequestID                 string `gorm:"type:VARCHAR(256);index:idx_request_id;comment:'审核请求在供应商侧的ID，用于排查问题'"`
	ProviderTemplateID        string `gorm:"type:VARCHAR(256);comment:'当前版本模板在供应商侧的ID，审核通过后才会有值'"`
	AuditStatus               string `gorm:"type:ENUM('PENDING', 'IN_REVIEW', 'REJECTED', 'APPROVED');NOT NULL;DEFAULT:'PENDING';index:idx_audit_status;comment:'供应商侧模板审核状态，PENDING表示未提交审核；IN_REVIEW表示未提交审核；APPROVED表示审核通过；REJECTED表示审核未通过'"`
	RejectReason              string `gorm:"type:VARCHAR(512);comment:'供应商侧拒绝原因'"`
	LastREeviewSubmissionTime int64  `gorm:"comment:'上一次提交审核时间'"`
	Ctime                     int64
	Utime                     int64
}

func (ChannelTemplateProvider) TableName() string {
	return "channel_template_providers"
}

// ChannelTemplateDAO 提供模板数据访问对象接口
type ChannelTemplateDAO interface {
	// 模板相关方法

	// GetTemplateByOwner 获取指定所有者的模板列表
	GetTemplateByOwner(ctx context.Context, ownerID int64, ownerType string) ([]ChannelTemplate, error)

	// GetTemplateByID 根据ID获取模板
	GetTemplateByID(ctx context.Context, id int64) (ChannelTemplate, error)

	// CreateTemplate 创建模板
	CreateTemplate(ctx context.Context, template ChannelTemplate) (ChannelTemplate, error)

	// UpdateTemplate 更新模板
	UpdateTemplate(ctx context.Context, template ChannelTemplate) error

	// SetTemplateActiveVersion 设置模块的活跃版本
	SetTemplateActiveVersion(ctx context.Context, templateID, VersionID int64) error

	// 模版版本相关方法

	// GetTemplateVersionByTemplateIDs 根据模板ID列表获取对应的版本列表
	GetTemplateVersionByTemplateIDs(ctx context.Context, templateIDs []int64) ([]ChannelTemplateVersion, error)

	// GetTemplateVersionByID 根据ID获取模块版本
	GetTemplateVersionByID(ctx context.Context, versionID int64) (ChannelTemplateVersion, error)

	// CreateTemplateVersion 创建模板版本
	CreateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) (ChannelTemplateVersion, error)

	// ForkTemplateVersion 基于已有版本创建新版本
	ForkTemplateVersion(ctx context.Context, versionID int64) (ChannelTemplateVersion, error)

	// 供应商相关方法

	// GetProviderByVersionIDs 根据版本ID列表获取供应商列表
	GetProviderByVersionIDs(ctx context.Context, versionIDs []int64) ([]ChannelTemplateProvider, error)

	// GetProviderByNameAndChannel 根据名称和渠道获取供应商
	GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName, channelName string) ([]ChannelTemplateProvider, error)

	// BatchCreateTemplateProviders 批量创建模块供应商关联
	BatchCreateTemplateProviders(ctx context.Context, providers []ChannelTemplateProvider) ([]ChannelTemplateProvider, error)

	// GetApprovedProvidersByTemplateIDAndVersionID 获取已审核通过的供应商列表
	GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID int64, versionID int64) ([]ChannelTemplateProvider, error)

	// GetProvidersByTemplateIDAndVersionID 获取模版和版本关联的所有供应商
	GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID int64, versionID int64) ([]ChannelTemplateProvider, error)

	// UpdateTemplateVersion 更新模块版本信息
	UpdateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) error

	// BatchUpdateTemplateVersionAuditInfo 批量更新模块版本审核信息
	BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []ChannelTemplateVersion) error

	// UpdateTemplateProviderAuditInfo 更新模块版本供应商审核信息
	UpdateTemplateProviderAuditInfo(ctx context.Context, provider ChannelTemplateProvider) error

	// BatchUpdateTemplateProvidersAuditInfo 批量更新模板供应商审核信息
	BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []ChannelTemplateProvider) error

	// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) ([]ChannelTemplateProvider, error)

	// TotalPendingOrInReviewProviders 统计未审核或审核中的供应商关联总数
	TotalPendingOrInReviewProviders(ctx context.Context, utime int64) (int64, error)
}

type channelTemplateDAO struct {
	db *gorm.DB
}

func NewChannelTemplateDAO(db *gorm.DB) ChannelTemplateDAO {
	return &channelTemplateDAO{db: db}
}

// 模版相关方法

// GetTemplateByOwner 根据所有者获取模版列表
func (c *channelTemplateDAO) GetTemplateByOwner(ctx context.Context, ownerID int64, ownerType string) ([]ChannelTemplate, error) {
	var templates []ChannelTemplate
	result := c.db.WithContext(ctx).Where("owner_id = ? and owner_type = ?", ownerID, ownerType).Find(&templates)
	if result.Error != nil {
		return nil, result.Error
	}
	return templates, nil
}

// GetTemplateByID 根据ID获取模版
func (c *channelTemplateDAO) GetTemplateByID(ctx context.Context, id int64) (ChannelTemplate, error) {
	var template ChannelTemplate
	err := c.db.WithContext(ctx).Where("id = ?", id).First(&template).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ChannelTemplate{}, fmt.Errorf("%w", errs.ErrTemplateNotFound)
		}
		return ChannelTemplate{}, err
	}
	return template, nil
}

// CreateTemplate 创建模板
func (c *channelTemplateDAO) CreateTemplate(ctx context.Context, template ChannelTemplate) (ChannelTemplate, error) {
	now := time.Now().UnixMilli()
	template.Ctime = now
	template.Utime = now
	result := c.db.WithContext(ctx).Create(&template)
	if result.Error != nil {
		return ChannelTemplate{}, result.Error
	}
	return template, nil
}

// UpdateTemplate 更新模板基本信息
func (c *channelTemplateDAO) UpdateTemplate(ctx context.Context, template ChannelTemplate) error {
	// 只允许更新name、descript、business_type
	updateData := map[string]interface{}{
		"name":          template.Name,
		"description":   template.Description,
		"business_type": template.BusinessType,
		"utime":         template.Utime,
	}
	return c.db.WithContext(ctx).Model(&ChannelTemplate{}).Where("id = ?").Updates(updateData).Error
}

// SetTemplateActiveVersion 设置模板活跃版本
func (c *channelTemplateDAO) SetTemplateActiveVersion(ctx context.Context, templateID, VersionID int64) error {
	return c.db.WithContext(ctx).Model(&ChannelTemplate{}).
		Where("id = ?", templateID).
		Updates(map[string]interface{}{
			"active_version": VersionID,
			"utime":          time.Now().UnixMilli(),
		}).Error
}

// 模版版本相关方法

// GetTemplateVersionByTemplateIDs 根据模板IDs获取版本列表
func (c *channelTemplateDAO) GetTemplateVersionByTemplateIDs(ctx context.Context, templateIDs []int64) ([]ChannelTemplateVersion, error) {
	if len(templateIDs) == 0 {
		return nil, nil
	}
	var versions []ChannelTemplateVersion
	result := c.db.WithContext(ctx).Where("channel_template_id IN (?)", templateIDs).Find(&versions)
	if result.Error != nil {
		return nil, result.Error
	}
	return versions, nil
}

func (c *channelTemplateDAO) GetTemplateVersionByID(ctx context.Context, versionID int64) (ChannelTemplateVersion, error) {
	var version ChannelTemplateVersion
	err := c.db.WithContext(ctx).Where("id = ?", versionID).First(&version).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ChannelTemplateVersion{}, fmt.Errorf("%w", errs.ErrTemplateVersionNotFound)
		}
		return ChannelTemplateVersion{}, err
	}
	return version, nil
}

// CreateTemplateVersion 创建模板版本
func (c *channelTemplateDAO) CreateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) (ChannelTemplateVersion, error) {
	now := time.Now().UnixMilli()
	version.Ctime = now
	version.Utime = now

	err := c.db.WithContext(ctx).Create(&version).Error
	if err != nil {
		return ChannelTemplateVersion{}, err
	}
	return version, nil
}

// ForkTemplateVersion 从已有的版本创建
func (c *channelTemplateDAO) ForkTemplateVersion(ctx context.Context, versionID int64) (ChannelTemplateVersion, error) {
	now := time.Now().UnixMilli()
	var created ChannelTemplateVersion
	err := c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 找到被拷贝的记录
		var old ChannelTemplateVersion
		if err := tx.First(&old, "id = ? ", versionID).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return fmt.Errorf("%w", errs.ErrTemplateVersionNotFound)
			}
			return err
		}

		// 拷贝记录
		fork := ChannelTemplateVersion{
			ChannelTemplateID:         old.ChannelTemplateID,
			Name:                      "Forked" + old.Name,
			Signature:                 old.Signature,
			Content:                   old.Content,
			Remark:                    old.Remark,
			AuditID:                   0,
			AuditorID:                 0,
			AuditTime:                 0,
			AuditStatus:               domain.AuditStatusPending.String(),
			RejectReason:              "",
			LastREeviewSubmissionTime: 0,
			Ctime:                     now,
			Utime:                     now,
		}
		if err := tx.Create(&fork).Error; err != nil {
			return err
		}

		created = fork

		// 获取供应商
		var providers []ChannelTemplateProvider
		if err := tx.Model(&ChannelTemplateProvider{}).
			Where("template_id = ? AND template_version = ?", old.ChannelTemplateID, versionID).
			Find(&providers).Error; err != nil {
			return err
		}

		forkedProviders := make([]ChannelTemplateProvider, 0, len(providers))
		for _, provider := range providers {
			forkedProviders = append(forkedProviders, ChannelTemplateProvider{
				TemplateID:                fork.ChannelTemplateID,
				TemplateVersionID:         fork.ID,
				ProviderID:                provider.ID,
				ProviderName:              provider.ProviderName,
				ProviderChannel:           provider.ProviderChannel,
				RequestID:                 "",
				ProviderTemplateID:        "",
				AuditStatus:               domain.AuditStatusPending.String(),
				RejectReason:              "",
				LastREeviewSubmissionTime: 0,
				Ctime:                     now,
				Utime:                     now,
			})
		}
		if err := tx.Create(&forkedProviders).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return ChannelTemplateVersion{}, err
	}
	return created, nil
}

// 供应商相关方法

// GetProviderByVersionIDs 根据版本IDs获取供应商关联
func (c *channelTemplateDAO) GetProviderByVersionIDs(ctx context.Context, versionIDs []int64) ([]ChannelTemplateProvider, error) {
	if len(versionIDs) == 0 {
		return nil, nil
	}
	var providers []ChannelTemplateProvider
	err := c.db.WithContext(ctx).Where("template_version_id IN (?)", versionIDs).Find(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// GetProviderByNameAndChannel 根据名称和渠道获取已通过审核的供应商
func (c *channelTemplateDAO) GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName, channelName string) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := c.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("template_id = ? AND template_version_id = ? AND provider_name = ? AND provider_channel = ? AND audit_status = ?", templateID, versionID, providerName, channelName, domain.AuditStatusPending).
		Find(&providers).Error
	return providers, err

}

// BatchCreateTemplateProviders 批量创建模板供应商
func (c *channelTemplateDAO) BatchCreateTemplateProviders(ctx context.Context, providers []ChannelTemplateProvider) ([]ChannelTemplateProvider, error) {
	if len(providers) == 0 {
		return nil, nil
	}

	now := time.Now().UnixMilli()

	for i := range providers {
		providers[i].Ctime = now
		providers[i].Utime = now
	}
	err := c.db.WithContext(ctx).Create(&providers).Error
	if err != nil {
		return nil, err
	}
	return providers, nil
}

// GetApprovedProvidersByTemplateIDAndVersionID 根据模板ID和和版本ID查找审核通过的供应商
func (c *channelTemplateDAO) GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID int64, versionID int64) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := c.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("template_id = ? AND template_version_id = ? AND audit_status", templateID, versionID, domain.AuditStatusApproved).
		Find(&providers).Error
	return providers, err
}

// GetProvidersByTemplateIDAndVersionID 根据模板ID和版本ID获取所有的供应商
func (c *channelTemplateDAO) GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID int64, versionID int64) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := c.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("template_id = ? AND template_version_id = ?", templateID, versionID).
		Find(&providers).Error
	return providers, err
}

// UpdateTemplateVersion 更新模板版本信息
func (c *channelTemplateDAO) UpdateTemplateVersion(ctx context.Context, version ChannelTemplateVersion) error {
	// 只允许更新的字段
	updateData := map[string]interface{}{
		"name":      version.Name,
		"signature": version.Signature,
		"content":   version.Content,
		"remark":    version.Remark,
		"utime":     time.Now().UnixMilli(),
	}

	return c.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).Where("id = ? ", version.ID).Updates(updateData).Error
}

// BatchUpdateTemplateVersionAuditInfo 更新模板版本审核信息
func (c *channelTemplateDAO) BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []ChannelTemplateVersion) error {
	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range versions {
			updateData := map[string]interface{}{
				"audit_status": versions[i].AuditStatus,
				"utime":        time.Now().UnixMilli(),
			}

			// 有条件地添加其他字段
			if versions[i].AuditID > 0 {
				updateData["audit_id"] = versions[i].AuditID
			}

			if versions[i].AuditorID > 0 {
				updateData["audit_status"] = versions[i].AuditorID
			}
			if versions[i].AuditTime > 0 {
				updateData["audit_time"] = versions[i].AuditTime
			}
			if versions[i].RejectReason != "" {
				updateData["reject_reason"] = versions[i].RejectReason
			}
			if versions[i].LastREeviewSubmissionTime > 0 {
				updateData["last_review_submission_time"] = versions[i].LastREeviewSubmissionTime
			}

			if err := tx.Model(&ChannelTemplateProvider{}).Where("id = ? ", versions[i].ID).Updates(updateData).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// UpdateTemplateProviderAuditInfo 更新模版供应商审核信息
func (c *channelTemplateDAO) UpdateTemplateProviderAuditInfo(ctx context.Context, provider ChannelTemplateProvider) error {
	updateData := map[string]interface{}{
		"utime": time.Now().UnixMilli(),
	}
	if provider.RequestID != "" {
		updateData["request_id"] = provider.RequestID
	}
	if provider.ProviderTemplateID != "" {
		updateData["provider_template_id"] = provider.ProviderTemplateID
	}
	if provider.AuditStatus != "" {
		updateData["audit_status"] = provider.AuditStatus
	}
	if provider.RejectReason != "" {
		updateData["reject_reason"] = provider.RejectReason
	}
	if provider.LastREeviewSubmissionTime > 0 {
		updateData["last_review_submission_time"] = provider.LastREeviewSubmissionTime
	}

	return c.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).Where("id = ? ", provider.ID).Updates(updateData).Error
}

// BatchUpdateTemplateProvidersAuditInfo 批量更新模版供应商审核信息
func (c *channelTemplateDAO) BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []ChannelTemplateProvider) error {
	if len(providers) == 0 {
		return nil
	}

	return c.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for i := range providers {
			updateData := map[string]interface{}{
				"utime": time.Now().UnixMilli(),
			}
			provider := providers[i]
			if provider.RequestID != "" {
				updateData["request_id"] = provider.RequestID
			}
			if provider.ProviderTemplateID != "" {
				updateData["provider_template_id"] = provider.ProviderTemplateID
			}
			if provider.AuditStatus != "" {
				updateData["audit_status"] = provider.AuditStatus
			}
			if provider.RejectReason != "" {
				updateData["reject_reason"] = provider.RejectReason
			}
			if provider.LastREeviewSubmissionTime > 0 {
				updateData["last_review_submission_time"] = provider.LastREeviewSubmissionTime
			}

			err := tx.Model(&ChannelTemplateProvider{}).Where("id = ? ", provider.ID).Updates(updateData).Error
			if err != nil {
				return err
			}
		}
		return nil
	})
}

// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
func (c *channelTemplateDAO) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) ([]ChannelTemplateProvider, error) {
	var providers []ChannelTemplateProvider
	err := c.db.WithContext(ctx).
		Where("(audit_status = ? OR audit_status = ?) AND utime <= ?", domain.AuditStatusPending, domain.AuditStatusPending, utime).
		Offset(offset).
		Limit(limit).
		Find(&providers).Error
	return providers, err
}

// TotalPendingOrInReviewProviders 统计未审核或者审核中的供应商关联总数
func (c *channelTemplateDAO) TotalPendingOrInReviewProviders(ctx context.Context, utime int64) (int64, error) {
	var res int64
	err := c.db.WithContext(ctx).Model(&ChannelTemplateProvider{}).
		Where("(audit_status = ? OR audit_status = ?) AND utime <= ?", domain.AuditStatusPending, domain.AuditStatusPending, utime).
		Count(&res).Error
	return res, err
}
