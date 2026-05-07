package kafka

import (
	"context"
	"encoding/json"
	"log"

	"github.com/segmentio/kafka-go"
)

type LetterPublisher struct {
	writer *kafka.Writer
}

func NewLetterPublisher(brokers []string) *LetterPublisher {
	return &LetterPublisher{
		writer: &kafka.Writer{
			Addr:         kafka.TCP(brokers...),
			Topic:        TopicVacancyInput,
			Balancer:     &kafka.LeastBytes{},
			RequiredAcks: kafka.RequireNone, // fire-and-forget, не блокируем откликание
		},
	}
}

func (p *LetterPublisher) Close() error {
	return p.writer.Close()
}

// Publish отправляет запрос генерации письма в generate_service.
func (p *LetterPublisher) Publish(ctx context.Context, req LetterRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	log.Printf("vacancy_input published: correlationId=%s title=%s", req.CorrelationID, req.Title)
	return p.writer.WriteMessages(ctx, kafka.Message{
		Key:   []byte(req.CorrelationID),
		Value: data,
	})
}
