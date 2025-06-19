package manage

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ecodeclub/ekit/slice"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/repository"
	"go-notification/internal/service/audit"
	"go-notification/internal/service/provider/manage"
	"go-notification/internal/service/provider/sms/client"
	"regexp"
	"time"
)

// ChannelTemplateService 提供模版管理的服务接口
//
//go:generate mockgen -source=./manage.go -destination=../mocks/manage.mock.go -package=templatemocks -typed ChannelTemplateService
type ChannelTemplateService interface {
	// GetTemplatesByOwner 获取指定所有者的模板列表
	GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error)

	// GetTemplateByIDAndProviderInfo 根据模板ID和供应商信息获取模板
	GetTemplateByIDAndProviderInfo(ctx context.Context, templateID int64, providerName string, channel domain.Channel) (domain.ChannelTemplate, error)

	// GetTemplateByID 根据模板ID获取模板
	GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error)

	// CreateTemplate 创建模板
	CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error)

	// UpdateTemplate 更新模板
	UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error

	// PublishTemplate 发布模板
	PublishTemplate(ctx context.Context, templateID, versionID int64) error

	// 模板版本相关方法

	// ForkVersion 基于已有版本创建模板版本
	ForkVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error)

	// UpdateVersion 更新模板版本
	UpdateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error

	// SubmitForInternalReview 提交内部审核
	SubmitForInternalReview(ctx context.Context, versionID int64) error

	// BatchUpdateVersionAuditStatus 批量更新版本审核状态
	BatchUpdateVersionAuditStatus(ctx context.Context, versions []domain.ChannelTemplateVersion) error

	// 供应商相关方法

	// BatchSubmitForProviderReview 批量提交供应商审核
	BatchSubmitForProviderReview(ctx context.Context, versionID []int64) error

	// GetPendingOrInReviewProviders 获取未审核或审核中的供应商关联
	GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error)

	// BatchQueryAndUpdateProviderAuditInfo 批量查询并更新供应商审核信息
	BatchQueryAndUpdateProviderAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error
}

// templateService 实现 ChannelTemplateService 接口，提供模版管理的具体实现
type templateService struct {
	repo        repository.ChannelTemplateRepository
	providerSvc manage.Service
	auditSvc    audit.Service
	smsClients  map[string]client.Client
}

func NewTemplateService(repo repository.ChannelTemplateRepository, providerSvc manage.Service, auditSvc audit.Service, smsClients map[string]client.Client) ChannelTemplateService {
	return &templateService{repo: repo, providerSvc: providerSvc, auditSvc: auditSvc, smsClients: smsClients}
}

// 模版相关方法

func (t *templateService) GetTemplatesByOwner(ctx context.Context, ownerID int64, ownerType domain.OwnerType) ([]domain.ChannelTemplate, error) {
	if ownerID <= 0 {
		return nil, fmt.Errorf("%w: 业务方ID必须大于0", errs.ErrInvalidParameter)
	}

	if !ownerType.IsValid() {
		return nil, fmt.Errorf("%w: 所有者类型", errs.ErrInvalidParameter)
	}

	// 从存储层获取模版列表
	templates, err := t.repo.GetTemplatesByOwner(ctx, ownerID, ownerType)
	if err != nil {
		return nil, fmt.Errorf("获取模版列表失败: %w", err)
	}
	return templates, nil
}

func (t *templateService) GetTemplateByIDAndProviderInfo(ctx context.Context, templateID int64, providerName string, channel domain.Channel) (domain.ChannelTemplate, error) {
	// 1. 获取模版基本信息
	template, err := t.repo.GetTemplateByID(ctx, templateID)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	if template.ID == 0 {
		return domain.ChannelTemplate{}, errs.ErrTemplateNotFound
	}

	// 2. 获取指定的版本信息
	version, err := t.repo.GetTemplateVersionByID(ctx, template.ActiveVersionID)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	if version.AuditStatus != domain.AuditStatusApproved {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: versionID=%d", errs.ErrTemplateVersionNotApprovedByPlatform, version.Id)
	}

	// 3. 获取供应商信息
	providers, err := t.repo.GetProviderByNameAndChannel(ctx, templateID, version.Id, providerName, channel)
	if err != nil {
		return domain.ChannelTemplate{}, err
	}

	if len(providers) == 0 {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: providerName=%s, channel=%s", errs.ErrTemplateNotFound, providerName, channel)
	}

	// 4. 组装完整模版
	version.Providers = providers
	template.Versions = []domain.ChannelTemplateVersion{version}
	return template, nil
}

func (t *templateService) GetTemplateByID(ctx context.Context, templateID int64) (domain.ChannelTemplate, error) {
	return t.repo.GetTemplateByID(ctx, templateID)
}

