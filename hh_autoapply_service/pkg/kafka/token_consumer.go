package kafka

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

// TokenCache — интерфейс, чтобы не было циклической зависимости с пакетом cache.
type TokenCache interface {
	Set(userID int64, tokenValue string)
}

// TokenConsumer слушает token.updated и обновляет кеш HH-токенов.
// Убирает необходимость HTTP-вызова Go → Java при каждом автооткликe.
type TokenConsumer struct {
	reader *kafka.Reader
	cache  TokenCache
}

func NewTokenConsumer(brokers []string, cache TokenCache) *TokenConsumer {
	return &TokenConsumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       TopicTokenUpdated,
			GroupID:     "autoapply-token-cache",
			StartOffset: kafka.LastOffset,
			MinBytes:    1,
			MaxBytes:    10e6,
		}),
		cache: cache,
	}
}

func (c *TokenConsumer) Close() error {
	return c.reader.Close()
}

// Run запускает фоновую горутину — вызывать один раз в main.
func (c *TokenConsumer) Run(ctx context.Context) {
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

			var event TokenUpdatedMessage
			if err := json.Unmarshal(msg.Value, &event); err != nil {
				log.Printf("token_consumer: unmarshal error: %v", err)
				c.reader.CommitMessages(ctx, msg) //nolint
				continue
			}

			c.cache.Set(event.UserID, event.TokenValue)
			log.Printf("token_consumer: cached token for userId=%d", event.UserID)

			c.reader.CommitMessages(ctx, msg) //nolint
		}
	}()
}
