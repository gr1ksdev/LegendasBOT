package middleware

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestMemoryRateLimiterConcurrent(t *testing.T) {
	limiter := &memoryRateLimiter{
		store: make(map[string]*memoryLimiterEntry),
	}

	key := "test_concurrent_key"
	limit := 100
	window := time.Second * 5

	var wg sync.WaitGroup
	workers := 50
	reqsPerWorker := 10

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < reqsPerWorker; j++ {
				limiter.Allow(key, limit, window)
			}
		}()
	}

	wg.Wait()

	// 500 chamadas feitas para um limite de 100 -> as primeiras 100 devem passar, o resto falhar
	if limiter.Allow(key, limit, window) {
		t.Error("Expected rate limit to block request 501, but allowed it")
	}
}

func TestRateLimitStrictFallbackMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.POST("/api/login", RateLimitStrict(2, time.Second*10, true), func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Requisição 1: OK
	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/login", nil)
	req1.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w1, req1)
	if w1.Code != http.StatusOK {
		t.Errorf("Expected 200 OK on req 1, got %d", w1.Code)
	}

	// Requisição 2: OK
	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/login", nil)
	req2.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w2, req2)
	if w2.Code != http.StatusOK {
		t.Errorf("Expected 200 OK on req 2, got %d", w2.Code)
	}

	// Requisição 3: 429 Too Many Requests
	w3 := httptest.NewRecorder()
	req3, _ := http.NewRequest("POST", "/api/login", nil)
	req3.RemoteAddr = "192.168.1.1:12345"
	r.ServeHTTP(w3, req3)
	if w3.Code != http.StatusTooManyRequests {
		t.Errorf("Expected 429 Too Many Requests on req 3, got %d", w3.Code)
	}
}