func (t *templateService) CreateTemplate(ctx context.Context, template domain.ChannelTemplate) (domain.ChannelTemplate, error) {
	// 参数校验
	if err := template.Validate(); err != nil {
		return domain.ChannelTemplate{}, err
	}

	// 设置初始状态
	template.ActiveVersionID = 0 // 默认无活跃版本

	// 创建模版
	created, err := t.repo.CreateTemplate(ctx, template)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 创建模板失败: %w", errs.ErrCreateTemplateFailed, err)
	}

	// 创建模版版本，填充数据
	version := domain.ChannelTemplateVersion{
		ChannelTemplateID: created.ID,
		Name:              "版本名称，比如v1.0.0",
		Signature:         "提前配置好的可用的短信签名或者Email收件人",
		Content:           "模版变量使用${code}格式，也可以没有变量",
		Remark:            "模版使用场景或者用途说明，有利于供应商审核通过",
	}

	createdVersion, err := t.repo.CreateTemplateVersion(ctx, version)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 创建模板版本: %w", errs.ErrCreateTemplateFailed, err)
	}

	// 为每个供应商创建关联
	providers, err := t.providerSvc.GetByChannel(ctx, template.Channel)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 获取供应商列表失败: %w", errs.ErrCreateTemplateFailed, err)
	}
	if len(providers) == 0 {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 渠道 %s 没有可用的供应商，联系管理员配置供应商", errs.ErrCreateTemplateFailed, template.Channel)
	}
	templateProviders := make([]domain.ChannelTemplateProvider, 0, len(providers))
	for i := range providers {
		templateProviders = append(templateProviders, domain.ChannelTemplateProvider{
			TemplateID:        created.ID,
			TemplateVersionID: createdVersion.Id,
			ProviderID:        providers[i].ID,
			ProviderName:      providers[i].Name,
			ProviderChannel:   providers[i].Channel,
		})
	}
	createdProviders, err := t.repo.BatchCreateTemplateProviders(ctx, templateProviders)
	if err != nil {
		return domain.ChannelTemplate{}, fmt.Errorf("%w: 创建模板供应商关联失败: %w", errs.ErrCreateTemplateFailed, err)
	}

	// 组合
	createdVersion.Providers = createdProviders
	created.Versions = []domain.ChannelTemplateVersion{createdVersion}
	return created, nil
}

func (t *templateService) UpdateTemplate(ctx context.Context, template domain.ChannelTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("%w: 模板名称", errs.ErrInvalidParameter)
	}

	if template.Description == "" {
		return fmt.Errorf("%w: 模板描述", errs.ErrInvalidParameter)
	}

	if !template.BusinessType.IsValid() {
		return fmt.Errorf("%w: 业务类型", errs.ErrInvalidParameter)
	}

	if err := t.repo.UpdateTemplate(ctx, template); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateFailed, err)
	}

	return nil
}

func (t *templateService) PublishTemplate(ctx context.Context, templateID, versionID int64) error {
	if templateID <= 0 {
		return fmt.Errorf("%w: 模板id必须大于0", errs.ErrInvalidParameter)
	}

	if versionID <= 0 {
		return fmt.Errorf("%w: 版本ID必须大于0", errs.ErrInvalidParameter)
	}

	// 检查是否存在并且已通过内部审核
	version, err := t.repo.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return err
	}

	// 确认版本属于该模板
	if version.ChannelTemplateID != templateID {
		return fmt.Errorf("%w: %w", errs.ErrInvalidParameter, errs.ErrTemplateAndVersionMisMatch)
	}

	// 检查版本是否通过内部审核
	if version.AuditStatus != domain.AuditStatusApproved {
		return fmt.Errorf("%w: %w", errs.ErrInvalidParameter, errs.ErrTemplateVersionNotApprovedByPlatform)
	}

	// 检查是否有通过供应商审核的记录
	providers, err := t.repo.GetApprovedProvidersByTemplateIDAndVersionID(ctx, templateID, versionID)
	if err != nil {
		return err
	}
	if len(providers) == 0 {
		return fmt.Errorf("%w", errs.ErrTemplateVersionNotApprovedByPlatform)
	}

	// 设置活跃版本
	err = t.repo.SetTemplateActiveVersion(ctx, templateID, versionID)
	if err != nil {
		return fmt.Errorf("发布模板失败: %w", err)
	}
	return nil
}

// 版本相关方法

func (t *templateService) ForkVersion(ctx context.Context, versionID int64) (domain.ChannelTemplateVersion, error) {
	return t.repo.ForkTemplateVersion(ctx, versionID)
}

