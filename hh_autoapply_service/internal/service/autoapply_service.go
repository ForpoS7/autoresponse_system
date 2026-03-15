package service

import (
	"context"
	"fmt"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/repository"
	"hh_autoapply_service/pkg/ai"
	"hh_autoapply_service/pkg/httpclient"
	"hh_autoapply_service/pkg/kafka"
	"log"
	"math/rand"
	"time"

	playwrightgo "github.com/playwright-community/playwright-go"
)

type AutoApplyService struct {
	parserService      *ParserService
	playwrightService  *PlaywrightService
	coverLetterService *ai.MockCoverLetterService
	autoApplyRepo      *repository.AutoApplyRepository
	vacancyRepo        *repository.VacancyRepository
	hhTokenRepo        *repository.HhTokenRepository
	kafkaConsumer      *kafka.Consumer
	hhClient           *httpclient.HHAggregateClient
	javaServiceToken   string
}

func NewAutoApplyService(
	parserService *ParserService,
	playwrightService *PlaywrightService,
	coverLetterService *ai.MockCoverLetterService,
	autoApplyRepo *repository.AutoApplyRepository,
	vacancyRepo *repository.VacancyRepository,
	hhTokenRepo *repository.HhTokenRepository,
	kafkaConsumer *kafka.Consumer,
	hhClient *httpclient.HHAggregateClient,
	javaServiceToken string,
) *AutoApplyService {
	return &AutoApplyService{
		parserService:      parserService,
		playwrightService:  playwrightService,
		coverLetterService: coverLetterService,
		autoApplyRepo:      autoApplyRepo,
		vacancyRepo:        vacancyRepo,
		hhTokenRepo:        hhTokenRepo,
		kafkaConsumer:      kafkaConsumer,
		hhClient:           hhClient,
		javaServiceToken:   javaServiceToken,
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

	// Запускаем процесс автоотклика в горутине с фоновым контекстом
	go s.processAutoApply(context.Background(), req)

	return req, nil
}

func (s *AutoApplyService) processAutoApply(ctx context.Context, req *model.AutoApplyRequest) {
	log.Printf("Starting auto-apply process for request %d, user %d", req.ID, req.UserID)

	req.Status = "processing"
	if err := s.autoApplyRepo.UpdateRequest(req); err != nil {
		log.Printf("Failed to update request status: %v", err)
	}

	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic in auto-apply process: %v", r)
			req.Status = "failed"
			s.autoApplyRepo.UpdateRequest(req)
		}
	}()

	// Получаем токен HH.ru из Java сервиса (storageState JSON)
	hhToken, err := s.getHHTokenFromJava(ctx, req.UserID)
	if err != nil {
		log.Printf("Failed to get HH token from Java service for user %d: %v", req.UserID, err)
		req.Status = "failed"
		s.autoApplyRepo.UpdateRequest(req)
		return
	}

	// Проверяем что токен не пустой
	if hhToken == "" || len(hhToken) < 100 {
		log.Printf("HH token is empty or too short (%d chars). Please run POST /api/hh-token first", len(hhToken))
		req.Status = "failed"
		s.autoApplyRepo.UpdateRequest(req)
		return
	}

	log.Printf("Got HH token (storageState) from Java service for user %d, length: %d", req.UserID, len(hhToken))

	// Сохраняем токен локально
	if err := s.hhTokenRepo.Save(&model.HhToken{
		UserID:     req.UserID,
		TokenValue: hhToken,
	}); err != nil {
		log.Printf("Failed to save HH token locally: %v", err)
	}

	// Пробуем получить вакансии из Kafka (5 секунд таймаут)
	vacancies, err := s.kafkaConsumer.ConsumeVacanciesBatch(ctx, 5*time.Second)
	if err != nil {
		log.Printf("Failed to get vacancies from Kafka: %v", err)
	}

	// Если Kafka пуста, пробуем получить из Java сервиса
	if len(vacancies) == 0 {
		log.Printf("No vacancies in Kafka, trying Java service")
		vacancies, err = s.getVacanciesFromJava(ctx, req)
		if err != nil {
			log.Printf("Failed to get vacancies from Java service: %v", err)
			req.Status = "failed"
			s.autoApplyRepo.UpdateRequest(req)
			return
		}
	}

	if len(vacancies) == 0 {
		log.Printf("No vacancies found")
		req.Status = "completed"
		s.autoApplyRepo.UpdateRequest(req)
		return
	}

	log.Printf("Found %d vacancies to apply", len(vacancies))

	for _, vacancy := range vacancies {
		if req.AppliedCount >= req.ApplyCount {
			break
		}

		success, err := s.applyToVacancy(ctx, req, vacancy, hhToken)
		if err != nil {
			log.Printf("Failed to apply to vacancy %d: %v", vacancy.ID, err)
			s.createLog(req.ID, vacancy.ID, vacancy.URL, "", "failed", err.Error())
		} else if success {
			req.AppliedCount++
			s.createLog(req.ID, vacancy.ID, vacancy.URL, "", "success", "")
		}

		// Добавляем задержку между откликами (5-10 секунд + случайная вариация)
		// Это нужно чтобы избежать блокировки со стороны HH.ru
		delaySeconds := 5 + rand.Intn(5) // от 5 до 9 секунд
		log.Printf("Waiting %d seconds before next application...", delaySeconds)
		time.Sleep(time.Duration(delaySeconds) * time.Second)
	}

	req.Status = "completed"
	s.autoApplyRepo.UpdateRequest(req)
	log.Printf("Auto-apply process completed for request %d. Applied to %d vacancies", req.ID, req.AppliedCount)
}

