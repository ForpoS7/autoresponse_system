package service

import (
	"context"
	"fmt"
	"hh_autoapply_service/internal/model"
	"log"
	"regexp"
	"strconv"
	"strings"

	playwrightgo "github.com/playwright-community/playwright-go"
)

type ParserService struct {
	playwrightService *PlaywrightService
	vacancyPublisher  *VacancyPublisher
	areaCode          int
}

func NewParserService(
	playwrightService *PlaywrightService,
	vacancyPublisher *VacancyPublisher,
	areaCode int,
) *ParserService {
	return &ParserService{
		playwrightService: playwrightService,
		vacancyPublisher:  vacancyPublisher,
		areaCode:          areaCode,
	}
}

func (s *ParserService) ParseVacancies(ctx context.Context, query string, page int, userID int64) ([]model.Vacancy, error) {
	log.Printf("Parsing vacancies: query='%s', page=%d, userId=%d", query, page, userID)

	browserPage, err := s.playwrightService.GetPage(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get browser page: %w", err)
	}
	defer browserPage.Close()

	pg := browserPage.Page

	// Формируем URL
	url := fmt.Sprintf(
		"https://hh.ru/search/vacancy?text=%s&area=%d&items_on_page=20&page=%d",
		query,
		s.areaCode,
		page,
	)

	log.Printf("URL: %s", url)

	// Переход на страницу
	if _, err := pg.Goto(url); err != nil {
		return nil, fmt.Errorf("failed to navigate: %w", err)
	}

	// Ждем загрузки вакансий
	_, err = pg.WaitForSelector("[data-qa='vacancy-serp__vacancy']", playwrightgo.PageWaitForSelectorOptions{
		Timeout: playwrightgo.Float(10000),
	})
	if err != nil {
		return nil, fmt.Errorf("vacancies not found: %w", err)
	}

	// Получаем все вакансии
	cards, err := pg.QuerySelectorAll("[data-qa='vacancy-serp__vacancy']")
	if err != nil {
		return nil, fmt.Errorf("failed to get vacancy cards: %w", err)
	}

	var vacancies []model.Vacancy

	for _, card := range cards {
		vacancy, err := s.parseVacancyCard(pg, card)
		if err != nil {
			log.Printf("Failed to parse vacancy card: %v", err)
			continue
		}
		if vacancy != nil {
			vacancy.UserID = userID
			vacancies = append(vacancies, *vacancy)
		}
	}

	log.Printf("Found %d vacancies", len(vacancies))

	// Публикуем в Kafka
	if len(vacancies) > 0 {
		if err := s.vacancyPublisher.Publish(ctx, vacancies); err != nil {
			log.Printf("Failed to publish vacancies to Kafka: %v", err)
		}
	}

	return vacancies, nil
}

func (s *ParserService) parseVacancyCard(page playwrightgo.Page, card playwrightgo.ElementHandle) (*model.Vacancy, error) {
	// Заголовок вакансии
	titleEl, err := card.QuerySelector("[data-qa='serp-item__title']")
	if err != nil || titleEl == nil {
		return nil, fmt.Errorf("title element not found")
	}
	defer titleEl.Dispose()

	href, err := titleEl.GetAttribute("href")
	if err != nil {
		return nil, fmt.Errorf("failed to get href: %w", err)
	}

	title, err := titleEl.TextContent()
	if err != nil {
		return nil, fmt.Errorf("failed to get title text: %w", err)
	}
	title = strings.TrimSpace(title)

	// Работодатель
	employerEl, err := card.QuerySelector("[data-qa='vacancy-serp__vacancy-employer']")
	if err != nil {
		employerEl = nil
	}
	employer := "Не указан"
	if employerEl != nil {
		defer employerEl.Dispose()
		employerText, err := employerEl.TextContent()
		if err == nil {
			employer = strings.TrimSpace(employerText)
		}
	}

	// Извлекаем ID вакансии из URL
	vacancyID, err := s.extractVacancyID(href)
	if err != nil {
		log.Printf("Failed to extract vacancy ID: %v", err)
		vacancyID = 0
	}

	// Парсим зарплату и регион из карточки
	salaryFrom, salaryTo, currency := s.parseSalary(card)
	region := s.parseRegion(card)

	// Получаем описание (переходим на страницу вакансии)
	description := ""
	// Пока не парсим описание для экономии времени

	return &model.Vacancy{
		ID:          vacancyID,
		Title:       title,
		Employer:    employer,
		URL:         href,
		Description: description,
		SalaryFrom:  salaryFrom,
		SalaryTo:    salaryTo,
		Currency:    currency,
		Region:      region,
	}, nil
}

func (s *ParserService) extractVacancyID(href string) (int64, error) {
	// URL вида: https://hh.ru/vacancy/12345678?...
	re := regexp.MustCompile(`/vacancy/(\d+)`)
	matches := re.FindStringSubmatch(href)
	if len(matches) < 2 {
		return 0, fmt.Errorf("vacancy ID not found in URL")
	}
	return strconv.ParseInt(matches[1], 10, 64)
}

func (s *ParserService) parseSalary(card playwrightgo.ElementHandle) (*int64, *int64, string) {
	// Ищем элемент с зарплатой
	salaryEl, err := card.QuerySelector("[data-qa='serp-item__compensation']")
	if err != nil || salaryEl == nil {
		return nil, nil, ""
	}
	defer salaryEl.Dispose()

	salaryText, err := salaryEl.TextContent()
	if err != nil {
		return nil, nil, ""
	}
	salaryText = strings.TrimSpace(salaryText)

	// Парсим зарплату из текста вида "от 100 000 до 150 000 ₽"
	var salaryFrom, salaryTo int64
	currency := "RUR"

	// Удаляем пробелы между цифрами
	re := regexp.MustCompile(`(\d+)\s*(\d+)?`)
	matches := re.FindAllString(salaryText, -1)

	for _, match := range matches {
		cleanMatch := strings.ReplaceAll(match, " ", "")
		val, err := strconv.ParseInt(cleanMatch, 10, 64)
		if err != nil {
			continue
		}
		if salaryFrom == 0 {
			salaryFrom = val
		} else {
			salaryTo = val
			break
		}
	}

	if strings.Contains(salaryText, "€") {
		currency = "EUR"
	} else if strings.Contains(salaryText, "$") {
		currency = "USD"
	}

	if salaryFrom == 0 && salaryTo == 0 {
		return nil, nil, currency
	}

	return &salaryFrom, &salaryTo, currency
}

func (s *ParserService) parseRegion(card playwrightgo.ElementHandle) string {
	regionEl, err := card.QuerySelector("[data-qa='serp-item__region']")
	if err != nil || regionEl == nil {
		return ""
	}
	defer regionEl.Dispose()

	region, err := regionEl.TextContent()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(region)
}
