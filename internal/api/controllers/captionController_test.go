package controllers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDownloadTelegramFileUsesServerSideResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		_, _ = w.Write([]byte("photo"))
	}))
	defer server.Close()

	data, err := downloadTelegramFile(context.Background(), &http.Client{Timeout: time.Second}, server.URL+"/file/bot-secret/photo.jpg", 1024)
	if err != nil {
		t.Fatalf("download Telegram file: %v", err)
	}
	if string(data) != "photo" {
		t.Fatalf("unexpected downloaded data: %q", data)
	}
}

func TestDownloadTelegramFileRejectsOversizedResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(make([]byte, 33))
	}))
	defer server.Close()

	if _, err := downloadTelegramFile(context.Background(), server.Client(), server.URL, 32); err == nil {
		t.Fatal("expected oversized Telegram file to be rejected")
	}
}
