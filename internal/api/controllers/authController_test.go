package controllers

import "testing"

func TestParseTelegramLoginUserPreservesExactID(t *testing.T) {
	user, err := parseTelegramLoginUser(`{"id":12345,"first_name":"Ana","username":"ana"}`)
	if err != nil {
		t.Fatalf("parse user: %v", err)
	}
	if user.ID != 12345 {
		t.Fatalf("expected exact Telegram ID 12345, got %d", user.ID)
	}
	if user.ID == 1234 {
		t.Fatal("a prefix of the Telegram ID must never be accepted as the same user")
	}
}

func TestParseTelegramLoginUserRejectsInvalidJSON(t *testing.T) {
	if _, err := parseTelegramLoginUser(`{"id":`); err == nil {
		t.Fatal("expected malformed Telegram user payload to be rejected")
	}
}

func TestTelegramUserMatchesRequiresExactID(t *testing.T) {
	raw := `{"id":12345,"first_name":"Ana"}`
	if !telegramUserMatches(raw, 12345) {
		t.Fatal("expected exact Telegram ID to match")
	}
	if telegramUserMatches(raw, 1234) {
		t.Fatal("a Telegram ID prefix must not authenticate another user")
	}
}
