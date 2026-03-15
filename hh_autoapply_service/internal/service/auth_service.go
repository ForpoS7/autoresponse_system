package service

import (
	"context"
	"errors"
	"hh_autoapply_service/internal/jwt"
	"hh_autoapply_service/internal/model"
	"hh_autoapply_service/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrUserAlreadyExists  = errors.New("user with this email already exists")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type AuthService struct {
	userRepo   *repository.UserRepository
	jwtManager *jwt.JWTManager
}

func NewAuthService(userRepo *repository.UserRepository, jwtManager *jwt.JWTManager) *AuthService {
	return &AuthService{
		userRepo:   userRepo,
		jwtManager: jwtManager,
	}
}

// Register регистрирует нового пользователя
func (s *AuthService) Register(ctx context.Context, email, password string) (string, int64, error) {
	// Проверяем, существует ли пользователь
	_, err := s.userRepo.GetByEmail(email)
	if err == nil {
		return "", 0, ErrUserAlreadyExists
	}

	// Хэшируем пароль
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", 0, err
	}

	// Создаем пользователя
	user := &model.User{
		Email:    email,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(user); err != nil {
		return "", 0, err
	}

	// Генерируем JWT токен
	return s.jwtManager.GenerateToken(user.ID, user.Email)
}

// Login аутентифицирует пользователя и возвращает токен
func (s *AuthService) Login(ctx context.Context, email, password string) (string, int64, error) {
	// Получаем пользователя
	user, err := s.userRepo.GetByEmail(email)
	if err != nil {
		return "", 0, ErrInvalidCredentials
	}

	// Проверяем пароль
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", 0, ErrInvalidCredentials
	}

	// Генерируем JWT токен
	return s.jwtManager.GenerateToken(user.ID, user.Email)
}

// ValidateToken проверяет валидность токена
func (s *AuthService) ValidateToken(tokenString string) (*jwt.Claims, error) {
	return s.jwtManager.ValidateToken(tokenString)
}
