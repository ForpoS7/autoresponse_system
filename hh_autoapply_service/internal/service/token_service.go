package service

import (
	"context"
	"fmt"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/repository"
)

type TokenService struct {
	hhTokenRepo       *repository.HhTokenRepository
	playwrightService *PlaywrightService
}

func NewTokenService(
	hhTokenRepo *repository.HhTokenRepository,
	playwrightService *PlaywrightService,
) *TokenService {
	return &TokenService{
		hhTokenRepo:       hhTokenRepo,
		playwrightService: playwrightService,
	}
}

func (s *TokenService) GetToken(ctx context.Context, userID int64) (*model.HhToken, error) {
	token, err := s.hhTokenRepo.GetByUserID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get token: %w", err)
	}
	return token, nil
}

func (s *TokenService) ExtractToken(ctx context.Context, userID int64) (string, error) {
	// Используем Playwright для извлечения токена из сессии HH.ru
	tokenValue, err := s.playwrightService.ExtractHhToken(ctx, userID)
	if err != nil {
		return "", fmt.Errorf("failed to extract token: %w", err)
	}
	return tokenValue, nil
}

func (s *TokenService) SaveToken(ctx context.Context, userID int64, tokenValue string) error {
	token := &model.HhToken{
		UserID:     userID,
		TokenValue: tokenValue,
	}
	return s.hhTokenRepo.Save(token)
}
