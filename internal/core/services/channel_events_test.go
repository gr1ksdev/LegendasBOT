package services

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"gorm.io/gorm"
)

func TestChannelEventService_DeleteAll(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&models.ChannelEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := repositories.NewChannelEventRepository(db)
	svc := NewChannelEventService(repo)
	ctx := context.Background()

	// Insert events via Record
	svc.Record(ctx, ChannelEventRecordInput{
		ChannelID: 1001,
		Source:    ChannelEventSourceChannelPost,
		EventType: "post_received",
		Status:    ChannelEventStatusSuccess,
	})
	svc.Record(ctx, ChannelEventRecordInput{
		ChannelID: 1002,
		Source:    ChannelEventSourcePostBuilder,
		EventType: "postbuilder_saved",
		Status:    ChannelEventStatusSuccess,
	})

	// Verify events exist
	list, err := svc.ListAdmin(ctx, ChannelEventListFilters{})
	if err != nil {
		t.Fatalf("failed to list events: %v", err)
	}
	if list.Total != 2 {
		t.Fatalf("expected 2 events, got %d", list.Total)
	}

	// Delete all
	deleted, err := svc.DeleteAll(ctx)
	if err != nil {
		t.Fatalf("failed to delete all: %v", err)
	}
	if deleted != 2 {
		t.Errorf("expected 2 deleted rows, got %d", deleted)
	}

	// Verify list is empty
	listAfter, err := svc.ListAdmin(ctx, ChannelEventListFilters{})
	if err != nil {
		t.Fatalf("failed to list after delete: %v", err)
	}
	if listAfter.Total != 0 {
		t.Errorf("expected 0 events, got %d", listAfter.Total)
	}
}

func TestChannelEventService_CleanupOld(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&models.ChannelEvent{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	repo := repositories.NewChannelEventRepository(db)
	svc := NewChannelEventService(repo)
	ctx := context.Background()

	// Insert old event directly into db
	oldEvent := &models.ChannelEvent{
		ID:        "evt-old",
		ChannelID: 1001,
		Source:    "channel_post",
		EventType: "post_received",
		Status:    "success",
		CreatedAt: time.Now().AddDate(0, 0, -45),
	}
	db.Create(oldEvent)

	// Insert recent event
	svc.Record(ctx, ChannelEventRecordInput{
		ChannelID: 1002,
		Source:    ChannelEventSourceChannelPost,
		EventType: "post_received",
		Status:    ChannelEventStatusSuccess,
	})

	// Cleanup with 30 days retention
	svc.CleanupOld(ctx, 30)

	// Verify only recent remains
	list, err := svc.ListAdmin(ctx, ChannelEventListFilters{})
	if err != nil {
		t.Fatalf("failed to list: %v", err)
	}
	if list.Total != 1 {
		t.Errorf("expected 1 event, got %d", list.Total)
	}
}
