package services

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	apperrors "github.com/leirbagxis/FreddyBot/pkg/errors"
	"gorm.io/gorm"
)

func setupChannelSecurityService(t *testing.T) (*ChannelService, *repositories.ChannelRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:channel-security?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Channel{}, &models.ScheduledPost{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	repo := repositories.NewChannelRepository(db)
	return NewChannelService(repo, nil, nil, nil, nil), repo, db
}

func TestTransferChannelRejectsNonOwner(t *testing.T) {
	svc, repo, db := setupChannelSecurityService(t)
	ctx := context.Background()
	for _, userID := range []int64{10, 20, 30} {
		if err := db.Create(&models.User{UserId: userID, FirstName: "User"}).Error; err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	if err := repo.CreateChannel(ctx, &models.Channel{ID: 100, OwnerID: 10, Title: "Private"}); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	if _, err := svc.TransferChannel(ctx, 20, 100, 30, false); err != apperrors.ErrForbidden {
		t.Fatalf("expected forbidden transfer, got %v", err)
	}

	channel, err := repo.GetChannelByIDLight(ctx, 100)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel.OwnerID != 10 {
		t.Fatalf("channel owner changed after forbidden transfer: got %d", channel.OwnerID)
	}
}

func TestTransferChannelAllowsCurrentOwner(t *testing.T) {
	svc, repo, db := setupChannelSecurityService(t)
	ctx := context.Background()
	for _, userID := range []int64{11, 31} {
		if err := db.Create(&models.User{UserId: userID, FirstName: "User"}).Error; err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	if err := repo.CreateChannel(ctx, &models.Channel{ID: 101, OwnerID: 11, Title: "Owned"}); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	if _, err := svc.TransferChannel(ctx, 11, 101, 31, false); err != nil {
		t.Fatalf("transfer by owner: %v", err)
	}

	channel, err := repo.GetChannelByIDLight(ctx, 101)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel.OwnerID != 31 {
		t.Fatalf("expected new owner 31, got %d", channel.OwnerID)
	}
}

func TestTransferChannelAllowsPrivilegedActor(t *testing.T) {
	svc, repo, db := setupChannelSecurityService(t)
	ctx := context.Background()
	for _, userID := range []int64{12, 22, 32} {
		if err := db.Create(&models.User{UserId: userID, FirstName: "User"}).Error; err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	if err := repo.CreateChannel(ctx, &models.Channel{ID: 102, OwnerID: 12, Title: "Managed"}); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	if _, err := svc.TransferChannel(ctx, 22, 102, 32, true); err != nil {
		t.Fatalf("transfer by privileged actor: %v", err)
	}

	channel, err := repo.GetChannelByIDLight(ctx, 102)
	if err != nil {
		t.Fatalf("get channel: %v", err)
	}
	if channel.OwnerID != 32 {
		t.Fatalf("expected new owner 32, got %d", channel.OwnerID)
	}
}

func TestCreateScheduledPostRejectsForeignChannel(t *testing.T) {
	_, channelRepo, db := setupChannelSecurityService(t)
	ctx := context.Background()
	for _, userID := range []int64{40, 50} {
		if err := db.Create(&models.User{UserId: userID, FirstName: "User"}).Error; err != nil {
			t.Fatalf("create user %d: %v", userID, err)
		}
	}
	if err := channelRepo.CreateChannel(ctx, &models.Channel{ID: 400, OwnerID: 40, Title: "Foreign"}); err != nil {
		t.Fatalf("create channel: %v", err)
	}

	scheduler := NewSchedulerService(repositories.NewScheduledPostRepository(db), channelRepo, nil, nil, nil)
	if _, err := scheduler.CreateScheduledPost(ctx, 50, 400, `{}`, ScheduleOptions{ScheduleType: "once"}); err != apperrors.ErrForbidden {
		t.Fatalf("expected forbidden schedule creation, got %v", err)
	}

	var count int64
	if err := db.Model(&models.ScheduledPost{}).Count(&count).Error; err != nil {
		t.Fatalf("count schedules: %v", err)
	}
	if count != 0 {
		t.Fatalf("foreign channel schedule was created: %d records", count)
	}
}

func TestScheduleTimeValidationRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"99:99", "12:60", "1:2", "invalid"} {
		if _, _, err := validateScheduleTime(value); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}

func TestUpdateScheduleTimeRecalculatesRecurringPost(t *testing.T) {
	_, channelRepo, db := setupChannelSecurityService(t)
	ctx := context.Background()
	if err := db.Create(&models.User{UserId: 70, FirstName: "Owner"}).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := channelRepo.CreateChannel(ctx, &models.Channel{ID: 700, OwnerID: 70, Title: "Owned"}); err != nil {
		t.Fatalf("create channel: %v", err)
	}
	post := &models.ScheduledPost{ID: "daily-time", OwnerID: 70, ChannelID: 700, ScheduleType: "daily", ScheduleTime: "08:00", NextRunAt: time.Now().Add(24 * time.Hour), Status: "pending"}
	if err := db.Create(post).Error; err != nil {
		t.Fatalf("create schedule: %v", err)
	}

	scheduler := NewSchedulerService(repositories.NewScheduledPostRepository(db), channelRepo, nil, nil, nil)
	if err := scheduler.UpdateScheduleTime(ctx, post.ID, 70, nil, "23:59"); err != nil {
		t.Fatalf("update schedule time: %v", err)
	}

	var updated models.ScheduledPost
	if err := db.First(&updated, "id = ?", post.ID).Error; err != nil {
		t.Fatalf("read schedule: %v", err)
	}
	if updated.NextRunAt.In(utils.BrazilTZ()).Format("15:04") != "23:59" {
		t.Fatalf("schedule was not recalculated for the requested time: %s", updated.NextRunAt.Format(time.RFC3339))
	}
}
