package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"hh_autoapply_service/internal/model"
	"log"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// VacancyCorrelator слушает vacancies.parsed и маршрутизирует батчи
// к ожидающим горутинам по jobId.
type VacancyCorrelator struct {
	reader  *kafka.Reader
	pending sync.Map // map[jobId]chan []model.Vacancy
}

func NewVacancyCorrelator(brokers []string) *VacancyCorrelator {
	return &VacancyCorrelator{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     brokers,
			Topic:       TopicVacanciesParsed,
			GroupID:     "autoapply-correlator",
			StartOffset: kafka.LastOffset,
			MinBytes:    1,
			MaxBytes:    10e6,
		}),
	}
}

func (c *VacancyCorrelator) Close() error {
	return c.reader.Close()
}

// Run запускает фоновую горутину чтения — вызывать один раз в main.
func (c *VacancyCorrelator) Run(ctx context.Context) {
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

			var batch VacanciesBatch
			if err := json.Unmarshal(msg.Value, &batch); err != nil {
				log.Printf("vacancy_correlator: unmarshal error: %v", err)
				c.reader.CommitMessages(ctx, msg) //nolint
				continue
			}

			if ch, ok := c.pending.Load(batch.JobID); ok {
				vacancies := toModelVacancies(batch.Vacancies)
				ch.(chan []model.Vacancy) <- vacancies
				log.Printf("vacancy_correlator: routed %d vacancies to jobId=%s", len(vacancies), batch.JobID)
			}

			c.reader.CommitMessages(ctx, msg) //nolint
		}
	}()
}

// WaitForJob блокирует до получения вакансий с нужным jobId или таймаута.
func (c *VacancyCorrelator) WaitForJob(jobID string, timeout time.Duration) ([]model.Vacancy, error) {
	ch := make(chan []model.Vacancy, 1)
	c.pending.Store(jobID, ch)
	defer c.pending.Delete(jobID)

	select {
	case vacancies := <-ch:
		return vacancies, nil
	case <-time.After(timeout):
		return nil, fmt.Errorf("timeout waiting for vacancies with jobId=%s", jobID)
	}
}

func toModelVacancies(payloads []VacancyPayload) []model.Vacancy {
	result := make([]model.Vacancy, 0, len(payloads))
	for _, p := range payloads {
		result = append(result, model.Vacancy{
			ID:          fmt.Sprintf("%d", p.ID),
			Title:       p.Title,
			Employer:    p.Employer,
			URL:         p.URL,
			Description: p.Description,
			SalaryFrom:  p.SalaryFrom,
			SalaryTo:    p.SalaryTo,
			Currency:    p.Currency,
			Region:      p.Region,
			UserID:      p.UserID,
		})
	}
	return result
}
