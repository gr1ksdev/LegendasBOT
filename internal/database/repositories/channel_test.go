package repositories

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

func TestDeleteChannelWithRelations(t *testing.T) {
	// Setup in-memory sqlite
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	// Ativar foreign keys para que OnDelete:CASCADE funcione no SQLite
	db.Exec("PRAGMA foreign_keys = ON;")

	// Migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.Channel{},
		&models.DefaultCaption{},
		&models.MessagePermission{},
		&models.ButtonsPermission{},
		&models.Button{},
		&models.Separator{},
		&models.CustomCaption{},
		&models.CustomCaptionButton{},
	)
	if err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := NewChannelRepository(db)

	// Create a channel with some relations
	channelID := int64(123)
	ownerID := int64(1)

	// Create owner first due to FK constraint
	if err := db.Create(&models.User{UserId: ownerID, FirstName: "Owner"}).Error; err != nil {
		t.Fatalf("failed to create owner: %v", err)
	}

	channel := &models.Channel{
		ID:      channelID,
		OwnerID: ownerID,
		Title:   "Test Channel",
		DefaultCaption: &models.DefaultCaption{
			CaptionID: "cap1",
			Caption:   "Hello",
			MessagePermission: &models.MessagePermission{
				MessagePermissionID: "msg1",
			},
			ButtonsPermission: &models.ButtonsPermission{
				ButtonsPermissionID: "btn1",
			},
		},
		Buttons: []models.Button{
			{ButtonID: "b1", NameButton: "Btn 1"},
		},
	}

	if err := db.Create(channel).Error; err != nil {
		t.Fatalf("failed to create channel: %v", err)
	}

	// Verify it exists
	var count int64
	db.Model(&models.Channel{}).Count(&count)
	if count != 1 {
		t.Errorf("expected 1 channel, got %d", count)
	}

	// Delete
	err = repo.DeleteChannelWithRelations(context.Background(), ownerID, channelID)
	if err != nil {
		t.Fatalf("failed to delete channel: %v", err)
	}

	// Verify it's gone
	db.Model(&models.Channel{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 channels, got %d", count)
	}

	// Verify relations are also gone
	db.Model(&models.DefaultCaption{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 default captions, got %d", count)
	}
	db.Model(&models.Button{}).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 buttons, got %d", count)
	}
}
