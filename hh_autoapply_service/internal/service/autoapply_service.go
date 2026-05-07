package service

import (
	"context"
	"fmt"
	"hh_autoapply_service/internal/cache"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/repository"
	"hh_autoapply_service/pkg/kafka"
	"log"
	"math/rand"
	"time"

	playwrightgo "github.com/playwright-community/playwright-go"
)

// fallbackLetters используются когда generate_service не ответил за 30 секунд.
var fallbackLetters = []string{
	"Имею релевантный опыт по данной позиции. Готов обсудить детали на собеседовании.",
	"Опыт соответствует требованиям вакансии. Буду рад рассмотреть предложение.",
	"Мой background хорошо подходит для этой роли. Готов к диалогу.",
}

type AutoApplyService struct {
	playwrightService  *PlaywrightService
	autoApplyRepo      *repository.AutoApplyRepository
	vacancyRepo        *repository.VacancyRepository
	hhTokenRepo        *repository.HhTokenRepository
	tokenCache         *cache.TokenCache
	parsePublisher     *kafka.ParsePublisher
	vacancyCorrelator  *kafka.VacancyCorrelator
	letterPublisher    *kafka.LetterPublisher
	letterCorrelator   *kafka.LetterCorrelator
}

func NewAutoApplyService(
	playwrightService *PlaywrightService,
	autoApplyRepo *repository.AutoApplyRepository,
	vacancyRepo *repository.VacancyRepository,
	hhTokenRepo *repository.HhTokenRepository,
	tokenCache *cache.TokenCache,
	parsePublisher *kafka.ParsePublisher,
	vacancyCorrelator *kafka.VacancyCorrelator,
	letterPublisher *kafka.LetterPublisher,
	letterCorrelator *kafka.LetterCorrelator,
) *AutoApplyService {
	return &AutoApplyService{
		playwrightService: playwrightService,
		autoApplyRepo:     autoApplyRepo,
		vacancyRepo:       vacancyRepo,
		hhTokenRepo:       hhTokenRepo,
		tokenCache:        tokenCache,
		parsePublisher:    parsePublisher,
		vacancyCorrelator: vacancyCorrelator,
		letterPublisher:   letterPublisher,
		letterCorrelator:  letterCorrelator,
	}
}

func (s *AutoApplyService) CreateAutoApplyRequest(ctx context.Context, userID int64, query string, applyCount int) (*model.AutoApplyRequest, error) {
	req := &model.AutoApplyRequest{
		UserID:       userID,
		Query:        query,
		ApplyCount:   applyCount,
		AppliedCount: 0,
		Status:       "pending",
	}

	if err := s.autoApplyRepo.CreateRequest(req); err != nil {
		return nil, err
	}

	go s.processAutoApply(context.Background(), req)

	return req, nil
}

func (s *AutoApplyService) processAutoApply(ctx context.Context, req *model.AutoApplyRequest) {
	log.Printf("[autoapply] start requestId=%d userId=%d query=%s", req.ID, req.UserID, req.Query)

	req.Status = "processing"
	if err := s.autoApplyRepo.UpdateRequest(req); err != nil {
		log.Printf("[autoapply] status update error: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("[autoapply] panic: %v", r)
			req.Status = "failed"
			s.autoApplyRepo.UpdateRequest(req)
		}
	}()

	// 1. Получаем HH-токен из кеша (не HTTP-вызов к Java)
	hhToken, err := s.getHHToken(ctx, req.UserID)
	if err != nil || len(hhToken) < 100 {
		log.Printf("[autoapply] no HH token for userId=%d: %v", req.UserID, err)
		req.Status = "failed"
		s.autoApplyRepo.UpdateRequest(req)
		return
	}

	// 2. Публикуем запрос на парсинг в Java через Kafka (EDA вместо ожидания вслепую)
	jobID := fmt.Sprintf("autoapply-%d", req.ID)
	if err := s.parsePublisher.Publish(ctx, kafka.ParseRequestMessage{
		JobID:  jobID,
		Query:  req.Query,
		Page:   0,
		UserID: req.UserID,
	}); err != nil {
		log.Printf("[autoapply] parse.requested publish error: %v", err)
		req.Status = "failed"
		s.autoApplyRepo.UpdateRequest(req)
		return
	}
	log.Printf("[autoapply] parse.requested sent: jobId=%s", jobID)

	// 3. Ждём вакансии с нашим jobId (не 5 минут вслепую, а детерминированно)
	vacancies, err := s.vacancyCorrelator.WaitForJob(jobID, 2*time.Minute)
	if err != nil {
		log.Printf("[autoapply] no vacancies for jobId=%s: %v", jobID, err)
		req.Status = "failed"
		s.autoApplyRepo.UpdateRequest(req)
		return
	}
	log.Printf("[autoapply] received %d vacancies for jobId=%s", len(vacancies), jobID)

	// 4. Откликаемся
	for _, vacancy := range vacancies {
		if req.AppliedCount >= req.ApplyCount {
			break
		}

		letter := s.generateLetter(ctx, vacancy, req.Query)

		success, err := s.applyToVacancy(ctx, req, vacancy, hhToken, letter)
		if err != nil {
			log.Printf("[autoapply] vacancy %s error: %v", vacancy.ID, err)
			s.createLog(req.ID, vacancy, letter, "failed", err.Error())
		} else if success {
			req.AppliedCount++
			s.createLog(req.ID, vacancy, letter, "success", "")
			s.autoApplyRepo.UpdateRequest(req)
			log.Printf("[autoapply] applied %d/%d", req.AppliedCount, req.ApplyCount)
		}

		delay := 5 + rand.Intn(5)
		time.Sleep(time.Duration(delay) * time.Second)
	}

	req.Status = "completed"
	s.autoApplyRepo.UpdateRequest(req)
	log.Printf("[autoapply] done requestId=%d applied=%d", req.ID, req.AppliedCount)
}

