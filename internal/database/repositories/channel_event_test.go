package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

func TestChannelEventRepository_DeleteAll(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&models.ChannelEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := NewChannelEventRepository(db)
	ctx := context.Background()

	// Insert 3 test events
	for i := 1; i <= 3; i++ {
		event := &models.ChannelEvent{
			ID:        string(rune('a' + i)),
			ChannelID: int64(100 + i),
			Source:    "channel_post",
			EventType: "post_received",
			Status:    "success",
		}
		if err := repo.Create(ctx, event); err != nil {
			t.Fatalf("failed to create event: %v", err)
		}
	}

	// Verify count is 3
	events, total, err := repo.List(ctx, ChannelEventFilters{})
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if total != 3 || len(events) != 3 {
		t.Fatalf("expected 3 events, got total=%d len=%d", total, len(events))
	}

	// Delete all
	deleted, err := repo.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("failed to delete all events: %v", err)
	}
	if deleted != 3 {
		t.Errorf("expected 3 deleted rows, got %d", deleted)
	}

	// Verify count is 0
	_, totalAfter, err := repo.List(ctx, ChannelEventFilters{})
	if err != nil {
		t.Fatalf("failed to list events after delete: %v", err)
	}
	if totalAfter != 0 {
		t.Errorf("expected 0 events, got %d", totalAfter)
	}
}

func TestChannelEventRepository_DeleteOlderThan(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&models.ChannelEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := NewChannelEventRepository(db)
	ctx := context.Background()

	// Insert old event
	oldEvent := &models.ChannelEvent{
		ID:        "old-1",
		ChannelID: 101,
		Source:    "channel_post",
		EventType: "post_received",
		Status:    "success",
		CreatedAt: time.Now().AddDate(0, 0, -40),
	}
	if err := db.Create(oldEvent).Error; err != nil {
		t.Fatalf("failed to create old event: %v", err)
	}

	// Insert recent event
	recentEvent := &models.ChannelEvent{
		ID:        "recent-1",
		ChannelID: 102,
		Source:    "channel_post",
		EventType: "post_received",
		Status:    "success",
		CreatedAt: time.Now(),
	}
	if err := db.Create(recentEvent).Error; err != nil {
		t.Fatalf("failed to create recent event: %v", err)
	}

	// Delete older than 30 days
	cutoff := time.Now().AddDate(0, 0, -30)
	deleted, err := repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		t.Fatalf("failed to delete older events: %v", err)
	}
	if deleted != 1 {
		t.Errorf("expected 1 deleted row, got %d", deleted)
	}

	// Verify only recent event remains
	events, total, err := repo.List(ctx, ChannelEventFilters{})
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if total != 1 || len(events) != 1 || events[0].ID != "recent-1" {
		t.Errorf("expected only recent-1 event, got total=%d len=%d", total, len(events))
	}
}
