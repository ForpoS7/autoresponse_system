package service

import (
	"context"
	"errors"

	"auth_service/internal/jwt"
	"auth_service/internal/model"
	"auth_service/internal/repository"

	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	users *repository.UserRepository
	jwt   *jwt.Manager
}

func NewAuthService(users *repository.UserRepository, jwt *jwt.Manager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

func (s *AuthService) Register(ctx context.Context, email, password string) (*model.AuthResponse, error) {
	exists, err := s.users.ExistsByEmail(email)
	if err != nil {
		return nil, err
	}
	if exists {
		return nil, errors.New("email already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user, err := s.users.Create(email, string(hash))
	if err != nil {
		return nil, err
	}

	token, expiresAt, err := s.jwt.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, Email: user.Email, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*model.AuthResponse, error) {
	user, err := s.users.GetByEmail(email)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	token, expiresAt, err := s.jwt.Generate(user.ID, user.Email)
	if err != nil {
		return nil, err
	}

	return &model.AuthResponse{Token: token, Email: user.Email, ExpiresAt: expiresAt}, nil
}

func (s *AuthService) Validate(tokenString string) (*model.ValidateResponse, error) {
	claims, err := s.jwt.Validate(tokenString)
	if err != nil {
		return &model.ValidateResponse{Valid: false}, nil
	}
	return &model.ValidateResponse{
		Valid:  true,
		UserID: claims.UserID,
		Email:  claims.Email,
	}, nil
}
