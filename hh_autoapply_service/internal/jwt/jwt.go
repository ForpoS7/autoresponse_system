package jwt

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTManager struct {
	secretKey  string
	expiration int64 // в миллисекундах
}

type Claims struct {
	UserID int64  `json:"user_id"`
	Email  string `json:"email"`
	jwt.RegisteredClaims
}

func NewJWTManager(secretKey string, expirationMilliseconds int64) *JWTManager {
	return &JWTManager{
		secretKey:  secretKey,
		expiration: expirationMilliseconds,
	}
}

// GenerateToken генерирует JWT токен для пользователя
func (m *JWTManager) GenerateToken(userID int64, email string) (string, int64, error) {
	expirationTime := time.Now().Add(time.Duration(m.expiration) * time.Millisecond).Unix()

	claims := &Claims{
		UserID: userID,
		Email:  email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Unix(expirationTime, 0)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(m.secretKey))
	if err != nil {
		return "", 0, err
	}

	return tokenString, expirationTime * 1000, nil
}

// ValidateToken проверяет валидность токена и возвращает claims
func (m *JWTManager) ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(m.secretKey), nil
	})

	if err != nil {
		return nil, err
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// GetExpirationMilliseconds возвращает время истечения токена в миллисекундах
func (m *JWTManager) GetExpirationMilliseconds() int64 {
	return m.expiration
}
