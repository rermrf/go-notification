package ioc

import (
	"go-notification/internal/pkg/task"
	"go-notification/internal/service/notification"
	"go-notification/internal/service/notification/callback"
	"go-notification/internal/service/scheduler"
)

func InitTasks(
	t1 *callback.AsyncRequestResultCallbackTask,
	t2 scheduler.NotificationScheduler,
	t3 *notification.SendingTimeoutTask,
	t4 *notification.TxCheckTask,
) []task.Task {
	var tasks = make([]task.Task, 0)
	tasks = append(tasks, t1)
	tasks = append(tasks, t2)
	tasks = append(tasks, t3)
	tasks = append(tasks, t4)
	return tasks
}
