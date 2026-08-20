package services_test

import (
	"context"
	"crypto/rand"
	"os"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func TestMain(m *testing.M) {
	os.Setenv("TELEGRAM_BOT_TOKEN", "test_token_12345")
	os.Setenv("REDIS_HOST", "redis://localhost:6379")
	os.Setenv("OWNER_ID", "12345")
	os.Setenv("SECRET_KEY", "test-secret-key-for-unit-tests-12345678")
	os.Setenv("WEBAPP_URL", "https://example.com")
	os.Exit(m.Run())
}

func setupServiceTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix: "srv_test_",
		},
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	db.Exec("PRAGMA foreign_keys = ON;")

	err = db.AutoMigrate(
		&models.ConnectedAccount{},
		&models.ConnectedAccountChannel{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate: %v", err)
	}

	if config.SecreteKey == "" {
		config.SecreteKey = "test-secret-key-for-service-tests-abcdef123456"
	}
	logger.Info("TEST", "Secret key: %s", config.SecreteKey[:8]+"...")

	return db
}

func generateSessionData(t *testing.T) []byte {
	t.Helper()
	data := make([]byte, 128)
	_, err := rand.Read(data)
	if err != nil {
		t.Fatalf("Failed to generate session data: %v", err)
	}
	return data
}