// getHHToken читает токен из кеша, при miss — из БД.
// HTTP-вызова к Java сервису больше нет.
func (s *AutoApplyService) getHHToken(ctx context.Context, userID int64) (string, error) {
	if token, ok := s.tokenCache.Get(userID); ok {
		log.Printf("[autoapply] HH token from cache for userId=%d", userID)
		return token, nil
	}

	log.Printf("[autoapply] HH token cache miss, loading from DB for userId=%d", userID)
	hhToken, err := s.hhTokenRepo.GetByUserID(userID)
	if err != nil {
		return "", fmt.Errorf("token not found in DB: %w", err)
	}

	s.tokenCache.Set(userID, hhToken.TokenValue)
	return hhToken.TokenValue, nil
}

// generateLetter запрашивает письмо через Kafka и ждёт 30 секунд.
// При таймауте — возвращает fallback шаблон.
func (s *AutoApplyService) generateLetter(ctx context.Context, vacancy model.Vacancy, query string) string {
	corrID := fmt.Sprintf("letter-%s-%d", vacancy.ID, time.Now().UnixNano())

	if err := s.letterPublisher.Publish(ctx, kafka.LetterRequest{
		CorrelationID: corrID,
		Title:         vacancy.Title,
		Company:       vacancy.Employer,
		Requirements:  vacancy.Description,
	}); err != nil {
		log.Printf("[autoapply] vacancy_input publish error: %v", err)
		return fallbackLetter()
	}

	letter, err := s.letterCorrelator.WaitForLetter(corrID, 30*time.Second)
	if err != nil {
		log.Printf("[autoapply] letter timeout for corrId=%s, using fallback", corrID)
		return fallbackLetter()
	}

	return letter
}

func fallbackLetter() string {
	return fallbackLetters[rand.Intn(len(fallbackLetters))]
}

func (s *AutoApplyService) applyToVacancy(ctx context.Context, req *model.AutoApplyRequest, vacancy model.Vacancy, hhToken, letter string) (bool, error) {
	log.Printf("[autoapply] applying to: %s (%s)", vacancy.Title, vacancy.URL)

	browserPage, err := s.playwrightService.GetPageWithToken(ctx, req.UserID, hhToken)
	if err != nil {
		return false, fmt.Errorf("browser page error: %w", err)
	}
	defer browserPage.Close()

	pg := browserPage.Page

	if _, err := pg.Goto(vacancy.URL); err != nil {
		return false, fmt.Errorf("navigation error: %w", err)
	}
	if err := pg.WaitForLoadState(); err != nil {
		return false, fmt.Errorf("page load error: %w", err)
	}

	time.Sleep(2 * time.Second)

	applyButton, err := pg.QuerySelector("[data-qa='vacancy-response-link-top'], .vacancy-actions a:text('Откликнуться')")
	if err != nil || applyButton == nil {
		return false, fmt.Errorf("apply button not found")
	}

	if err := applyButton.Click(); err != nil {
		return false, fmt.Errorf("click error: %w", err)
	}

	time.Sleep(time.Second)

	confirmButton, err := pg.QuerySelector(".magritte-modal-footer button[type='submit'], button:text('Откликнуться')")
	if err == nil && confirmButton != nil {
		if err := confirmButton.Click(); err != nil {
			return false, fmt.Errorf("confirm click error: %w", err)
		}
	}

	if letter != "" {
		textarea, err := pg.WaitForSelector("textarea", playwrightgo.PageWaitForSelectorOptions{
			Timeout: playwrightgo.Float(5000),
		})
		if err == nil && textarea != nil {
			textarea.Fill(letter) //nolint
		}
	}

	time.Sleep(time.Second)
	return true, nil
}

func (s *AutoApplyService) createLog(requestID int64, vacancy model.Vacancy, letter, status, errMsg string) {
	vacancyID := int64(0)
	fmt.Sscanf(vacancy.ID, "%d", &vacancyID)

	entry := &model.AutoApplyLog{
		RequestID:    requestID,
		VacancyID:    vacancyID,
		VacancyURL:   vacancy.URL,
		CoverLetter:  letter,
		Status:       status,
		ErrorMessage: errMsg,
	}
	if err := s.autoApplyRepo.CreateLog(entry); err != nil {
		log.Printf("[autoapply] log error: %v", err)
	}
}

func (s *AutoApplyService) GetAutoApplyRequest(ctx context.Context, requestID int64) (*model.AutoApplyRequest, error) {
	return s.autoApplyRepo.GetRequestByID(requestID)
}
