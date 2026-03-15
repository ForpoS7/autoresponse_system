package ratelimit

import (
	"sync"

	"golang.org/x/time/rate"
)

// RateLimiter - ограничитель запросов для каждого пользователя
type RateLimiter struct {
	mu       sync.Mutex
	limiters map[int64]*rate.Limiter
	rps      float64
	burst    int
}

// NewRateLimiter создает новый rate limiter
// requestsPerMinute - количество запросов в минуту
// burst - максимальное количество запросов в пакете
func NewRateLimiter(requestsPerMinute int, burst int) *RateLimiter {
	return &RateLimiter{
		limiters: make(map[int64]*rate.Limiter),
		rps:      float64(requestsPerMinute) / 60.0,
		burst:    burst,
	}
}

// getLimiter получает или создает limiter для пользователя
func (rl *RateLimiter) getLimiter(userID int64) *rate.Limiter {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	limiter, exists := rl.limiters[userID]
	if !exists {
		limiter = rate.NewLimiter(rate.Limit(rl.rps), rl.burst)
		rl.limiters[userID] = limiter
	}

	return limiter
}

// Allow проверяет, разрешен ли запрос для пользователя
func (rl *RateLimiter) Allow(userID int64) bool {
	limiter := rl.getLimiter(userID)
	return limiter.Allow()
}

// Wait блокирует до тех пор, пока запрос не будет разрешен
func (rl *RateLimiter) Wait(userID int64) error {
	limiter := rl.getLimiter(userID)
	return limiter.Wait(nil)
}

// Remove удаляет limiter для пользователя (освобождает память)
func (rl *RateLimiter) Remove(userID int64) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.limiters, userID)
}