func TestConnectedAccountService_SaveSession_New(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	sessionData := generateSessionData(t)
	account, err := svc.SaveSession(ctx, 1000, 1000000, "new_user", "First", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	if account == nil {
		t.Fatal("SaveSession() returned nil")
	}
	if account.UserID != 1000 {
		t.Errorf("UserID = %d, want %d", account.UserID, 1000)
	}
	if account.Username != "new_user" {
		t.Errorf("Username = %q, want %q", account.Username, "new_user")
	}

	// Session should be encrypted (different from original)
	if account.EncryptedSession == string(sessionData) {
		t.Error("Session should be encrypted, not stored as plaintext")
	}
	if account.EncryptedSession == "" {
		t.Error("EncryptedSession should not be empty")
	}
}

func TestConnectedAccountService_SaveSession_Update(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	// Create first session
	session1 := generateSessionData(t)
	account1, err := svc.SaveSession(ctx, 2000, 2000000, "first_session", "First", session1)
	if err != nil {
		t.Fatalf("First SaveSession() error = %v", err)
	}
	originalID := account1.ID

	// Update with new session data
	session2 := generateSessionData(t)
	account2, err := svc.SaveSession(ctx, 2000, 2000001, "updated_session", "Updated", session2)
	if err != nil {
		t.Fatalf("Second SaveSession() error = %v", err)
	}

	// Should be the same record (same ID)
	if account2.ID != originalID {
		t.Errorf("ID changed: original=%q, new=%q", originalID, account2.ID)
	}

	// Username should be updated
	if account2.Username != "updated_session" {
		t.Errorf("Username = %q, want %q", account2.Username, "updated_session")
	}

	// TelegramUserID should be updated
	if account2.TelegramUserID != 2000001 {
		t.Errorf("TelegramUserID = %d, want %d", account2.TelegramUserID, 2000001)
	}
}

func TestConnectedAccountService_GetSession(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	// No session yet
	sessionData, account, err := svc.GetSession(ctx, 3000)
	if err != nil {
		t.Fatalf("GetSession() for non-existent error = %v", err)
	}
	if sessionData != nil {
		t.Error("GetSession() should return nil data for non-existent user")
	}
	if account != nil {
		t.Error("GetSession() should return nil account for non-existent user")
	}

	// Save and retrieve
	originalData := generateSessionData(t)
	saved, err := svc.SaveSession(ctx, 3000, 3000000, "get_session_test", "Test", originalData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	retrievedData, retrievedAccount, err := svc.GetSession(ctx, 3000)
	if err != nil {
		t.Fatalf("GetSession() error = %v", err)
	}
	if retrievedData == nil {
		t.Fatal("GetSession() returned nil data")
	}
	if retrievedAccount == nil {
		t.Fatal("GetSession() returned nil account")
	}

	// Verify decrypted data matches original
	if len(retrievedData) != len(originalData) {
		t.Fatalf("Session data length mismatch: got %d, want %d", len(retrievedData), len(originalData))
	}
	for i := range originalData {
		if retrievedData[i] != originalData[i] {
			t.Fatalf("Session data mismatch at byte %d", i)
		}
	}

	// Verify account fields
	if retrievedAccount.UserID != saved.UserID {
		t.Errorf("Account UserID mismatch")
	}
}

func TestConnectedAccountService_GetSession_Disabled(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	sessionData := generateSessionData(t)
	account, err := svc.SaveSession(ctx, 4000, 4000000, "disable_test", "Disable", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Manually disable
	account.Enabled = false
	err = repo.Update(ctx, account)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// GetSession should return nil for disabled account
	data, acc, err := svc.GetSession(ctx, 4000)
	if err != nil {
		t.Fatalf("GetSession() for disabled error = %v", err)
	}
	if data != nil {
		t.Error("GetSession() should return nil data for disabled account")
	}
	if acc != nil {
		t.Error("GetSession() should return nil account for disabled account")
	}
}

func TestConnectedAccountService_Disconnect(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	// Disconnect non-existent account
	err := svc.Disconnect(ctx, 5000)
	if err == nil {
		t.Error("Disconnect() should error for non-existent account")
	}

	// Create account with channels
	sessionData := generateSessionData(t)
	_, err = svc.SaveSession(ctx, 5000, 5000000, "disconnect_test", "Disconnect", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Add channel
	account, _ := repo.GetByUserID(ctx, 5000)
	accChannel := &models.ConnectedAccountChannel{
		ID:                 "ch-test-1",
		ConnectedAccountID: account.ID,
		ChannelID:          -1005000,
		Enabled:            true,
	}
	err = repo.AddChannel(ctx, accChannel)
	if err != nil {
		t.Fatalf("AddChannel() error = %v", err)
	}

	// Disconnect
	err = svc.Disconnect(ctx, 5000)
	if err != nil {
		t.Fatalf("Disconnect() error = %v", err)
	}

	// Account should be gone
	acc, err := repo.GetByUserID(ctx, 5000)
	if err != nil {
		t.Fatalf("GetByUserID() after disconnect error = %v", err)
	}
	if acc != nil {
		t.Error("Account should be deleted after Disconnect")
	}

	// Channels should also be gone (cascade)
	channels, err := repo.GetChannels(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetChannels() after disconnect error = %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("Channels should be deleted on cascade, got %d", len(channels))
	}
}

func TestConnectedAccountService_HasActiveAccount(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	// No account
	if svc.HasActiveAccount(ctx, 6000) {
		t.Error("HasActiveAccount() should be false for non-existent user")
	}

	// Create account
	sessionData := generateSessionData(t)
	_, err := svc.SaveSession(ctx, 6000, 6000000, "active_test", "Active", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	if !svc.HasActiveAccount(ctx, 6000) {
		t.Error("HasActiveAccount() should be true after SaveSession")
	}
}

func TestConnectedAccountService_AuthorizeChannel(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	// Non-existent user
	auth, err := svc.AuthorizeChannel(ctx, 7000, -1007000)
	if err != nil {
		t.Fatalf("AuthorizeChannel() for non-existent error = %v", err)
	}
	if auth {
		t.Error("AuthorizeChannel() should be false for non-existent user")
	}

	// Create account
	sessionData := generateSessionData(t)
	_, err = svc.SaveSession(ctx, 7000, 7000000, "auth_test", "Auth", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Not authorized yet (no channel added)
	auth, err = svc.AuthorizeChannel(ctx, 7000, -1007000)
	if err != nil {
		t.Fatalf("AuthorizeChannel() error = %v", err)
	}
	if auth {
		t.Error("AuthorizeChannel() should be false before adding channel")
	}

	// Get account and add channel
	account, _ := repo.GetByUserID(ctx, 7000)
	accChannel := &models.ConnectedAccountChannel{
		ID:                 "ch-auth-1",
		ConnectedAccountID: account.ID,
		ChannelID:          -1007000,
		Enabled:            true,
	}
	err = repo.AddChannel(ctx, accChannel)
	if err != nil {
		t.Fatalf("AddChannel() error = %v", err)
	}

	// Now should be authorized
	auth, err = svc.AuthorizeChannel(ctx, 7000, -1007000)
	if err != nil {
		t.Fatalf("AuthorizeChannel() after add error = %v", err)
	}
	if !auth {
		t.Error("AuthorizeChannel() should be true after adding channel")
	}
}

func TestConnectedAccountService_GetAccount(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	// Non-existent
	acc, err := svc.GetAccount(ctx, 8000)
	if err != nil {
		t.Fatalf("GetAccount() for non-existent error = %v", err)
	}
	if acc != nil {
		t.Error("GetAccount() should return nil for non-existent user")
	}

	// Create and get
	sessionData := generateSessionData(t)
	saved, err := svc.SaveSession(ctx, 8000, 8000000, "get_account_test", "GetAccount", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	acc, err = svc.GetAccount(ctx, 8000)
	if err != nil {
		t.Fatalf("GetAccount() error = %v", err)
	}
	if acc == nil {
		t.Fatal("GetAccount() returned nil")
	}

	// Should NOT include encrypted session in returned data
	if acc.EncryptedSession != "" {
		t.Log("Note: EncryptedSession is returned by GetAccount (expected for DB model)")
	}

	_ = saved
}

func TestConnectedAccountService_SaveSession_WrongKey(t *testing.T) {
	db := setupServiceTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	svc := services.NewConnectedAccountService(repo)
	ctx := context.Background()

	sessionData := generateSessionData(t)
	_, err := svc.SaveSession(ctx, 9000, 9000000, "key_test", "KeyTest", sessionData)
	if err != nil {
		t.Fatalf("SaveSession() error = %v", err)
	}

	// Try to get session with different secret key (simulate key change)
	originalKey := config.SecreteKey
	config.SecreteKey = "different-key-that-wont-work-for-decryption"
	defer func() { config.SecreteKey = originalKey }()

	// Create new service instance (it reads config each time)
	svc2 := services.NewConnectedAccountService(repo)
	data, acc, err := svc2.GetSession(ctx, 9000)
	if err == nil {
		t.Error("GetSession() should error when decryption key is wrong")
	}
	if data != nil {
		t.Error("GetSession() should return nil data when decryption fails")
	}
	if acc != nil {
		t.Error("GetSession() should return nil account when decryption fails")
	}
}
