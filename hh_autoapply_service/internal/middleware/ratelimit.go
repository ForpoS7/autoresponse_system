package middleware

import (
	"hh_autoapply_service/pkg/ratelimit"
	"net/http"
)

// RateLimitMiddleware - middleware для ограничения запросов
func RateLimitMiddleware(limiter *ratelimit.RateLimiter, getUserID func(r *http.Request) int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID := getUserID(r)
			if userID == 0 {
				// Если пользователь не аутентифицирован, используем default limiter
				// или пропускаем запрос
				next.ServeHTTP(w, r)
				return
			}

			if !limiter.Allow(userID) {
				http.Error(w, `{"error": "rate limit exceeded"}`, http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
