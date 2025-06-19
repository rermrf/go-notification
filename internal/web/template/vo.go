package template

type ListTemplatesReq struct {
	OwnerID   int64  `json:"ownerId"` // 商品信息
	OwnerType string `json:"ownerType"`
}

type ListTemplatesResp struct {
	Templates []ChannelTemplate `json:"templates"`
}

// ChannelTemplate 渠道模板
type ChannelTemplate struct {
	ID              int64  `json:"id"`              // 模板ID
	OwnerID         int64  `json:"ownerId"`         // 拥有者ID，用户ID或者部门ID
	OwnerType       string `json:"ownerType"`       // 拥有者类型
	Name            string `json:"name"`            // 模版名称
	Description     string `json:"description"`     // 模版描述
	Channel         string `json:"channel"`         // 渠道类型
	BusinessType    int64  `json:"businessType"`    // 业务类型
	ActiveVersionID int64  `json:"activeVersionId"` // 活跃版本ID，0标识无活跃版本
	Ctime           int64  `json:"ctime"`           // 创建时间
	Utime           int64  `json:"utime"`           // 更新时间

	Versions []ChannelTemplateVersion `json:"versions"`
}

// ChannelTemplateVersion 渠道模版版本
type ChannelTemplateVersion struct {
	ID                       int64  `json:"id"`                       // 版本ID
	ChannelTemplateID        int64  `json:"channelTemplateId"`        // 模版ID
	Name                     string `json:"name"`                     // 模版名称
	Signature                string `json:"signature"`                // 签名
	Content                  string `json:"content"`                  // 模版内容
	Remark                   string `json:"remark"`                   // 申请说明
	AuditID                  int64  `json:"auditId"`                  // 审核记录ID
	AuditorID                int64  `json:"auditorId"`                // 审核人ID
	AuditTime                int64  `json:"auditTime"`                // 审核时间
	AuditStatus              string `json:"auditStatus"`              // 审核状态
	RejectReason             string `json:"rejectReason"`             // 拒绝原因
	LastReviewSubmissionTime int64  `json:"lastReviewSubmissionTime"` // 上一次提交审核时间
	Ctime                    int64  `json:"ctime"`                    // 创建时间
	Utime                    int64  `json:"utime"`                    // 更新时间

	Providers []ChannelTemplateProvider `json:"providers"` // 管理的所有供应商
}

// ChannelTemplateProvider 渠道模板供应商关联
type ChannelTemplateProvider struct {
	ID                       int64  `json:"id"`                       // 关联ID
	TemplateID               int64  `json:"templateId"`               // 模板ID
	TemplateVersionID        int64  `json:"templateVersionId"`        // 模版版本ID
	ProviderID               int64  `json:"providerId"`               // 供应商ID
	ProviderName             string `json:"providerName"`             // 供应商名称
	ProviderChannel          string `json:"providerChannel"`          // 供应商渠道类型
	RequestID                string `json:"requestId"`                // 审核请求ID
	ProviderTemplateID       string `json:"providerTemplateId"`       // 供应商侧模板ID
	AuditStatus              string `json:"auditStatus"`              // 审核状态
	RejectReason             string `json:"rejectReason"`             // 拒绝原因
	LastReviewSubmissionTime int64  `json:"lastReviewSubmissionTime"` // 上次提交审核时间
	Ctime                    int64  `json:"ctime"`                    // 创建时间
	Utime                    int64  `json:"utime"`                    // 更新时间
}

type CreateTemplateReq struct {
	OwnerID      int64  `json:"ownerId"`
	OwnerType    string `json:"ownerType"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Channel      string `json:"channel"`
	BusinessType int64  `json:"businessType"`
}

type CreateTemplateResp struct {
	Template ChannelTemplate `json:"template"`
}

type UpdateTemplateReq struct {
	TemplateID   int64  `json:"templateId"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	BusinessType int64  `json:"businessType"`
}

type PublishTemplateReq struct {
	TemplateID int64 `json:"templateId"`
	VersionID  int64 `json:"versionId"`
}

type ForkVersionReq struct {
	VersionID int64 `json:"versionId"`
}

type ForkVersionResp struct {
	TemplateVersion ChannelTemplateVersion `json:"templateVersion"`
}

type UpdateVersionReq struct {
	VersionID int64  `json:"versionId"`
	Name      string `json:"name"`
	Signature string `json:"signature"`
	Content   string `json:"content"`
	Remark    string `json:"remark"`
}

type SubmitForInternalReviewReq struct {
	VersionID int64 `json:"versionId"`
}
