package middleware

import (
	"context"
	"hh_autoapply_service/internal/jwt"
	"net/http"
	"strings"
)

type contextKey string

const UserIDKey contextKey = "userID"

// JWTMiddleware - middleware для проверки JWT токена
func JWTMiddleware(jwtManager *jwt.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				http.Error(w, `{"error": "authorization header required"}`, http.StatusUnauthorized)
				return
			}

			// Ожидаем формат: "Bearer <token>"
			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || parts[0] != "Bearer" {
				http.Error(w, `{"error": "invalid authorization header format"}`, http.StatusUnauthorized)
				return
			}

			tokenString := parts[1]

			// Проверяем токен
			claims, err := jwtManager.ValidateToken(tokenString)
			if err != nil {
				http.Error(w, `{"error": "invalid or expired token"}`, http.StatusUnauthorized)
				return
			}

			// Добавляем userID в контекст
			ctx := context.WithValue(r.Context(), UserIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext извлекает userID из контекста запроса
func GetUserIDFromContext(r *http.Request) int64 {
	userID, ok := r.Context().Value(UserIDKey).(int64)
	if !ok {
		return 0
	}
	return userID
}
