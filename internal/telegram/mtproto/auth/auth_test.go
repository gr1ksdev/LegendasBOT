package auth_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/leirbagxis/FreddyBot/internal/telegram/mtproto/auth"
	"github.com/redis/go-redis/v9"
)

var errMockSave = fmt.Errorf("mock save error")

type mockAccountSaver struct {
	savedUserID     int64
	savedTelegramID int64
	savedUsername   string
	savedFirstName  string
	saveCallCount   int
	failSave        bool
}

func (m *mockAccountSaver) SaveSession(ctx context.Context, userID int64, telegramUserID int64, username string, firstName string, sessionData []byte) error {
	m.saveCallCount++
	if m.failSave {
		return errMockSave
	}
	m.savedUserID = userID
	m.savedTelegramID = telegramUserID
	m.savedUsername = username
	m.savedFirstName = firstName
	return nil
}

func setupAuthTest(t *testing.T) (*miniredis.Miniredis, *auth.Service, *mockAccountSaver) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("Failed to start miniredis: %v", err)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	saver := &mockAccountSaver{}
	// appID=0 e appHash="" simulam uma instalacao sem credenciais MTProto.
	svc := auth.NewService(redisClient, 0, "", saver)

	t.Cleanup(func() {
		redisClient.Close()
		mr.Close()
	})

	return mr, svc, saver
}

// saveState helper para criar estado no Redis diretamente (simula o que SendCode faria).
func saveState(t *testing.T, mr *miniredis.Miniredis, userID int64, phoneNumber string) {
	t.Helper()
	key := fmt.Sprintf("mtproto_auth:%d", userID)
	val := fmt.Sprintf(`{"user_id":%d,"phone_number":"%s","phone_code_hash":"test_hash_%d"}`, userID, phoneNumber, userID)
	mr.Set(key, val)
	mr.SetTTL(key, 5*time.Minute)
}

func TestAuthService_SendCodeWithoutConfigurationFailsClosed(t *testing.T) {
	mr, svc, _ := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	status, err := svc.SendCode(ctx, 100, "+5511999999999")
	if err != nil {
		t.Fatalf("SendCode() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}
	if status.Error == "" {
		t.Fatal("expected a configuration error")
	}

	if mr.Exists("mtproto_auth:100") {
		t.Error("auth state must not be created without a real MTProto session")
	}
}

func TestAuthService_SendCode_PhoneTooLong(t *testing.T) {
	mr, svc, _ := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	longPhone := "+55" + string(make([]byte, 30))
	status, err := svc.SendCode(ctx, 102, longPhone)
	if err != nil {
		t.Fatalf("SendCode() for long phone error = %v", err)
	}
	if status.Step != "error" || status.Error == "" {
		t.Errorf("expected a fail-closed error, got %+v", status)
	}
}

func TestAuthService_VerifyCode_NoState(t *testing.T) {
	mr, svc, _ := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	// Tentar verificar codigo sem ter iniciado o fluxo
	status, err := svc.VerifyCode(ctx, 200, "12345")
	if err != nil {
		t.Fatalf("VerifyCode() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}
	if status.Error == "" {
		t.Error("expected configuration error")
	}
}

func TestAuthService_VerifyCodeWithoutConfigurationDoesNotSaveSession(t *testing.T) {
	mr, svc, saver := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	// Um estado antigo nunca pode habilitar uma conta quando MTProto esta indisponivel.
	saveState(t, mr, 300, "+5511999999999")

	status, err := svc.VerifyCode(ctx, 300, "12345")
	if err != nil {
		t.Fatalf("VerifyCode() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}

	if saver.saveCallCount != 0 {
		t.Errorf("saveCallCount = %d, want 0", saver.saveCallCount)
	}
}

func TestAuthService_VerifyPasswordWithoutConfigurationDoesNotSaveSession(t *testing.T) {
	mr, svc, saver := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	// Simular estado apos SendCode + VerifyCode com 2FA detectado
	key := fmt.Sprintf("mtproto_auth:%d", 400)
	val := `{"user_id":400,"phone_number":"+5511999999999","phone_code_hash":"test_hash_400","has_password":true}`
	mr.Set(key, val)
	mr.SetTTL(key, 5*time.Minute)

	status, err := svc.VerifyPassword(ctx, 400, "minha_senha")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}

	if saver.saveCallCount != 0 {
		t.Errorf("saveCallCount = %d, want 0", saver.saveCallCount)
	}
}

func TestAuthService_VerifyPassword_NoState(t *testing.T) {
	mr, svc, _ := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	status, err := svc.VerifyPassword(ctx, 500, "senha")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}
	if status.Error == "" {
		t.Error("expected configuration error")
	}
}

func TestAuthService_GetStatus_NoState(t *testing.T) {
	mr, svc, _ := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	status, err := svc.GetStatus(ctx, 600)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}
}

func TestAuthService_StateExpiration(t *testing.T) {
	mr, svc, _ := setupAuthTest(t)
	defer mr.Close()
	ctx := context.Background()

	// Salvar estado com TTL curto (manualmente no miniredis)
	key := fmt.Sprintf("mtproto_auth:%d", 700)
	val := `{"user_id":700,"phone_number":"+5511999999999","phone_code_hash":"test_hash_700"}`
	mr.Set(key, val)
	mr.SetTTL(key, 1*time.Millisecond)

	// Avancar o tempo para expirar
	mr.FastForward(10 * time.Millisecond)

	// Tentar verificar codigo com estado expirado
	status, err := svc.VerifyCode(ctx, 700, "12345")
	if err != nil {
		t.Fatalf("VerifyCode() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}
	if status.Error == "" {
		t.Error("expected configuration error")
	}

	// GetStatus tambem falha fechado sem credenciais.
	status, err = svc.GetStatus(ctx, 700)
	if err != nil {
		t.Fatalf("GetStatus() error = %v", err)
	}
	if status.Step != "error" {
		t.Errorf("Step = %q, want %q", status.Step, "error")
	}
}
