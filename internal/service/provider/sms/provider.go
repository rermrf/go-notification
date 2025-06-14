package sms

import "go-notification/internal/service/template/manage"

type smsProvider struct {
	name        string
	templateSvc manage.ChannelTemplateService
	client      client
}
