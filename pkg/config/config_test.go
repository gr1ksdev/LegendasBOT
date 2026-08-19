package config

import (
	"os"
	"testing"
)

func TestConfigValidation(t *testing.T) {
	t.Run("Valid default test mode config", func(t *testing.T) {
		Load()
		if err := Validate(); err != nil {
			t.Errorf("Expected config to be valid in test mode, got error: %v", err)
		}
	})

	t.Run("Fails when SECRET_KEY is too short (< 32 chars)", func(t *testing.T) {
		oldSecret := os.Getenv("SECRET_KEY")
		defer func() {
			os.Setenv("SECRET_KEY", oldSecret)
			Load()
		}()

		os.Setenv("SECRET_KEY", "short_secret_key")
		Load()

		err := Validate()
		if err == nil {
			t.Error("Expected validation error for short SECRET_KEY, got nil")
		}
	})

	t.Run("Fails when WebhookURL is set without TelegramWebhookSecret", func(t *testing.T) {
		oldWebhook := os.Getenv("WEBHOOK_URL")
		oldSecret := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
		defer func() {
			os.Setenv("WEBHOOK_URL", oldWebhook)
			os.Setenv("TELEGRAM_WEBHOOK_SECRET", oldSecret)
			Load()
		}()

		os.Setenv("WEBHOOK_URL", "https://example.com/webhook")
		os.Setenv("TELEGRAM_WEBHOOK_SECRET", "")
		Load()

		err := Validate()
		if err == nil {
			t.Error("Expected validation error when WEBHOOK_URL is set without TELEGRAM_WEBHOOK_SECRET, got nil")
		}
	})

	t.Run("Fails when MTProto credentials are incomplete", func(t *testing.T) {
		oldAppID := os.Getenv("MTPROTO_APP_ID")
		oldAppHash := os.Getenv("MTPROTO_APP_HASH")
		defer func() {
			os.Setenv("MTPROTO_APP_ID", oldAppID)
			os.Setenv("MTPROTO_APP_HASH", oldAppHash)
			Load()
		}()

		os.Setenv("MTPROTO_APP_ID", "123456")
		os.Setenv("MTPROTO_APP_HASH", "")
		Load()

		err := Validate()
		if err == nil {
			t.Error("Expected validation error for partial MTProto config, got nil")
		}
	})

	t.Run("Fails when WebhookSecret contains invalid characters", func(t *testing.T) {
		oldWebhook := os.Getenv("WEBHOOK_URL")
		oldSecret := os.Getenv("TELEGRAM_WEBHOOK_SECRET")
		defer func() {
			os.Setenv("WEBHOOK_URL", oldWebhook)
			os.Setenv("TELEGRAM_WEBHOOK_SECRET", oldSecret)
			Load()
		}()

		os.Setenv("WEBHOOK_URL", "https://example.com/webhook")
		os.Setenv("TELEGRAM_WEBHOOK_SECRET", "invalid secret with spaces!")
		Load()

		err := Validate()
		if err == nil {
			t.Error("Expected validation error for TELEGRAM_WEBHOOK_SECRET with spaces/invalid chars, got nil")
		}
	})
}
