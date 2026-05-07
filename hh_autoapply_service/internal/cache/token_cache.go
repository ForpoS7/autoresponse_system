package cache

import (
	"context"
	"fmt"
	"hh_autoapply_service/pkg/metrics"
	"hh_autoapply_service/pkg/redis"
	"log"
	"sync"
	"time"
)

const tokenTTL = 24 * time.Hour

// TokenCache — потокобезопасный кеш HH storageState по userId.
// Redis — primary (переживает рестарт сервиса).
// In-memory — fallback когда Redis недоступен.
type TokenCache struct {
	mu    sync.RWMutex
	local map[int64]string
	rdb   *redis.Client // nil когда Redis не сконфигурирован
}

func NewTokenCache(rdb *redis.Client) *TokenCache {
	return &TokenCache{
		local: make(map[int64]string),
		rdb:   rdb,
	}
}

func (c *TokenCache) Set(userID int64, tokenValue string) {
	// in-memory always
	c.mu.Lock()
	c.local[userID] = tokenValue
	c.mu.Unlock()

	if c.rdb == nil {
		return
	}
	key := fmt.Sprintf("hh_token:%d", userID)
	if err := c.rdb.Set(context.Background(), key, tokenValue, tokenTTL); err != nil {
		log.Printf("TokenCache: redis SET error (key=%s): %v", key, err)
	}
}

func (c *TokenCache) Get(userID int64) (string, bool) {
	// 1. in-memory fast path
	c.mu.RLock()
	v, ok := c.local[userID]
	c.mu.RUnlock()
	if ok {
		metrics.RedisCacheHits.WithLabelValues("token").Inc()
		return v, true
	}

	// 2. Redis warm-up (e.g. after restart)
	if c.rdb == nil {
		metrics.RedisCacheMisses.WithLabelValues("token").Inc()
		return "", false
	}
	key := fmt.Sprintf("hh_token:%d", userID)
	val, err := c.rdb.Get(context.Background(), key)
	if err != nil {
		metrics.RedisCacheMisses.WithLabelValues("token").Inc()
		return "", false
	}

	metrics.RedisCacheHits.WithLabelValues("token").Inc()
	// populate local
	c.mu.Lock()
	c.local[userID] = val
	c.mu.Unlock()
	return val, true
}
