package repository

import (
	"context"
	"go-notification/internal/domain"
	"go-notification/internal/repository/dao"
)

// ChannelTemplateRepository 提供模板数据存储的仓库接口
type ChannelTemplateRepository interface {
	// 模版相关方法

	// GetTemplatesByOwner 获取指定所有者的模板列表
	GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error)

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
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error)
}

type channelTemplateRepository struct {
	dao dao.ChannelTemplateDAO
}

func NewChannelTemplateRepository(dao dao.ChannelTemplateDAO) ChannelTemplateRepository {
	return &channelTemplateRepository{dao: dao}
}

func (r *channelTemplateRepository) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error) {
	// 获取模版列表
	templates, err := r.dao.GetTemplateByOwner(ctx, ownerID, ownerType.String())
	if err != nil {
		return nil, err
	}
	if len(templates) == 0 {
		return nil, err
	}

	return r.getTemplates(ctx, templates)
}

func (r *channelTemplateRepository) GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error) {
	templateEntity, err := r.dao.GetTemplateByID(ctx, templateID)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}
	templates, err := r.getTemplates(ctx, []dao.ChannelTemplate{templateEntity})
	if err != nil {
		return domain.ChannelTemplate{}, err
	}
	return templates[0], nil
}

func (r *channelTemplateRepository) CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error) {
	templateEntity := r.toTemplateEntity(template)

	// 创建模版
	createdTemplate, err := r.dao.CreateTemplate(ctx, templateEntity)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	return r.toTemplateDomain(createdTemplate), nil
}

func (r *channelTemplateRepository) UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error {
	return r.dao.UpdateTemplate(ctx, r.toTemplateEntity(template))
}

func (r *channelTemplateRepository) SetTemplateActiveVersion(ctx context.Context, templateID, versionID int64) error {
	return r.dao.SetTemplateActiveVersion(ctx, templateID, versionID)
}

// 模板版本相关方法

func (r *channelTemplateRepository) GetTemplateVersionByID(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	version, err := r.dao.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}
	providers, err := r.dao.GetProviderByVersionIDs(ctx, []int64{versionID})
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}
	domainProviders := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		domainProviders = append(domainProviders, r.toProviderDomain(providers[i]))
	}

	domainVersion := r.toVersionDomain(version)
	domainVersion.Providers = domainProviders
	return domainVersion, nil
}

func (r *channelTemplateRepository) CreateTemplateVersion(ctx context.Context, templateVersion domain.ChannelTemplateVersion) (domain.ChannelTemplateVersion, error) {
	versionEntity := r.toVersionEntity(templateVersion)
	createdVersion, err := r.dao.CreateTemplateVersion(ctx, versionEntity)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}
	return r.toVersionDomain(createdVersion), nil
}

func (r *channelTemplateRepository) ForkTemplateVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	v, err := r.dao.ForkTemplateVersion(ctx, versionID)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}

	version := r.toVersionDomain(v)

	providers, err := r.dao.GetProvidersByTemplateIDAndVersionID(ctx, v.ChannelTemplateID, v.ID)
	if err != nil {
		return domain.ChannelTemplateVersion{}, err
	}

	domainProviders := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		domainProviders = append(domainProviders, r.toProviderDomain(providers[i]))
	}
	version.Providers = domainProviders
	return version, nil
}

// 供应商相关方法

func (r *channelTemplateRepository) GetProviderByNameAndChannel(ctx context.Context, templateID, versionID int64, providerName string, channel domain.Channel) ([]domain.ChannelTemplateProvider, error) {
	providers, err := r.dao.GetProviderByNameAndChannel(ctx, templateID, versionID, providerName, channel.String())
	if err != nil {
		return nil, err
	}
	result := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		result = append(result, r.toProviderDomain(providers[i]))
	}
	return result, nil
}

func (r *channelTemplateRepository) BatchCreateTemplateProviders(ctx context.Context, providers []domain.ChannelTemplateProvider) ([]domain.ChannelTemplateProvider, error) {
	if len(providers) == 0 {
		return nil, nil
	}
	daoProviders := make([]dao.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		daoProviders = append(daoProviders, r.toProviderEntity(providers[i]))
	}

	createdProviders, err := r.dao.BatchCreateTemplateProviders(ctx, daoProviders)
	if err != nil {
		return nil, err
	}
	result := make([]domain.ChannelTemplateProvider, 0, len(createdProviders))
	for i := range createdProviders {
		result = append(result, r.toProviderDomain(createdProviders[i]))
	}
	return result, nil
}

