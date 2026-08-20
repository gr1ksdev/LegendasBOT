package repositories_test

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

func setupTestDB(t *testing.T) *gorm.DB {
	t.Helper()

	// Use in-memory SQLite for tests
	db, err := gorm.Open(sqlite.Open("file::memory:?mode=memory"), &gorm.Config{
		NamingStrategy: schema.NamingStrategy{
			TablePrefix:   "test_",
			SingularTable: false,
		},
	})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Enable foreign keys
	db.Exec("PRAGMA foreign_keys = ON;")

	// AutoMigrate
	err = db.AutoMigrate(
		&models.ConnectedAccount{},
		&models.ConnectedAccountChannel{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	// Ensure encryption key is set for service tests
	if config.SecreteKey == "" {
		config.SecreteKey = "test-secret-key-for-unit-tests-12345678"
	}
	logger.Info("TEST", "Using secret key: %s", config.SecreteKey[:8]+"...")

	return db
}

func newTestAccount(userID, telegramID int64, username string) *models.ConnectedAccount {
	return &models.ConnectedAccount{
		ID:               uuid.New().String(),
		UserID:           userID,
		TelegramUserID:   telegramID,
		Username:         username,
		EncryptedSession: "aabbccddee0011223344556677889900aabbccddee0011223344556677889900",
		Enabled:          true,
	}
}

func TestConnectedAccountRepository_Create(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(100, 10000, "testuser")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Verify it was saved
	saved, err := repo.GetByUserID(ctx, 100)
	if err != nil {
		t.Fatalf("GetByUserID() error = %v", err)
	}
	if saved == nil {
		t.Fatal("GetByUserID() returned nil after Create")
	}
	if saved.UserID != 100 {
		t.Errorf("UserID = %d, want %d", saved.UserID, 100)
	}
	if saved.Username != "testuser" {
		t.Errorf("Username = %q, want %q", saved.Username, "testuser")
	}
	if !saved.Enabled {
		t.Error("Enabled should be true by default")
	}
}

func TestConnectedAccountRepository_CreateDuplicateUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	// Create first account
	account1 := newTestAccount(200, 20000, "user1")
	err := repo.Create(ctx, account1)
	if err != nil {
		t.Fatalf("First Create() error = %v", err)
	}

	// Attempt to create second account with same UserID
	account2 := newTestAccount(200, 30000, "user2")
	err = repo.Create(ctx, account2)
	if err == nil {
		t.Error("Second Create() with same UserID should error (unique constraint)")
	}
}

func TestConnectedAccountRepository_GetByUserID_NotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account, err := repo.GetByUserID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByUserID() for non-existent user error = %v", err)
	}
	if account != nil {
		t.Error("GetByUserID() should return nil for non-existent user")
	}
}

func TestConnectedAccountRepository_GetByTelegramUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(300, 30000, "telegram_user")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Search by telegram ID
	found, err := repo.GetByTelegramUserID(ctx, 30000)
	if err != nil {
		t.Fatalf("GetByTelegramUserID() error = %v", err)
	}
	if found == nil {
		t.Fatal("GetByTelegramUserID() returned nil")
	}
	if found.UserID != 300 {
		t.Errorf("UserID = %d, want %d", found.UserID, 300)
	}

	// Non-existent telegram ID
	notFound, err := repo.GetByTelegramUserID(ctx, 99999)
	if err != nil {
		t.Fatalf("GetByTelegramUserID() for non-existent error = %v", err)
	}
	if notFound != nil {
		t.Error("GetByTelegramUserID() should return nil for non-existent")
	}
}

func TestConnectedAccountRepository_Update(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(400, 40000, "original")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update
	account.Username = "updated"
	account.Enabled = false
	err = repo.Update(ctx, account)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Verify
	saved, err := repo.GetByUserID(ctx, 400)
	if err != nil {
		t.Fatalf("GetByUserID() after update error = %v", err)
	}
	if saved.Username != "updated" {
		t.Errorf("Username after update = %q, want %q", saved.Username, "updated")
	}
	if saved.Enabled != false {
		t.Error("Enabled should be false after update")
	}
}

