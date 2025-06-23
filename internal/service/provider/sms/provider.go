package sms

import (
	"context"
	"fmt"
	"go-notification/internal/domain"
	"go-notification/internal/errs"
	"go-notification/internal/service/provider"
	"go-notification/internal/service/provider/sms/client"
	"go-notification/internal/service/template/manage"
	"strings"
)

type smsProvider struct {
	name        string
	templateSvc manage.ChannelTemplateService
	client      client.Client
}

func NewSmsProvider(name string, templateSvc manage.ChannelTemplateService, client client.Client) provider.Provider {
	return &smsProvider{name: name, templateSvc: templateSvc, client: client}
}

func (s *smsProvider) Send(ctx context.Context, notification domain.Notification) (domain.SendResponse, error) {
	tmpl, err := s.templateSvc.GetTemplateByIDAndProviderInfo(ctx, notification.Template.ID, s.name, domain.ChannelSMS)
	if err != nil {
		return domain.SendResponse{}, errs.ErrSendNotificationFailed
	}

	activeVersion := tmpl.ActiveVersion()
	if activeVersion == nil {
		return domain.SendResponse{}, fmt.Errorf("%w: 无已发布模板", errs.ErrSendNotificationFailed)
	}

	const first = 0
	resp, err := s.client.Send(client.SendReq{
		PhoneNumbers:  notification.Receivers,
		SignName:      activeVersion.Signature,
		TemplateID:    activeVersion.Providers[first].ProviderTemplateID,
		TemplateParam: notification.Template.Params,
	})
	if err != nil {
		return domain.SendResponse{}, fmt.Errorf("%w: %w", errs.ErrSendNotificationFailed, err)
	}

	for _, status := range resp.PhoneNumbers {
		if !strings.EqualFold(status.Code, "OK") {
			return domain.SendResponse{}, fmt.Errorf("%w: Code = %s, Message = %s", errs.ErrSendNotificationFailed, status.Code, status.Message)
		}
	}

	return domain.SendResponse{
		NotificationID: notification.ID,
		Status:         domain.SendStatusSucceeded,
	}, nil
}
