package telegram

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mymmrac/telego"
)

func webhookRequest(body []byte, secret string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
	if secret != "" {
		req.Header.Set(telego.WebhookSecretTokenHeader, secret)
	}
	return req
}

func TestWebhookHandlerRequiresSecretAndBoundsBody(t *testing.T) {
	updates := make(chan telego.Update, 1)
	handler := newWebhookHandler("expected-secret", updates)

	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, webhookRequest([]byte(`{"update_id":1}`), "wrong-secret"))
	if unauthorized.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d, want %d", unauthorized.Code, http.StatusUnauthorized)
	}

	ok := httptest.NewRecorder()
	handler.ServeHTTP(ok, webhookRequest([]byte(`{"update_id":2}`), "expected-secret"))
	if ok.Code != http.StatusOK {
		t.Fatalf("valid webhook status = %d, want %d", ok.Code, http.StatusOK)
	}
	if update := <-updates; update.UpdateID != 2 {
		t.Fatalf("queued update ID = %d, want 2", update.UpdateID)
	}

	tooLarge := httptest.NewRecorder()
	largeBody := append([]byte(`{"data":"`), bytes.Repeat([]byte("a"), int(maxWebhookBodyBytes+1))...)
	largeBody = append(largeBody, []byte(`"}`)...)
	handler.ServeHTTP(tooLarge, webhookRequest(largeBody, "expected-secret"))
	if tooLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("large webhook status = %d, want %d", tooLarge.Code, http.StatusRequestEntityTooLarge)
	}
}

func TestWebhookHandlerRejectsWhenQueueIsFull(t *testing.T) {
	updates := make(chan telego.Update, 1)
	updates <- telego.Update{UpdateID: 1}
	handler := newWebhookHandler("expected-secret", updates)

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, webhookRequest([]byte(`{"update_id":2}`), "expected-secret"))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("queue full status = %d, want %d", response.Code, http.StatusServiceUnavailable)
	}
}
