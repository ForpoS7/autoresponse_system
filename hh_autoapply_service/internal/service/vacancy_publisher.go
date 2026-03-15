package service

import (
	"context"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/pkg/kafka"
)

type VacancyPublisher struct {
	producer *kafka.Producer
	topic    string
}

func NewVacancyPublisher(producer *kafka.Producer, topic string) *VacancyPublisher {
	return &VacancyPublisher{
		producer: producer,
		topic:    topic,
	}
}

func (p *VacancyPublisher) Publish(ctx context.Context, vacancies []model.Vacancy) error {
	messages := make([]interface{}, len(vacancies))
	for i, v := range vacancies {
		messages[i] = v
	}
	return p.producer.Publish(ctx, p.topic, messages)
}
