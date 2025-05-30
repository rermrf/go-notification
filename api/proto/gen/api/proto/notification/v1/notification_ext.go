package notificationv1

import (
	"errors"
	"fmt"
	"strconv"
	"time"
)

func (x *Notification) FindReceivers() []string {
	receivers := x.Receivers
	if x.Receiver != "" {
		receivers = append(receivers, x.Receiver)
	}
	return receivers
}

func (x *Notification) CustomValidate() error {
	switch val := x.SendStrategy.StrategyType.(type) {
	case *SendStrategy_Delayed:
		// 延迟时间超过一天，就返回错误
		if time.Duration(val.Delayed.DelaySeconds)*time.Second > time.Hour*24 {
			return errors.New("delayed send strategy is too long")
		}
	}
	return nil
}

// ReceiverAsUid 比如说站内信之类的，receivers 其实是 uid
func (x *Notification) ReceiverAsUid() ([]int64, error) {
	receivers := x.FindReceivers()
	result := make([]int64, 0, len(receivers))
	for _, receiver := range receivers {
		val, err := strconv.ParseInt(receiver, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("必须是数字: %w", err)
		}
		result = append(result, val)
	}
	return result, nil
}

type NotificationCarrier interface {
	GetNotifications() []*Notification
}

func (x *SendNotificationRequest) GetNotifications() []*Notification {
	n := x.GetNotification()
	if n != nil {
		return []*Notification{n}
	}
	return nil
}

func (x *SendNotificationAsyncRequest) GetNotifications() []*Notification {
	n := x.GetNotification()
	if n != nil {
		return []*Notification{n}
	}
	return nil
}

type IdempotencyCarrier interface {
	GetIdempotencyKeys() []string
}

func (x *SendNotificationRequest) GetIdempotencyKeys() []string {
	n := x.GetNotification()
	if n != nil {
		return []string{n.Key}
	}
	return nil
}

func (x *SendNotificationAsyncRequest) GetIdempotencyKeys() []string {
	n := x.GetNotification()
	if n != nil {
		return []string{n.Key}
	}
	return nil
}

func (x *SendNotificationBatchAsyncRequest) GetIdempotencyKeys() []string {
	if x != nil {
		notifications := x.GetNotifications()
		var results []string
		for _, notification := range notifications {
			results = append(results, notification.Key)
		}
		return results
	}
	return nil
}

func (x *SendNotificationBatchRequest) GetIdempotencyKeys() []string {
	if x != nil {
		notifications := x.GetNotifications()
		var results []string
		for _, notification := range notifications {
			results = append(results, notification.Key)
		}
		return results
	}
	return nil
}
