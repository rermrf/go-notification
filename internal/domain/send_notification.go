package domain

import (
	"fmt"
	"go-notification/internal/errs"
	"time"
)

// SendStrategyType 发送策略类型
type SendStrategyType string

const (
	SendStrategyImmediate SendStrategyType = "IMMEDIATE" // 立即发送
	SendStrategyDelayed   SendStrategyType = "DELAYED"   // 延迟发送
	SendStrategyScheduled SendStrategyType = "SCHEDULED" // 定时发送
	SendStrategyWindow    SendStrategyType = "WINDOW"    // 窗口发送
	SendStrategyDeadline  SendStrategyType = "DEADLINE"  // 截止日期发送
)

// SendStrategyConfig 发送策略配置
type SendStrategyConfig struct {
	Type         SendStrategyType `json:"type"`         // 策略类型
	Delay        time.Duration    `json:"delay"`        // 延迟发送时间
	ScheduleTime time.Time        `json:"scheduleTime"` // 定时发送时间
	StartTime    time.Time        `json:"startTime"`    // 窗口发送开始时间
	EndTime      time.Time        `json:"endTime"`      // 窗口发送结束时间
	DeadlineTime time.Time        `json:"deadlineTime"` // 截止日期
}

// SendTimeWindow 计算最早发送时间和最晚发送时间
func (e SendStrategyConfig) SendTimeWindow() (stime, etime time.Time) {
	switch e.Type {
	case SendStrategyImmediate:
		now := time.Now()
		const defaultEndDuration = 30 * time.Minute
		return now, now.Add(defaultEndDuration)
	case SendStrategyDelayed:
		now := time.Now()
		return now, now.Add(e.Delay)
	case SendStrategyDeadline:
		now := time.Now()
		return now, e.DeadlineTime
	case SendStrategyWindow:
		return e.StartTime, e.EndTime
	case SendStrategyScheduled:
		const scheduledTimeTolerance = 3 * time.Second
		return e.ScheduleTime.Add(-scheduledTimeTolerance), e.ScheduleTime
	default:
		now := time.Now()
		return now, now
	}
}

func (e SendStrategyConfig) Validate() error {
	// 校验策略相关字段
	switch e.Type {
	case SendStrategyImmediate:
		return nil
	case SendStrategyDelayed:
		if e.Delay <= 0 {
			return fmt.Errorf("%w: 延迟发送策略需要指定正数的延迟秒数", errs.ErrInvalidParameter)
		}
	case SendStrategyDeadline:
		if e.DeadlineTime.IsZero() || e.DeadlineTime.Before(time.Now()) {
			return fmt.Errorf("%w: 截至日期发送策略需要指定未来的发送时间", errs.ErrInvalidParameter)
		}
	case SendStrategyWindow:
		if e.StartTime.IsZero() || e.StartTime.After(e.EndTime) {
			return fmt.Errorf("%w: 时间窗口发送策略需要指定有效的开始时间和结束时间", errs.ErrInvalidParameter)
		}
	case SendStrategyScheduled:
		if e.ScheduleTime.IsZero() || e.ScheduleTime.Before(time.Now()) {
			return fmt.Errorf("%w: 定时发送策略需要指定未来的发送时间", errs.ErrInvalidParameter)
		}
	}
	return nil
}

// SendResponse 发送响应
type SendResponse struct {
	NotificationID int64
	Status         SendStatus
}

// BatchSendResponse 批量发送响应
type BatchSendResponse struct {
	Results []SendResponse
}

// BatchSendAsyncResponse 批量异步发送响应
type BatchSendAsyncResponse struct {
	NotificationIDs []int64 // 生成的通知ID列表
}
