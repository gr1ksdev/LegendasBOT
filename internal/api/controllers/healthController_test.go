package controllers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestHealthzController(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	ctrl := NewHealthController(nil)
	router.GET("/healthz", ctrl.Healthz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/healthz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 for /healthz, got %d", w.Code)
	}

	expectedBody := `{"status":"ok"}`
	if w.Body.String() != expectedBody {
		t.Errorf("Expected body %s, got %s", expectedBody, w.Body.String())
	}
}

func TestReadyzControllerUnhealthy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// Container nulo -> DB e Redis indisponíveis -> 503
	ctrl := NewHealthController(nil)
	router.GET("/readyz", ctrl.Readyz)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/readyz", nil)
	router.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 Service Unavailable for /readyz without DB/Redis, got %d", w.Code)
	}
}
