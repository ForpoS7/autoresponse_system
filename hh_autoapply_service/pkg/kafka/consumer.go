package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"hh_autoapply_service/internal/model"
	"log"
	"time"

	"github.com/segmentio/kafka-go"
)

type Consumer struct {
	reader *kafka.Reader
}

type KafkaVacancyMessage struct {
	ID          int64  `json:"id"`
	Title       string `json:"title"`
	Employer    string `json:"employer"`
	URL         string `json:"url"`
	Description string `json:"description,omitempty"`
	SalaryFrom  *int64 `json:"salaryFrom,omitempty"`
	SalaryTo    *int64 `json:"salaryTo,omitempty"`
	Currency    string `json:"currency,omitempty"`
	Region      string `json:"region,omitempty"`
	UserID      int64  `json:"userId"`
}

func NewConsumer(brokers []string, topic, groupID string) *Consumer {
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       topic,
			GroupID:     groupID,
			StartOffset: kafka.LastOffset,
			MinBytes:    10e3,
			MaxBytes:    10e6,
		}),
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

// ConsumeVacanciesBatch читает вакансии пачками с таймаутом
// Java сервис отправляет массив вакансий одним сообщением
func (c *Consumer) ConsumeVacanciesBatch(ctx context.Context, timeout time.Duration) ([]model.Vacancy, error) {
	var allVacancies []model.Vacancy
	deadline := time.Now().Add(timeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return allVacancies, ctx.Err()
		default:
			// Устанавливаем таймаут на чтение
			readCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			msg, err := c.reader.FetchMessage(readCtx)
			cancel()

			if err != nil {
				// Таймаут - сообщений нет
				if len(allVacancies) > 0 {
					return allVacancies, nil
				}
				continue
			}

			// Java сервис отправляет массив вакансий
			var kafkaMessages []KafkaVacancyMessage
			if err := json.Unmarshal(msg.Value, &kafkaMessages); err != nil {
				log.Printf("Failed to unmarshal vacancy array: %v", err)
				continue
			}

			log.Printf("Received batch of %d vacancies from Kafka", len(kafkaMessages))

			for _, kafkaMsg := range kafkaMessages {
				vacancy := model.Vacancy{
					ID:          kafkaMsg.ID,
					Title:       kafkaMsg.Title,
					Employer:    kafkaMsg.Employer,
					URL:         kafkaMsg.URL,
					Description: kafkaMsg.Description,
					SalaryFrom:  kafkaMsg.SalaryFrom,
					SalaryTo:    kafkaMsg.SalaryTo,
					Currency:    kafkaMsg.Currency,
					Region:      kafkaMsg.Region,
					UserID:      kafkaMsg.UserID,
				}
				allVacancies = append(allVacancies, vacancy)
			}

			// Подтверждаем сообщение
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Failed to commit message: %v", err)
			}
		}
	}

	return allVacancies, nil
}

// ConsumeVacanciesBatchAndWait читает вакансии из Kafka и ждет пока все спарсятся
// Ждем сообщения до тех пор пока не будет паузы в idleTimeout или не истечет maxWait
func (c *Consumer) ConsumeVacanciesBatchAndWait(ctx context.Context, maxWait time.Duration, idleTimeout time.Duration) ([]model.Vacancy, error) {
	var allVacancies []model.Vacancy
	deadline := time.Now().Add(maxWait)
	lastMessageTime := time.Now()

	log.Printf("Waiting for vacancies from Kafka (max wait: %v, idle timeout: %v)", maxWait, idleTimeout)

	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return allVacancies, ctx.Err()
		default:
			// Проверяем не было ли паузы в поступлениях сообщений
			if time.Since(lastMessageTime) > idleTimeout && len(allVacancies) > 0 {
				log.Printf("No new vacancies for %v, assuming parsing completed", idleTimeout)
				return allVacancies, nil
			}

			// Устанавливаем таймаут на чтение
			readCtx, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
			msg, err := c.reader.FetchMessage(readCtx)
			cancel()

			if err != nil {
				// Таймаут - сообщений нет, продолжаем ждать
				continue
			}

			// Обновляем время последнего сообщения
			lastMessageTime = time.Now()

			// Java сервис отправляет массив вакансий
			var kafkaMessages []KafkaVacancyMessage
			if err := json.Unmarshal(msg.Value, &kafkaMessages); err != nil {
				log.Printf("Failed to unmarshal vacancy array: %v", err)
				continue
			}

			log.Printf("Received batch of %d vacancies from Kafka (total: %d)", len(kafkaMessages), len(allVacancies)+len(kafkaMessages))

			for _, kafkaMsg := range kafkaMessages {
				vacancy := model.Vacancy{
					ID:          kafkaMsg.ID,
					Title:       kafkaMsg.Title,
					Employer:    kafkaMsg.Employer,
					URL:         kafkaMsg.URL,
					Description: kafkaMsg.Description,
					SalaryFrom:  kafkaMsg.SalaryFrom,
					SalaryTo:    kafkaMsg.SalaryTo,
					Currency:    kafkaMsg.Currency,
					Region:      kafkaMsg.Region,
					UserID:      kafkaMsg.UserID,
				}
				allVacancies = append(allVacancies, vacancy)
			}

			fmt.Println(allVacancies)

			// Подтверждаем сообщение
			if err := c.reader.CommitMessages(ctx, msg); err != nil {
				log.Printf("Failed to commit message: %v", err)
			}
		}
	}

	log.Printf("Max wait time reached, returning %d vacancies", len(allVacancies))
	return allVacancies, nil
}
