package failover

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
	"go-notification/internal/pkg/database/monitor"
	"go-notification/internal/pkg/logger"
	"go-notification/internal/pkg/mq"
	"gorm.io/gorm"
	"time"
)

const (
	defaultSleepTime = 2 * time.Second
	poll             = 1000
)

type ConnPoolEventConsumer struct {
	consumer  *kafka.Consumer
	db        *gorm.DB
	dbMonitor monitor.DBMonitor
	log       logger.Logger

	topic string
}

func NewConnPoolEventConsumer(consumer *kafka.Consumer, db *gorm.DB, dbMonitor monitor.DBMonitor, log logger.Logger, topic string) *ConnPoolEventConsumer {
	return &ConnPoolEventConsumer{
		consumer:  consumer,
		db:        db,
		dbMonitor: dbMonitor,
		log:       log,
		topic:     topic,
	}
}

// Start 在后台携程中开始消费kafka消息
func (c *ConnPoolEventConsumer) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-ctx.Done():
				c.log.Info("ConnPoolEventConsumer 因上下文取消而停止")
				return
			default:
				err := c.Consume(ctx)
				if err != nil {
					c.log.Error("消费消息失败", logger.Error(err))
				}
			}
		}
	}()
	c.log.Info("正在启动ConnPoolEventConsumer，监听主题", logger.String("topic", FailoverTopic))
}

func (c *ConnPoolEventConsumer) Consume(ctx context.Context) error {
	// 检查数据库健康状态
	if !c.dbMonitor.Health() {
		// 获取当前消费者分配的分区
		assigned, err := c.consumer.Assignment()
		if err != nil {
			return fmt.Errorf("获取消费者已分配分区失败: %w", err)
		}
		// 如果有分配的分区，暂停它们
		if len(assigned) > 0 {
			if err := c.consumer.Pause(assigned); err != nil {
				return fmt.Errorf("暂停分区失败: %w", err)
			}
			c.log.Info("已暂停消费者分配的分区", logger.Int32("分区数量", int32(len(assigned))))

			// 等待2秒
			time.Sleep(defaultSleepTime)

			// 恢复分区，即使数据库仍然不健康也恢复分区
			// 因为下一次消费循环会再次检查并暂停
			if err := c.consumer.Resume(assigned); err != nil {
				return fmt.Errorf("恢复分区失败: %w", err)
			}
			c.log.Info("已恢复消费者分配的分区", logger.Int32("分区数量", int32(len(assigned))))
		}
		return nil
	}

	// 数据库健康，获取并处理消息
	ev := c.consumer.Poll(poll)
	if ev == nil {
		return nil
	}

	switch e := ev.(type) {
	case *kafka.Message:
		// 处理消息
		msg := &mq.Message{
			Topic:     *e.TopicPartition.Topic,
			Partition: int64(e.TopicPartition.Partition),
			Offset:    int64(e.TopicPartition.Offset),
			Key:       e.Key,
			Value:     e.Value,
		}
		if err := c.processMessage(ctx, msg); err != nil {
			c.log.Error("处理消息失败",
				logger.Error(err),
				logger.String("topic", msg.Topic),
				logger.String("partition", fmt.Sprintf("%d", msg.Partition)),
				logger.String("offset", fmt.Sprintf("%d", msg.Offset)),
			)
			return err
		}
		// 提交消息
		if _, err := c.consumer.CommitMessage(e); err != nil {
			return fmt.Errorf("提交消息失败: %w", err)
		}
	case kafka.Error:
		return fmt.Errorf("kafka错误: %w", e)
	}

	return nil
}

func (c *ConnPoolEventConsumer) processMessage(ctx context.Context, msg *mq.Message) error {
	// 反序列化消息
	var event ConnPoolEvent
	if err := json.Unmarshal(msg.Key, &event); err != nil {
		return fmt.Errorf("反序列化消息失败: %w", err)
	}

	c.log.Info("正在处理ConnPoolEvent", logger.String("sql", event.SQL), logger.Any("参数", event.Args))

	// 在数据库上执行SQL
	_, err := c.db.ConnPool.ExecContext(ctx, event.SQL, event.Args...)
	if err != nil {
		return fmt.Errorf("执行事件中的SQL失败: %w", err)
	}

	return nil
}

func (c *ConnPoolEventConsumer) Stop() error {
	return c.consumer.Close()
}
