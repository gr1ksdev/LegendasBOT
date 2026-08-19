package handlers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func resetClientErrorRateLimiter() {
	clientErrorRateLimiter.Lock()
	defer clientErrorRateLimiter.Unlock()
	clientErrorRateLimiter.clients = make(map[string]clientErrorRateWindow)
}

func TestClientErrorRateLimiterCleansExpiredClients(t *testing.T) {
	resetClientErrorRateLimiter()
	now := time.Now()
	if !allowClientError("198.51.100.1", now) {
		t.Fatal("expected initial request to be allowed")
	}
	if !allowClientError("198.51.100.2", now.Add(2*time.Minute)) {
		t.Fatal("expected request after cleanup to be allowed")
	}

	clientErrorRateLimiter.Lock()
	defer clientErrorRateLimiter.Unlock()
	if len(clientErrorRateLimiter.clients) != 1 {
		t.Fatalf("expected expired clients to be removed, found %d", len(clientErrorRateLimiter.clients))
	}
}

func TestClientErrorHandlerRejectsOversizedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetClientErrorRateLimiter()

	router := gin.New()
	router.POST("/api/log/client-error", ClientErrorHandler(nil))
	payload := bytes.Repeat([]byte("x"), clientErrorMaxBodyBytes+1)
	req := httptest.NewRequest(http.MethodPost, "/api/log/client-error", bytes.NewReader(payload))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, req)
	if response.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d", http.StatusRequestEntityTooLarge, response.Code)
	}
}

func TestClientErrorHandlerRateLimitsClient(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetClientErrorRateLimiter()

	router := gin.New()
	router.POST("/api/log/client-error", ClientErrorHandler(nil))
	for i := 0; i < clientErrorMaxPerMinute; i++ {
		request := httptest.NewRequest(http.MethodPost, "/api/log/client-error", bytes.NewBufferString(`{"message":"client error"}`))
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if response.Code != http.StatusNoContent {
			t.Fatalf("request %d: expected status %d, got %d", i+1, http.StatusNoContent, response.Code)
		}
	}

	request := httptest.NewRequest(http.MethodPost, "/api/log/client-error", bytes.NewBufferString(`{"message":"client error"}`))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusTooManyRequests {
		t.Fatalf("expected status %d, got %d", http.StatusTooManyRequests, response.Code)
	}
}
