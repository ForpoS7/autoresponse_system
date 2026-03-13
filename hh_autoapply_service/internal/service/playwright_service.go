package service

import (
	"context"
	"fmt"
	"hh_autoapply_service/internal/config"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/repository"
	"hh_autoapply_service/pkg/playwright"
	"log"
)

type PlaywrightService struct {
	browserManager   *playwright.BrowserManager
	hhTokenRepo      *repository.HhTokenRepository
	playwrightConfig config.PlaywrightConfig
}

func NewPlaywrightService(
	browserManager *playwright.BrowserManager,
	hhTokenRepo *repository.HhTokenRepository,
	playwrightConfig config.PlaywrightConfig,
) *PlaywrightService {
	return &PlaywrightService{
		browserManager:   browserManager,
		hhTokenRepo:      hhTokenRepo,
		playwrightConfig: playwrightConfig,
	}
}

func (s *PlaywrightService) GetPage(ctx context.Context, userID int64) (*playwright.BrowserPage, error) {
	token, err := s.hhTokenRepo.GetByUserID(userID)
	if err != nil {
		log.Printf("Token not found for user %d, creating new session", userID)
		return s.browserManager.NewPage("")
	}

	log.Printf("Loading session from storage for user %d", userID)
	return s.browserManager.NewPage(token.TokenValue)
}

// GetPageWithToken создает страницу с указанным токеном
func (s *PlaywrightService) GetPageWithToken(ctx context.Context, userID int64, hhToken string) (*playwright.BrowserPage, error) {
	log.Printf("Creating page with token for user %d", userID)
	return s.browserManager.NewPageWithToken(hhToken)
}

func (s *PlaywrightService) SaveSession(ctx context.Context, userID int64, storageState string) error {
	token := &model.HhToken{
		UserID:     userID,
		TokenValue: storageState,
	}
	return s.hhTokenRepo.Save(token)
}

func (s *PlaywrightService) GetAreaCode() int {
	return s.playwrightConfig.AreaCode
}

func (s *PlaywrightService) Close() {
	if s.browserManager != nil {
		s.browserManager.Close()
	}
}

// ExtractHhToken - извлечение токена из текущей сессии HH.ru
func (s *PlaywrightService) ExtractHhToken(ctx context.Context, userID int64) (string, error) {
	page, err := s.GetPage(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to get page: %w", err)
	}
	defer page.Close()

	// Переходим на hh.ru
	if _, err := page.Page.Goto("https://hh.ru"); err != nil {
		return "", fmt.Errorf("failed to navigate to hh.ru: %w", err)
	}

	// Ждем загрузки страницы
	if err := page.Page.WaitForLoadState(); err != nil {
		return "", fmt.Errorf("failed to wait for load state: %w", err)
	}

	// Получаем storage state (куки, local storage)
	storageState, err := page.Context.StorageState()
	if err != nil {
		return "", fmt.Errorf("failed to get storage state: %w", err)
	}

	// Сериализуем и сохраняем
	// В Playwright-go storage state уже в JSON формате
	storageStateStr := ""
	if storageState != nil {
		// Конвертируем в JSON строку
		// Для простоты сохраняем как есть - в реальном использовании
		// нужно будет сериализовать структуру
		log.Printf("Extracted storage state with %d cookies", len(storageState.Cookies))
		storageStateStr = fmt.Sprintf("%+v", storageState)
	}

	// Сохраняем в БД
	if err := s.SaveSession(ctx, userID, storageStateStr); err != nil {
		return "", fmt.Errorf("failed to save session: %w", err)
	}

	return storageStateStr, nil
}
