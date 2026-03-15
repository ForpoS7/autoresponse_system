package ai

import (
	"context"
	"fmt"
	"hh_autoapply_service/internal/model"
	"log"
	"math/rand"
	"time"
)

// CoverLetterService - интерфейс для сервиса генерации сопроводительных писем
type CoverLetterService interface {
	GenerateCoverLetter(ctx context.Context, vacancy model.Vacancy, userQuery string) (string, error)
	GenerateBulkCoverLetters(ctx context.Context, vacancies []model.Vacancy, userQuery string) (map[int64]string, error)
}

// MockCoverLetterService - мок сервиса для генерации сопроводительных писем
// В будущем будет заменен на реальный AI сервис
type MockCoverLetterService struct {
	templates []string
}

func NewMockCoverLetterService() *MockCoverLetterService {
	return &MockCoverLetterService{
		templates: []string{
			"Здравствуйте!\n\nМеня заинтересовала ваша вакансия \"%s\". Я имею релевантный опыт работы и готов внести значительный вклад в развитие вашей компании.\n\nБуду рад возможности обсудить детали на собеседовании.\n\nС уважением,\nКандидат",
			"Добрый день!\n\nЗаинтересовала позиция \"%s\" в вашей компании. Мой профессиональный опыт соответствует требованиям вакансии. Готов применить свои навыки для решения задач вашей команды.\n\nЖду вашего ответа!\n\nС наилучшими пожеланиями,\nКандидат",
			"Приветствую!\n\nВакансия \"%s\" меня очень заинтересовала. Имею успешный опыт работы в аналогичной сфере и уверен, что смогу быть полезен вашей компании.\n\nГотов обсудить детали на встрече.\n\nС уважением,\nКандидат",
			"Здравствуйте, уважаемые коллеги!\n\nМеня заинтересовала позиция \"%s\". Мой опыт и навыки соответствуют требованиям вакансии. Готов внести свой вклад в развитие вашей компании.\n\nБуду рад возможности личного знакомства.\n\nС уважением,\nКандидат",
			"Добрый день!\n\nВакансия \"%s\" показалась мне очень интересной. Имею релевантный опыт работы и готов к новым вызовам в вашей компании.\n\nЖду возможности обсудить сотрудничество!\n\nС наилучшими пожеланиями,\nКандидат",
		},
	}
}

// GenerateCoverLetter генерирует сопроводительное письмо для вакансии
func (s *MockCoverLetterService) GenerateCoverLetter(ctx context.Context, vacancy model.Vacancy, userQuery string) (string, error) {
	log.Printf("Generating cover letter for vacancy: %s", vacancy.Title)

	// Имитация задержки генерации (как будто обращаемся к AI)
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case <-time.After(time.Duration(100+rand.Intn(200)) * time.Millisecond):
	}

	// Выбираем случайный шаблон
	template := s.templates[rand.Intn(len(s.templates))]

	// Формируем письмо с названием вакансии
	coverLetter := fmt.Sprintf(template, vacancy.Title)

	return coverLetter, nil
}

// GenerateBulkCoverLetters генерирует сопроводительные письма для нескольких вакансий
func (s *MockCoverLetterService) GenerateBulkCoverLetters(ctx context.Context, vacancies []model.Vacancy, userQuery string) (map[int64]string, error) {
	result := make(map[int64]string)

	for _, vacancy := range vacancies {
		letter, err := s.GenerateCoverLetter(ctx, vacancy, userQuery)
		if err != nil {
			log.Printf("Failed to generate cover letter for vacancy %d: %v", vacancy.ID, err)
			continue
		}
		result[vacancy.ID] = letter
	}

	return result, nil
}
