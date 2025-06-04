package sendstrategy

import "go-notification/internal/repository"

type ImmediateSendStrategy struct {
	repo   repository.NotificationRepository
	sender sender.NotificationSender
}