func (r *channelTemplateRepository) GetApprovedProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error) {
	providers, err := r.dao.GetApprovedProvidersByTemplateIDAndVersionID(ctx, templateID, versionID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		result = append(result, r.toProviderDomain(providers[i]))
	}
	return result, nil
}

func (r *channelTemplateRepository) GetProvidersByTemplateIDAndVersionID(ctx context.Context, templateID, versionID int64) ([]domain.ChannelTemplateProvider, error) {
	providers, err := r.dao.GetProvidersByTemplateIDAndVersionID(ctx, templateID, versionID)
	if err != nil {
		return nil, err
	}
	result := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		result = append(result, r.toProviderDomain(providers[i]))
	}
	return result, nil
}

func (r *channelTemplateRepository) UpdateTemplateVersion(ctx context.Context, templateVersion domain.ChannelTemplateVersion) error {
	return r.dao.UpdateTemplateVersion(ctx, r.toVersionEntity(templateVersion))
}

func (r *channelTemplateRepository) BatchUpdateTemplateVersionAuditInfo(ctx context.Context, versions []domain.ChannelTemplateVersion) error {
	versionEntitys := make([]dao.ChannelTemplateVersion, 0, len(versions))
	for i := range versions {
		versionEntitys = append(versionEntitys, r.toVersionEntity(versions[i]))
	}
	return r.dao.BatchUpdateTemplateVersionAuditInfo(ctx, versionEntitys)
}

func (r *channelTemplateRepository) UpdateTemplateProviderAuditInfo(ctx context.Context, provider domain.ChannelTemplateProvider) error {
	return r.dao.UpdateTemplateProviderAuditInfo(ctx, r.toProviderEntity(provider))
}

func (r *channelTemplateRepository) BatchUpdateTemplateProvidersAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	providerEntitys := make([]dao.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		providerEntitys = append(providerEntitys, r.toProviderEntity(providers[i]))
	}
	return r.dao.BatchUpdateTemplateProvidersAuditInfo(ctx, providerEntitys)
}

func (r *channelTemplateRepository) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error) {
	var providerEntitys []dao.ChannelTemplateProvider
	providerEntitys, err = r.dao.GetPendingOrInReviewProviders(ctx, offset, limit, utime)
	if err != nil {
		return nil, 0, err
	}
	total, err = r.dao.TotalPendingOrInReviewProviders(ctx, utime)
	if err != nil {
		return nil, 0, err
	}
	result := make([]domain.ChannelTemplateProvider, 0, len(providerEntitys))
	for i := range providerEntitys {
		result = append(result, r.toProviderDomain(providerEntitys[i]))
	}
	return result, total, nil
}

func (r *channelTemplateRepository) getTemplates(ctx context.Context, templates []dao.ChannelTemplate) ([]domain.ChannelTemplate, error) {
	// 提取模版IDs
	templateIDs := make([]int64, len(templates))
	for i, template := range templates {
		templateIDs[i] = template.ID
	}

	// 获取所有模版关联的版本
	versions, err := r.dao.GetTemplateVersionByTemplateIDs(ctx, templateIDs)
	if err != nil {
		return nil, err
	}

	// 提取版本IDs
	versionIDs := make([]int64, len(versions))
	for i, version := range versions {
		versionIDs[i] = version.ID
	}

	// 获取所有版本关联的供应商
	providers, err := r.dao.GetProviderByVersionIDs(ctx, versionIDs)
	if err != nil {
		return nil, err
	}

	// 构建版本ID到供应商列表的映射
	versionToProviders := make(map[int64][]domain.ChannelTemplateProvider)
	for i := range providers {
		domainProvider := r.toProviderDomain(providers[i])
		versionToProviders[providers[i].TemplateVersionID] = append(versionToProviders[providers[i].TemplateVersionID], domainProvider)
	}

	// 构建模板ID到版本列表的映射
	templateToVersions := make(map[int64][]domain.ChannelTemplateVersion)
	for i := range versions {
		domainVersion := r.toVersionDomain(versions[i])
		// 添加版本关联的供应商
		domainVersion.Providers = versionToProviders[versions[i].ID]
		templateToVersions[versions[i].ChannelTemplateID] = append(templateToVersions[versions[i].ChannelTemplateID], domainVersion)
	}

	// 构建最终的领域模型列表
	result := make([]domain.ChannelTemplate, 0, len(templates))
	for _, template := range templates {
		domainTemplate := r.toTemplateDomain(template)
		// 添加模版关联的版本
		domainTemplate.Versions = templateToVersions[template.ID]
		result = append(result, domainTemplate)
	}
	return result, nil
}