func TestConnectedAccountRepository_Delete(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(500, 50000, "deletable")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Delete
	err = repo.Delete(ctx, account.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify gone
	saved, err := repo.GetByUserID(ctx, 500)
	if err != nil {
		t.Fatalf("GetByUserID() after delete error = %v", err)
	}
	if saved != nil {
		t.Error("GetByUserID() should return nil after Delete")
	}
}

func TestConnectedAccountRepository_DeleteByUserID(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(600, 60000, "user_to_delete")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	err = repo.DeleteByUserID(ctx, 600)
	if err != nil {
		t.Fatalf("DeleteByUserID() error = %v", err)
	}

	saved, err := repo.GetByUserID(ctx, 600)
	if err != nil {
		t.Fatalf("GetByUserID() after DeleteByUserID error = %v", err)
	}
	if saved != nil {
		t.Error("GetByUserID() should return nil after DeleteByUserID")
	}
}

func TestConnectedAccountRepository_HasActiveAccount(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	// No account yet
	active, err := repo.HasActiveAccount(ctx, 700)
	if err != nil {
		t.Fatalf("HasActiveAccount() for non-existent error = %v", err)
	}
	if active {
		t.Error("HasActiveAccount() should be false for non-existent user")
	}

	// Create enabled account
	account := newTestAccount(700, 70000, "active_user")
	err = repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	active, err = repo.HasActiveAccount(ctx, 700)
	if err != nil {
		t.Fatalf("HasActiveAccount() error = %v", err)
	}
	if !active {
		t.Error("HasActiveAccount() should be true for user with enabled account")
	}

	// Disable account
	account.Enabled = false
	err = repo.Update(ctx, account)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	active, err = repo.HasActiveAccount(ctx, 700)
	if err != nil {
		t.Fatalf("HasActiveAccount() after disable error = %v", err)
	}
	if active {
		t.Error("HasActiveAccount() should be false for disabled account")
	}
}

func TestConnectedAccountRepository_UpdateLastUsed(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(800, 80000, "last_used_test")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Update last used
	err = repo.UpdateLastUsed(ctx, account.ID)
	if err != nil {
		t.Fatalf("UpdateLastUsed() error = %v", err)
	}

	saved, err := repo.GetByUserID(ctx, 800)
	if err != nil {
		t.Fatalf("GetByUserID() after UpdateLastUsed error = %v", err)
	}
	if saved.LastUsedAt == nil {
		t.Fatal("LastUsedAt should not be nil after UpdateLastUsed")
	}
	if time.Since(*saved.LastUsedAt) > 5*time.Second {
		t.Error("LastUsedAt should be recent (within 5 seconds)")
	}
}

func TestConnectedAccountRepository_CountAll(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	// Initial count should be 0
	count, err := repo.CountAll(ctx)
	if err != nil {
		t.Fatalf("CountAll() error = %v", err)
	}
	if count != 0 {
		t.Errorf("CountAll() = %d, want 0", count)
	}

	// Add 3 accounts
	for i := 0; i < 3; i++ {
		userID := int64(900 + i)
		telegramID := int64(90000 + i)
		account := newTestAccount(userID, telegramID, "user")
		if err := repo.Create(ctx, account); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	count, err = repo.CountAll(ctx)
	if err != nil {
		t.Fatalf("CountAll() error = %v", err)
	}
	if count != 3 {
		t.Errorf("CountAll() = %d, want 3", count)
	}
}

// --- ConnectedAccountChannel tests ---

func TestConnectedAccountRepository_ChannelAuthorization(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	// Create account first
	account := newTestAccount(1000, 100000, "channel_owner")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add channel authorization
	accChannel := &models.ConnectedAccountChannel{
		ID:                 uuid.New().String(),
		ConnectedAccountID: account.ID,
		ChannelID:          -1001234567890,
		Enabled:            true,
	}
	err = repo.AddChannel(ctx, accChannel)
	if err != nil {
		t.Fatalf("AddChannel() error = %v", err)
	}

	// Check authorization
	authorized, err := repo.IsChannelAuthorized(ctx, account.ID, -1001234567890)
	if err != nil {
		t.Fatalf("IsChannelAuthorized() error = %v", err)
	}
	if !authorized {
		t.Error("IsChannelAuthorized() should be true for authorized channel")
	}

	// Check non-authorized channel
	notAuthorized, err := repo.IsChannelAuthorized(ctx, account.ID, -1009999999999)
	if err != nil {
		t.Fatalf("IsChannelAuthorized() for non-authorized error = %v", err)
	}
	if notAuthorized {
		t.Error("IsChannelAuthorized() should be false for non-authorized channel")
	}

	// Get channels
	channels, err := repo.GetChannels(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetChannels() error = %v", err)
	}
	if len(channels) != 1 {
		t.Fatalf("GetChannels() returned %d channels, want 1", len(channels))
	}
	if channels[0].ChannelID != -1001234567890 {
		t.Errorf("ChannelID = %d, want %d", channels[0].ChannelID, -1001234567890)
	}
}

func TestConnectedAccountRepository_RemoveChannel(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(1100, 110000, "remove_channel_test")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	accChannel := &models.ConnectedAccountChannel{
		ID:                 uuid.New().String(),
		ConnectedAccountID: account.ID,
		ChannelID:          -1001111111111,
		Enabled:            true,
	}
	err = repo.AddChannel(ctx, accChannel)
	if err != nil {
		t.Fatalf("AddChannel() error = %v", err)
	}

	// Remove channel
	err = repo.RemoveChannel(ctx, account.ID, -1001111111111)
	if err != nil {
		t.Fatalf("RemoveChannel() error = %v", err)
	}

	authorized, err := repo.IsChannelAuthorized(ctx, account.ID, -1001111111111)
	if err != nil {
		t.Fatalf("IsChannelAuthorized() after remove error = %v", err)
	}
	if authorized {
		t.Error("IsChannelAuthorized() should be false after RemoveChannel")
	}
}

func TestConnectedAccountRepository_RemoveAllChannels(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(1200, 120000, "remove_all_test")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add multiple channels
	for _, chID := range []int64{-1001, -1002, -1003} {
		accChannel := &models.ConnectedAccountChannel{
			ID:                 uuid.New().String(),
			ConnectedAccountID: account.ID,
			ChannelID:          chID,
			Enabled:            true,
		}
		if err := repo.AddChannel(ctx, accChannel); err != nil {
			t.Fatalf("AddChannel() error = %v", err)
		}
	}

	// Remove all
	err = repo.RemoveAllChannels(ctx, account.ID)
	if err != nil {
		t.Fatalf("RemoveAllChannels() error = %v", err)
	}

	// Verify none remain
	channels, err := repo.GetChannels(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetChannels() after RemoveAll error = %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("GetChannels() returned %d channels after RemoveAll, want 0", len(channels))
	}
}

func TestConnectedAccountRepository_CascadeDelete(t *testing.T) {
	db := setupTestDB(t)
	repo := repositories.NewConnectedAccountRepository(db)
	ctx := context.Background()

	account := newTestAccount(1300, 130000, "cascade_test")
	err := repo.Create(ctx, account)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	// Add channel
	accChannel := &models.ConnectedAccountChannel{
		ID:                 uuid.New().String(),
		ConnectedAccountID: account.ID,
		ChannelID:          -1001300,
		Enabled:            true,
	}
	err = repo.AddChannel(ctx, accChannel)
	if err != nil {
		t.Fatalf("AddChannel() error = %v", err)
	}

	// Delete the account (should cascade delete channels)
	err = repo.Delete(ctx, account.ID)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	// Verify channels also deleted
	channels, err := repo.GetChannels(ctx, account.ID)
	if err != nil {
		t.Fatalf("GetChannels() after cascade error = %v", err)
	}
	if len(channels) != 0 {
		t.Errorf("GetChannels() returned %d after cascade, want 0", len(channels))
	}
}
