package ioc

import (
	"github.com/robfig/cron/v3"
	"go-notification/internal/repository"
	"go-notification/internal/service/quota"
)

func InitCron(q *quota.MonthlyResetCron, bCfg repository.BusinessConfigRepository) *cron.Cron {
	c := cron.New(cron.WithSeconds())

	return c
}