func (s *AutoApplyService) getHHTokenFromJava(ctx context.Context, userID int64) (string, error) {
	log.Printf("Getting HH token from Java service for user %d", userID)

	tokenResp, err := s.hhClient.GetHHToken(ctx, userID, s.javaServiceToken)
	if err != nil {
		return "", fmt.Errorf("failed to get token from Java service: %w", err)
	}

	log.Printf("Got HH token from Java service for user %d", userID)
	return tokenResp.TokenValue, nil
}

func (s *AutoApplyService) getVacanciesFromJava(ctx context.Context, req *model.AutoApplyRequest) ([]model.Vacancy, error) {
	log.Printf("Getting vacancies from Java service for user %d, query: %s", req.UserID, req.Query)

	page := 0
	var allVacancies []model.Vacancy

	for len(allVacancies) < req.ApplyCount {
		vacancies, err := s.hhClient.GetVacancies(ctx, req.UserID, s.javaServiceToken, req.Query, page)
		if err != nil {
			log.Printf("Failed to get vacancies from Java service: %v", err)
			break
		}

		if len(vacancies) == 0 {
			break
		}

		for _, v := range vacancies {
			allVacancies = append(allVacancies, model.Vacancy{
				ID:          v.ID,
				Title:       v.Title,
				Employer:    v.Employer,
				URL:         v.URL,
				Description: v.Description,
				SalaryFrom:  v.SalaryFrom,
				SalaryTo:    v.SalaryTo,
				Currency:    v.Currency,
				Region:      v.Region,
				UserID:      v.UserID,
			})
		}

		page++
		time.Sleep(500 * time.Millisecond)
	}

	return allVacancies, nil
}

