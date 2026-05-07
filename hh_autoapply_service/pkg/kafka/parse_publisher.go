package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type ParsePublisher struct {
	writer *kafka.Writer
}

func NewParsePublisher(brokers []string) *ParsePublisher {
	return &ParsePublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        TopicParseRequested,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireAll,
		},
	}
}

func (p *ParsePublisher) Close() error {
	return p.writer.Close()
}

// Publish отправляет запрос на парсинг в Java-сервис.
// jobId используется для корреляции — Go ждёт вакансии с тем же jobId.
func (p *ParsePublisher) Publish(ctx context.Context, msg ParseRequestMessage) error {
	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	log.Printf("parse.requested published: jobId=%s query=%s userId=%d", msg.JobID, msg.Query, msg.UserID)
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(msg.JobID),
		Value: data,
	})
}