func (r *channelTemplateRepository) toProviderDomain(provider dao.ChannelTemplateProvider) domain.ChannelTemplateProvider {
	return domain.ChannelTemplateProvider{
		ID:                       provider.ID,
		TemplateID:               provider.TemplateID,
		TemplateVersionID:        provider.TemplateVersionID,
		ProviderID:               provider.ProviderID,
		ProviderName:             provider.ProviderName,
		ProviderChannel:          domain.Channel(provider.ProviderChannel),
		RequestID:                provider.RequestID,
		ProviderTemplateID:       provider.ProviderTemplateID,
		AuditStatus:              domain.AuditStatus(provider.AuditStatus),
		RejectReason:             provider.RejectReason,
		LastReviewSubmissionTime: provider.LastREeviewSubmissionTime,
		Ctime:                    provider.Ctime,
		Utime:                    provider.Utime,
	}
}

func (r *channelTemplateRepository) toProviderEntity(provider domain.ChannelTemplateProvider) dao.ChannelTemplateProvider {
	return dao.ChannelTemplateProvider{
		ID:                        provider.ID,
		TemplateID:                provider.TemplateID,
		TemplateVersionID:         provider.TemplateVersionID,
		ProviderID:                provider.ProviderID,
		ProviderName:              provider.ProviderName,
		ProviderChannel:           provider.ProviderChannel.String(),
		RequestID:                 provider.RequestID,
		ProviderTemplateID:        provider.ProviderTemplateID,
		AuditStatus:               provider.AuditStatus.String(),
		RejectReason:              provider.RejectReason,
		LastREeviewSubmissionTime: provider.LastReviewSubmissionTime,
	}
}

func (r *channelTemplateRepository) toVersionDomain(version dao.ChannelTemplateVersion) domain.ChannelTemplateVersion {
	return domain.ChannelTemplateVersion{
		Id:                       version.ID,
		ChannelTemplateID:        version.ChannelTemplateID,
		Name:                     version.Name,
		Signature:                version.Signature,
		Content:                  version.Content,
		Remark:                   version.Remark,
		AuditId:                  version.AuditID,
		AuditorId:                version.AuditorID,
		AuditTime:                version.AuditTime,
		AuditStatus:              domain.AuditStatus(version.AuditStatus),
		RejectReason:             version.RejectReason,
		LastReviewSubmissionTime: version.LastREeviewSubmissionTime,
		Ctime:                    version.Ctime,
		Utime:                    version.Utime,
	}
}

func (r *channelTemplateRepository) toVersionEntity(version domain.ChannelTemplateVersion) dao.ChannelTemplateVersion {
	return dao.ChannelTemplateVersion{
		ID:                        version.Id,
		ChannelTemplateID:         version.ChannelTemplateID,
		Name:                      version.Name,
		Signature:                 version.Signature,
		Content:                   version.Content,
		Remark:                    version.Remark,
		AuditID:                   version.AuditorId,
		AuditorID:                 version.AuditorId,
		AuditTime:                 version.AuditTime,
		AuditStatus:               version.AuditStatus.String(),
		RejectReason:              version.RejectReason,
		LastREeviewSubmissionTime: version.LastReviewSubmissionTime,
	}
}

func (r *channelTemplateRepository) toTemplateDomain(template dao.ChannelTemplate) domain.ChannelTemplate {
	return domain.ChannelTemplate{
		ID:              template.ID,
		OwnerID:         template.OwnerID,
		OwnerType:       domain.OwnerType(template.OwnerType),
		Name:            template.Name,
		Description:     template.Description,
		Channel:         domain.Channel(template.Channel),
		BusinessType:    domain.BusinessType(template.BusinessType),
		ActiveVersionID: template.ActiveVersionID,
		Ctime:           template.Ctime,
		Utime:           template.Utime,
	}
}

func (r *channelTemplateRepository) toTemplateEntity(template domain.ChannelTemplate) dao.ChannelTemplate {
	return dao.ChannelTemplate{
		ID:              template.ID,
		OwnerID:         template.OwnerID,
		OwnerType:       template.OwnerType.String(),
		Name:            template.Name,
		Description:     template.Description,
		Channel:         template.Channel.String(),
		BusinessType:    template.BusinessType.ToInt64(),
		ActiveVersionID: template.ActiveVersionID,
	}
}
