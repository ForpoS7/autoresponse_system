package kafka

import (
	"context"
	"encoding/json"

	"github.com/segmentio/kafka-go"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(brokers []string) *Producer {
	return &Producer{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

func (p *Producer) Publish(ctx context.Context, topic string, messages []interface{}) error {
	kafkaMessages := make([]kafka.Message, len(messages))

	for i, msg := range messages {
		data, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		kafkaMessages[i] = kafka.Message{
			Topic: topic,
			Value: data,
		}
	}

	return p.writer.WriteMessages(ctx, kafkaMessages...)
}
