package ioc

import (
	"context"
	"github.com/robfig/cron/v3"
	"go-notification/internal/pkg/grpcx"
	"go-notification/internal/pkg/task"
)

type App struct {
	GrpcServer *grpcx.Server
	Tasks      []task.Task
	Cron       *cron.Cron
}

func (a *App) StartTasks(ctx context.Context) {
	for _, t := range a.Tasks {
		go func(t task.Task) {
			t.Start(ctx)
		}(t)
	}
}
