package dao

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
	AuditID int64 `gorm:"type:BIGINT;NOT NULL;DEFAULT:0;comment:'审核表ID，0表示尚未提交审核或者未拿到审核结果'"`
}
