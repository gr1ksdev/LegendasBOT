package controllers

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestBindLimitedJSONRejectsOversizedPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	response := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(response)
	ctx.Request = httptest.NewRequest(http.MethodPut, "/", bytes.NewReader(bytes.Repeat([]byte("x"), maxPermissionRequestBytes+1)))

	var body map[string]any
	if err := bindLimitedJSON(ctx, &body); err == nil {
		t.Fatal("expected oversized permission payload to be rejected")
	}
}
