package middleware

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

type memoryLimiterEntry struct {
	count   int
	resetAt time.Time
}

type memoryRateLimiter struct {
	mu          sync.Mutex
	store       map[string]*memoryLimiterEntry
	lastCleanup time.Time
}

var (
	globalMemoryLimiter = &memoryRateLimiter{
		store: make(map[string]*memoryLimiterEntry),
	}
)

func (m *memoryRateLimiter) Allow(key string, limit int, window time.Duration) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Limpeza oportunística a cada 1 minuto para evitar acúmulo de chaves na memória
	if now.Sub(m.lastCleanup) > time.Minute {
		for k, v := range m.store {
			if now.After(v.resetAt) {
				delete(m.store, k)
			}
		}
		m.lastCleanup = now
	}

	entry, ok := m.store[key]
	if !ok || now.After(entry.resetAt) {
		m.store[key] = &memoryLimiterEntry{
			count:   1,
			resetAt: now.Add(window),
		}
		return true
	}

	entry.count++
	return entry.count <= limit
}

func checkMemoryRateLimit(key string, limit int, window time.Duration) bool {
	return globalMemoryLimiter.Allow(key, limit, window)
}

// RateLimit limita a taxa de requisições por IP usando o Redis, com fallback em memória para rotas sensíveis.
func RateLimit(limit int, window time.Duration) gin.HandlerFunc {
	return RateLimitStrict(limit, window, false)
}

// RateLimitStrict permite forçar o uso do rate limiter em memória caso o Redis esteja fora.
func RateLimitStrict(limit int, window time.Duration, strictMemoryFallback bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		client := cache.GetRedisClient()
		ip := c.ClientIP()
		key := fmt.Sprintf("rl:%s:%s", c.FullPath(), ip)

		if client == nil {
			if strictMemoryFallback || c.FullPath() == "/api/login" {
				logger.Warn("RATELIMIT", "⚠️ Redis indisponível — usando rate limiter em memória para %s (IP: %s)", c.FullPath(), ip)
				if !checkMemoryRateLimit(key, limit, window) {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
						"success": false,
						"message": "Muitas requisições. Tente novamente mais tarde.",
					})
					return
				}
			}
			c.Next()
			return
		}

		count, err := client.Incr(c.Request.Context(), key).Result()
		if err != nil {
			if strictMemoryFallback || c.FullPath() == "/api/login" {
				logger.Warn("RATELIMIT", "⚠️ Erro no Redis — usando rate limiter em memória para %s: %v", c.FullPath(), err)
				if !checkMemoryRateLimit(key, limit, window) {
					c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
						"success": false,
						"message": "Muitas requisições. Tente novamente mais tarde.",
					})
					return
				}
			}
			c.Next()
			return
		}

		if count == 1 {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_ = client.Expire(ctx, key, window).Err()
			cancel()
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"message": "Muitas requisições. Tente novamente mais tarde.",
			})
			return
		}

		c.Next()
	}
}
