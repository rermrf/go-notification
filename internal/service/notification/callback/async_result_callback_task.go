package callback

import (
	"context"
	"github.com/meoying/dlock-go"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/loopjob"
	"time"
)

type AsyncRequestResultCallbackTask struct {
	dclient     dlock.Client
	log         logger.Logger
	callbackSvc Service
	batchSize   int64
}

func NewAsyncRequestResultCallbackTask(dclient dlock.Client, callbackSvc Service, log logger.Logger) *AsyncRequestResultCallbackTask {
	const defaultBatchSize = int64(10)
	return &AsyncRequestResultCallbackTask{dclient: dclient, callbackSvc: callbackSvc, log: log, batchSize: defaultBatchSize}
}

func (a *AsyncRequestResultCallbackTask) Start(ctx context.Context) {
	const key = "notification_handling_async_request_result_callback"
	lj := loopjob.NewInfiniteLoop(a.dclient, a.log, a.HandleSendResult, key)
	lj.Run(ctx)
}

func (a *AsyncRequestResultCallbackTask) HandleSendResult(ctx context.Context) error {
	const minDuration = 3 * time.Second

	now := time.Now()

	err := a.callbackSvc.SendCallback(ctx, now.UnixMilli(), a.batchSize)
	if err != nil {
		return err
	}

	duration := time.Since(now)
	if duration < minDuration {
		time.Sleep(minDuration - duration)
	}

	return nil
}