func (s *AutoApplyService) applyToVacancy(ctx context.Context, req *model.AutoApplyRequest, vacancy model.Vacancy, hhToken string) (bool, error) {
	log.Printf("Applying to vacancy: %s, URL: %s", vacancy.Title, vacancy.URL)
	log.Printf("HH token length: %d characters", len(hhToken))

	// Получаем страницу браузера с восстановленной сессией HH.ru
	browserPage, err := s.playwrightService.GetPageWithToken(ctx, req.UserID, hhToken)
	if err != nil {
		return false, fmt.Errorf("failed to get browser page: %w", err)
	}
	defer browserPage.Close()

	pg := browserPage.Page

	// Переходим на страницу вакансии
	if _, err := pg.Goto(vacancy.URL); err != nil {
		return false, fmt.Errorf("failed to navigate to vacancy: %w", err)
	}

	if err := pg.WaitForLoadState(); err != nil {
		return false, fmt.Errorf("failed to wait for page load: %w", err)
	}

	// Даем странице время на загрузку
	time.Sleep(2 * time.Second)

	// Ищем кнопку отклика. Пробуем несколько вариантов селекторов
	var applyButton interface{}
	var err error

	// Сначала ищем в блоке действий вакансии (основная кнопка)
	applyButton, err = pg.QuerySelector(".vacancy-action button, .vacancy-actions button, [data-qa='vacancy-action'] button")
	if err != nil || applyButton == nil {
		// Если не нашли, ищем кнопку по тексту "Откликнуться" в верхней части страницы
		// Ищем в заголовке или в первом блоке с кнопками
		applyButton, err = pg.QuerySelector("h1 ~ * button:text('Откликнуться'), header button:text('Откликнуться')")
	}

	if err != nil {
		return false, fmt.Errorf("failed to find apply button: %w", err)
	}

	if applyButton == nil {
		// Кнопка не найдена - возможно мы не авторизованы или это не та страница
		log.Printf("Apply button not found - checking authorization...")
		loginBtn, _ := pg.QuerySelector("[href='/login']")
		if loginBtn != nil {
			log.Printf("Found login button - not authorized")
		}
		return false, fmt.Errorf("apply button not found - may not be authorized or wrong page")
	}

	log.Printf("Successfully found apply button - proceeding with application")

	coverLetter, err := s.coverLetterService.GenerateCoverLetter(ctx, vacancy, req.Query)
	if err != nil {
		log.Printf("Failed to generate cover letter: %v", err)
		coverLetter = ""
	}

	if err := applyButton.Click(); err != nil {
		return false, fmt.Errorf("failed to click apply button: %w", err)
	}

	log.Printf("Clicked apply button on vacancy page, waiting for modal...")

	// Даем время на появление модального окна (если оно есть)
	time.Sleep(1 * time.Second)

	// Пробуем найти кнопку подтверждения в модальном окне (оно появляется не всегда)
	confirmButton, err := pg.QuerySelector(".magritte-modal-footer button[type='submit'], button:text('Откликнуться'), button:text('Confirm')")
	if err == nil && confirmButton != nil {
		log.Printf("Found confirm button in modal, clicking...")
		if err := confirmButton.Click(); err != nil {
			return false, fmt.Errorf("failed to click confirm button: %w", err)
		}
		log.Printf("Successfully clicked confirm button in modal")
	} else {
		log.Printf("No modal window found - application submitted directly")
	}

	// Если есть сопроводительное письмо, заполняем его
	if coverLetter != "" {
		textarea, err := pg.WaitForSelector("textarea", playwrightgo.PageWaitForSelectorOptions{
			Timeout: playwrightgo.Float(5000),
		})
		if err == nil && textarea != nil {
			if err := textarea.Fill(coverLetter); err != nil {
				log.Printf("Failed to fill cover letter: %v", err)
			}
		}
	}

	// Даем время на обработку отклика
	time.Sleep(1 * time.Second)

	return true, nil
}

func (s *AutoApplyService) createLog(requestID, vacancyID int64, vacancyURL, coverLetter, status, errorMessage string) {
	logEntry := &model.AutoApplyLog{
		RequestID:    requestID,
		VacancyID:    vacancyID,
		VacancyURL:   vacancyURL,
		CoverLetter:  coverLetter,
		Status:       status,
		ErrorMessage: errorMessage,
	}
	if err := s.autoApplyRepo.CreateLog(logEntry); err != nil {
		log.Printf("Failed to create log entry: %v", err)
	}
}

func (s *AutoApplyService) GetAutoApplyRequest(ctx context.Context, requestID int64) (*model.AutoApplyRequest, error) {
	return s.autoApplyRepo.GetRequestByID(requestID)
}
