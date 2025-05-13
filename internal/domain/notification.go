package domain

import (
	"encoding/json"
	"fmt"
	notificationv1 "go-notification/api/proto/gen/api/proto/notification/v1"
	"go-notification/internal/errs"
	"strconv"
	"time"
)

// Template 通知领域模型
type Template struct {
	ID        int64             `json:"id"`
	VersionID int64             `json:"versionId"`
	Params    map[string]string `json:"params"`

	Version int64 `json:"version"`
}

// SendStatus 通知状态
type SendStatus string

const (
	SendStatusPrepare   SendStatus = "PREPARE"   // 准备中
	SendStatusCanceled  SendStatus = "CANCELED"  // 已取消
	SendStatusPending   SendStatus = "PENDING"   // 待发送
	SendStatusSending   SendStatus = "SENDING"   // 待发送
	SendStatusSucceeded SendStatus = "SUCCEEDED" // 发送成功
	SendStatusFailed    SendStatus = "FAILED"    // 发送失败
)

func (s SendStatus) String() string {
	return string(s)
}

type Notification struct {
	ID                 string             `json:"id"`             // 通知唯一标识
	BizID              int64              `json:"bizId"`          // 业务唯一标识
	Key                string             `json:"key"`            // 业务内唯一标识
	Receivers          []string           `json:"receivers"`      // 接收者(手机/邮箱/用户ID)
	Channel            Channel            `json:"channel"`        // 发送渠道
	Template           Template           `json:"template"`       // 关联的模版
	Status             SendStatus         `json:"status"`         // 发送状态
	ScheduledSTime     time.Time          `json:"scheduledSTime"` // 计划发送开始时间
	ScheduledETime     time.Time          `json:"scheduledETime"` // 计划发送结束时间
	Version            int                `json:"version"`        // 版本号
	SendStrategyConfig SendStrategyConfig `json:"sendStrategyConfig"`
}

func (n *Notification) SetSendTime() {
	stime, etime := n.SendStrategyConfig.SendTimeWindow()
	n.ScheduledSTime = stime
	n.ScheduledETime = etime
}

func (n *Notification) IsImmediate() bool {
	return n.SendStrategyConfig.Type == SendStrategyImmediate
}

// ReplaceAsyncImmediate 如果是是立刻发送，就修改为默认的策略
func (n *Notification) ReplaceAsyncImmediate() {
	if n.IsImmediate() {
		n.SendStrategyConfig.DeadlineTime = time.Now().Add(time.Minute)
		n.SendStrategyConfig.Type = SendStrategyDeadline
	}
}

func (n *Notification) Validate() error {
	if n.BizID <= 0 {
		return fmt.Errorf("%w: 业务ID", errs.ErrInvalidParameter)
	}

	if n.Key == "" {
		return fmt.Errorf("%w: 业务内唯一标识", errs.ErrInvalidParameter)
	}

	if len(n.Receivers) == 0 {
		return fmt.Errorf("%w: 接收者", errs.ErrInvalidParameter)
	}

	if !n.Channel.IsValid() {
		return fmt.Errorf("%w: 渠道类型", errs.ErrInvalidParameter)
	}

	if n.Template.ID <= 0 {
		return fmt.Errorf("%w: 模板ID", errs.ErrInvalidParameter)
	}

	if n.Template.VersionID <= 0 {
		return fmt.Errorf("%w: 模板版本ID", errs.ErrInvalidParameter)
	}

	if len(n.Template.Params) == 0 {
		return fmt.Errorf("%w: 模板参数", errs.ErrInvalidParameter)
	}

	if err := n.SendStrategyConfig.Validate(); err != nil {
		return err
	}

	return nil
}

func (n *Notification) IsValidBizID() error {
	if n.BizID <= 0 {
		return fmt.Errorf("%w: 业务ID", errs.ErrInvalidParameter)
	}
	return nil
}

func (n *Notification) MarshalReceivers() (string, error) {
	return n.marshal(n.Receivers)
}

func (n *Notification) MarshalTemplateParms() (string, error) {
	return n.marshal(n.Template.Params)
}

func (n *Notification) marshal(v any) (string, error) {
	jsonBytes, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(jsonBytes), nil
}

func NewNotificationFromAPI(n *notificationv1.Notification) (Notification, error) {
	if n == nil {
		return Notification{}, fmt.Errorf("%w: 通知信息不能为空", errs.ErrInvalidParameter)
	}

	tid, err := strconv.ParseInt(n.TemplateId, 10, 64)
	if err != nil {
		return Notification{}, fmt.Errorf("%w: 模板ID: %s", errs.ErrInvalidParameter, n.TemplateId)
	}

	channel, err := getDomainChannel(n)
	if err != nil {
		return Notification{}, err
	}

	return Notification{
		Key:       n.Key,
		Receivers: n.GetReceivers(),
		Channel:   channel,
		Template: Template{
			ID:     tid,
			Params: n.TemplateParams,
		},
		SendStrategyConfig: getDomainSendStrategyConfig(n),
	}, nil
}

func getDomainSendStrategyConfig(n *notificationv1.Notification) SendStrategyConfig {
	// 构建发送策列
	sendStrategyType := SendStrategyImmediate // 默认立即发送
	var delaySeconds int64
	var scheduleTime time.Time
	var startTimeMilliseconds int64
	var endTimeMilliseconds int64
	var deadlineTime time.Time

	// 处理发送策略
	if n.SendStrategy != nil {
		switch s := n.SendStrategy.StrategyType.(type) {
		case *notificationv1.SendStrategy_Immediate:
			sendStrategyType = SendStrategyImmediate
		case *notificationv1.SendStrategy_Delayed:
			if s.Delayed != nil && s.Delayed.DelaySeconds > 0 {
				delaySeconds = s.Delayed.DelaySeconds
				sendStrategyType = SendStrategyDelayed
			}
		case *notificationv1.SendStrategy_Scheduled:
			if s.Scheduled != nil && s.Scheduled.SendTime != nil {
				scheduleTime = s.Scheduled.SendTime.AsTime()
				sendStrategyType = SendStrategyScheduled
			}
		case *notificationv1.SendStrategy_TimeWindow:
			if s.TimeWindow != nil {
				startTimeMilliseconds = s.TimeWindow.StartTimeMilliseconds
				endTimeMilliseconds = s.TimeWindow.EndTimeMilliseconds
				sendStrategyType = SendStrategyWindow
			}
		case *notificationv1.SendStrategy_Deadline:
			if s.Deadline != nil && s.Deadline.Deadline != nil {
				deadlineTime = s.Deadline.Deadline.AsTime()
				sendStrategyType = SendStrategyDeadline
			}
		}
	}
	return SendStrategyConfig{
		Type:         sendStrategyType,
		Delay:        time.Duration(delaySeconds) * time.Second,
		ScheduleTime: scheduleTime,
		StartTime:    time.Unix(startTimeMilliseconds, 0),
		EndTime:      time.Unix(endTimeMilliseconds, 0),
		DeadlineTime: deadlineTime,
	}
}

func getDomainChannel(n *notificationv1.Notification) (Channel, error) {
	switch n.Channel {
	case notificationv1.Channel_SMS:
		return ChannelSMS, nil
	case notificationv1.Channel_EMAIL:
		return ChannelEmail, nil
	case notificationv1.Channel_IN_APP:
		return ChannelInApp, nil
	default:
		return "", fmt.Errorf("%w: 无效的渠道类型", errs.ErrInvalidParameter)
	}
}
