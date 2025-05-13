package domain

// Channel 发送渠道
type Channel string

const (
	ChannelSMS    Channel = "SMS"     // 短信
	ChannelEmail  Channel = "EMAIL"   // 邮件
	ChannelInApp  Channel = "IN_APP"  // 站内信
)

func (c Channel) String() string {
	return string(c)
}

func (c Channel) IsValid() bool {
	return c == ChannelSMS || c == ChannelEmail || c == ChannelInApp
}

func (c Channel) IsSMS() bool {
	return c == ChannelSMS
}

func (c Channel) IsEmail() bool {
	return c == ChannelEmail
}

func (c Channel) IsInApp() bool {
	return c == ChannelInApp
}