func (t *templateService) UpdateVersion(ctx context.Context, version domain.ChannelTemplateVersion) error {
	// 参数校验
	if version.Id <= 0 {
		return fmt.Errorf("%w: 版本ID必须大于0", errs.ErrInvalidParameter)
	}

	// 获取当前版本
	currentVersion, err := t.repo.GetTemplateVersionByID(ctx, version.Id)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateVersionFailed, err)
	}

	// 检查版本状态，只有PENDING或REJECTED状态的版本才能修改
	if currentVersion.AuditStatus != domain.AuditStatusPending && currentVersion.AuditStatus != domain.AuditStatusRejected {
		return fmt.Errorf("%w: %w: 只有待审核或拒绝状态的版本可以修改", errs.ErrUpdateTemplateVersionFailed, errs.ErrInvalidOperation)
	}

	// 允许更新部分字段
	updateVersion := domain.ChannelTemplateVersion{
		Id:        version.Id,
		Name:      version.Name,
		Signature: version.Signature,
		Content:   version.Content,
		Remark:    version.Remark,
	}

	// 更新版本
	err = t.repo.UpdateTemplateVersion(ctx, updateVersion)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateVersionFailed, err)
	}
	return nil
}

func (t *templateService) SubmitForInternalReview(ctx context.Context, versionID int64) error {
	if versionID <= 0 {
		return fmt.Errorf("%w: 版本ID必须大于0", errs.ErrInvalidParameter)
	}

	// 获取版本信息
	version, err := t.repo.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	if version.AuditStatus == domain.AuditStatusInReview || version.AuditStatus == domain.AuditStatusApproved {
		return nil
	}

	// 获取模板信息
	template, err := t.repo.GetTemplateByID(ctx, version.ChannelTemplateID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	// 获取版本关联的供应商
	providers, err := t.repo.GetProvidersByTemplateIDAndVersionID(ctx, template.ID, version.Id)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	content, err := t.getJSONAuditContent(template, version, providers)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	// 创建审核记录
	auditID, err := t.auditSvc.CreateAudit(ctx, domain.Audit{
		ResourceID:   version.Id,
		ResourceType: domain.ResourceTypeTemplate,
		Content:      content,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	// 更新版本审核状态
	updateVersions := []domain.ChannelTemplateVersion{
		{
			Id:                       version.Id,
			AuditId:                  auditID,
			AuditStatus:              domain.AuditStatusInReview,
			LastReviewSubmissionTime: time.Now().Unix(),
		},
	}

	err = t.repo.BatchUpdateTemplateVersionAuditInfo(ctx, updateVersions)
	if err != nil {
		return fmt.Errorf("%w: 更新版本审核状态失败: %w", errs.ErrSubmitVersionForInternalReviewFailed, err)
	}

	return nil
}

func (t *templateService) getJSONAuditContent(template domain.ChannelTemplate, version domain.ChannelTemplateVersion, providers []domain.ChannelTemplateProvider) (string, error) {
	content := domain.AuditContent{
		OwnerID:      template.OwnerID,
		OwnerType:    template.OwnerType.String(),
		Name:         template.Name,
		Description:  template.Description,
		Channel:      template.Channel.String(),
		BusinessType: template.BusinessType.String(),
		Version:      version.Name,
		Signature:    version.Signature,
		Content:      version.Content,
		Remark:       version.Remark,
		ProviderNames: slice.Map(providers, func(_ int, src domain.ChannelTemplateProvider) string {
			return src.ProviderName
		}),
	}
	b, err := json.Marshal(content)
	if err != nil {
		return "", fmt.Errorf("序列化审核内容失败: %w", err)
	}
	return string(b), nil
}

func (t *templateService) BatchUpdateVersionAuditStatus(ctx context.Context, versions []domain.ChannelTemplateVersion) error {
	if len(versions) == 0 {
		return nil
	}
	if err := t.repo.BatchUpdateTemplateVersionAuditInfo(ctx, versions); err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateVersionAuditStatusFailed, err)
	}
	return nil
}

func (t *templateService) BatchSubmitForProviderReview(ctx context.Context, versionIDs []int64) error {
	for i := range versionIDs {
		_ = t.submitForProviderReview(ctx, versionIDs[i])
	}
	return nil
}

func (t *templateService) submitForProviderReview(ctx context.Context, versionID int64) error {
	// 获取版本信息
	version, err := t.repo.GetTemplateVersionByID(ctx, versionID)
	if err != nil {
		return err
	}

	// 获取模板信息
	template, err := t.repo.GetTemplateByID(ctx, version.ChannelTemplateID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	// 获取供应商关联信息
	providers, err := t.repo.GetProvidersByTemplateIDAndVersionID(ctx, template.ID, versionID)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	for i := range providers {
		if providers[i].AuditStatus == domain.AuditStatusPending ||
			providers[i].AuditStatus == domain.AuditStatusRejected {
			_ = t.submit(ctx, template, version, providers[i])
		}
	}
	return nil
}

func (t *templateService) submit(ctx context.Context, template domain.ChannelTemplate, version domain.ChannelTemplateVersion, provider domain.ChannelTemplateProvider) error {
	// 当前仅支持SMS渠道
	if provider.ProviderChannel != domain.ChannelSMS {
		return nil
	}
	// 获取对应的SMS客户端
	cli, err := t.getSMSClient(provider.ProviderName)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	// 构建供应商审核请求并调用
	resp, err := cli.CreateTemplate(client.CreateTemplateReq{
		TemplateName:    version.Name,
		TemplateContent: t.replacePlaceholders(version.Content, provider),
		TemplateType:    client.TemplateType(template.BusinessType),
		Remark:          version.Remark,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}

	// 更新供应商关联
	err = t.repo.UpdateTemplateProviderAuditInfo(ctx, domain.ChannelTemplateProvider{
		ID:                       provider.ID,
		RequestID:                resp.RequestID,
		ProviderTemplateID:       resp.TemplateID,
		AuditStatus:              domain.AuditStatusInReview,
		LastReviewSubmissionTime: time.Now().Unix(),
	})
	if err != nil {
		return fmt.Errorf("%w: 更新供应商关联失败: %w", errs.ErrSubmitVersionForProviderReviewFailed, err)
	}
	return nil
}

func (t *templateService) getSMSClient(providerName string) (client.Client, error) {
	smsClient, ok := t.smsClients[providerName]
	if !ok {
		return nil, fmt.Errorf("未找到对应的供应商客户端")
	}
	return smsClient, nil
}

func (t *templateService) replacePlaceholders(content string, provider domain.ChannelTemplateProvider) string {
	// 仅腾讯云需要替换占位符
	if provider.ProviderName != "tencentcloud" {
		return content
	}
	re := regexp.MustCompile(`\$\{[^}]+\}`)
	counter := 0
	output := re.ReplaceAllStringFunc(content, func(_ string) string {
		counter++
		return fmt.Sprintf("{%d}", counter)
	})
	return output
}

func (t *templateService) GetPendingOrInReviewProviders(ctx context.Context, offset, limit int, utime int64) (providers []domain.ChannelTemplateProvider, total int64, err error) {
	return t.repo.GetPendingOrInReviewProviders(ctx, offset, limit, utime)
}

func (t *templateService) BatchQueryAndUpdateProviderAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	if len(providers) == 0 {
		return nil
	}

	// 按渠道和供应商名称分组处理
	groupedProviders := make(map[domain.Channel]map[string][]domain.ChannelTemplateProvider)
	for i := range providers {
		channel := providers[i].ProviderChannel
		name := providers[i].ProviderName
		if _, ok := groupedProviders[channel]; !ok {
			groupedProviders[channel] = make(map[string][]domain.ChannelTemplateProvider)
		}
		groupedProviders[channel][name] = append(groupedProviders[channel][name], providers[i])
	}

	// 处理每个渠道的供应商
	for channel := range groupedProviders {
		for name := range groupedProviders[channel] {
			if channel.IsSMS() {
				if err := t.batchQueryAndUpdateSMSProvidersAuditInfo(ctx, groupedProviders[channel][name]); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (t *templateService) batchQueryAndUpdateSMSProvidersAuditInfo(ctx context.Context, providers []domain.ChannelTemplateProvider) error {
	const first = 0
	smsClient, err := t.getSMSClient(providers[first].ProviderName)
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateProviderAuditStatusFailed, err)
	}

	// 获取供应商侧的模版ID 和 映射关系
	templateIDs := make([]string, 0, len(providers))
	providerMap := make(map[string]domain.ChannelTemplateProvider, len(providers))
	for i := range providers {
		templateIDs = append(templateIDs, providers[i].ProviderTemplateID)
		providerMap[providers[i].ProviderTemplateID] = providers[i]
	}

	// 批量查询模版状态
	results, err := smsClient.BatchQueryTemplateStatus(client.BatchQueryTemplateStatusReq{
		TemplateIDs: templateIDs,
	})
	if err != nil {
		return fmt.Errorf("%w: %w", errs.ErrUpdateTemplateProviderAuditStatusFailed, err)
	}

	// 更新对应的状态信息
	updates := make([]domain.ChannelTemplateProvider, 0, len(results.Results))
	for i := range results.Results {
		p, ok := providerMap[results.Results[i].TemplateID]
		if !ok {
			continue
		}
		p.RequestID = results.Results[i].RequestID
		p.AuditStatus = results.Results[i].AuditStatus.ToDomain()
		p.RejectReason = results.Results[i].Reason
		updates = append(updates, p)
	}
	return t.repo.BatchUpdateTemplateProvidersAuditInfo(ctx, updates)
}
