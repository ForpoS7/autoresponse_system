package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// LetterCorrelator слушает vacancy_output и маршрутизирует сгенерированные письма
// к ожидающим горутинам по correlationId.
type LetterCorrelator struct {
	reader  *kafka.Reader
	pending sync.Map // map[correlationId]chan string
}

func NewLetterCorrelator(brokers []string) *LetterCorrelator {
	return &LetterCorrelator{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       TopicVacancyOutput,
			GroupID:     "autoapply-letter-correlator",
			StartOffset: kafka.LastOffset,
			MinBytes:    1,
			MaxBytes:    1e6,
		}),
	}
}

func (c *LetterCorrelator) Close() error {
	return c.reader.Close()
}

// Run запускает фоновую горутину чтения — вызывать один раз в main.
func (c *LetterCorrelator) Run(ctx context.Context) {
	go func() {
		for {
			if ctx.Err() != nil {
				return
			}

			readCtx, cancel := context.WithTimeout(ctx, time.Second)
			msg, err := c.reader.FetchMessage(readCtx)
			cancel()

			if err != nil {
				continue
			}

			var resp LetterResponse
			if err := json.Unmarshal(msg.Value, &resp); err != nil {
				log.Printf("letter_correlator: unmarshal error: %v", err)
				c.reader.CommitMessages(ctx, msg) //nolint
				continue
			}

			if ch, ok := c.pending.Load(resp.CorrelationID); ok {
				ch.(chan string) <- resp.GeneratedText
				log.Printf("letter_correlator: routed letter for correlationId=%s", resp.CorrelationID)
			}

			c.reader.CommitMessages(ctx, msg) //nolint
		}
	}()
}

// WaitForLetter блокирует до получения письма с нужным correlationId или таймаута.
func (c *LetterCorrelator) WaitForLetter(correlationID string, timeout time.Duration) (string, error) {
	ch := make(chan string, 1)
	c.pending.Store(correlationID, ch)
	defer c.pending.Delete(correlationID)

	select {
	case letter := <-ch:
		return letter, nil
	case <-time.After(timeout):
		return "", fmt.Errorf("timeout waiting for letter correlationId=%s", correlationID)
	}
}
