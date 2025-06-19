package failover

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/confluentinc/confluent-kafka-go/v2/kafka"
)

const (
	FailoverTopic = "fail_over_event"
)

type ConnPoolEvent struct {
	SQL  string        `json:"sql"`
	Args []interface{} `json:"args"`
}

//go:generate mockgen -source=./producer.go -package=evtmocks -destination=../mocks/conn_pool_event_producer.mock.go -typed ConnPoolEventProducer
type ConnPoolEventProducer interface {
	Produce(ctx context.Context, evt ConnPoolEvent) error
}

type Producer struct {
	producer *kafka.Producer
}

func NewProducer(producer *kafka.Producer) *Producer {
	return &Producer{producer: producer}
}

func (p *Producer) Produce(ctx context.Context, evt ConnPoolEvent) error {
	evtStr, err := json.Marshal(evt)
	if err != nil {
		return fmt.Errorf("序列化topic的消息失败: %w", err)
	}
	topic := FailoverTopic

	deliveryChan := make(chan kafka.Event)
	err = p.producer.Produce(&kafka.Message{
		TopicPartition: kafka.TopicPartition{
			Topic:     &topic,
			Partition: kafka.PartitionAny,
		},
		Value: evtStr,
	}, deliveryChan)
	if err != nil {
		return fmt.Errorf("发送消息到kafka失败: %w", err)
	}

	select {
	case <-ctx.Done():
		return ctx.Err()
	case e := <-deliveryChan:
		m, _ := e.(*kafka.Message)
		if m.TopicPartition.Error != nil {
			return fmt.Errorf("发送消息失败: %w", m.TopicPartition.Error)
		}
	}
	return nil
}
