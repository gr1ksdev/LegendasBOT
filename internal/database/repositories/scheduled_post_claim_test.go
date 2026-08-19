package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

func TestClaimDuePostsClaimsEachPostOnlyOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:scheduled-claim?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.ScheduledPost{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	repo := NewScheduledPostRepository(db)
	now := time.Now()
	post := &models.ScheduledPost{ID: "claim-once", OwnerID: 1, ChannelID: 2, PostData: `{}`, ScheduleType: "once", NextRunAt: now.Add(-time.Minute), Status: "pending"}
	if err := repo.Create(context.Background(), post); err != nil {
		t.Fatalf("create post: %v", err)
	}

	first, err := repo.ClaimDuePosts(context.Background(), now)
	if err != nil {
		t.Fatalf("first claim: %v", err)
	}
	second, err := repo.ClaimDuePosts(context.Background(), now)
	if err != nil {
		t.Fatalf("second claim: %v", err)
	}
	if len(first) != 1 || len(second) != 0 {
		t.Fatalf("claims = %d then %d, want 1 then 0", len(first), len(second))
	}

	if err := repo.RecoverStaleClaims(context.Background(), now.Add(time.Minute)); err != nil {
		t.Fatalf("recover stale claim: %v", err)
	}
	recovered, err := repo.ClaimDuePosts(context.Background(), now.Add(time.Minute))
	if err != nil {
		t.Fatalf("claim recovered post: %v", err)
	}
	if len(recovered) != 1 {
		t.Fatalf("recovered claims = %d, want 1", len(recovered))
	}
}
