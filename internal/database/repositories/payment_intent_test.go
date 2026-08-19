package repositories

import (
	"context"
	"testing"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

func TestPaymentIntentProcessValidatesAndConsumesOnce(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:payment-intent?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.PaymentIntent{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	repo := NewPaymentIntentRepository(db)
	intent := &models.PaymentIntent{ID: "intent-1", Payload: "premium_sub:intent-1", UserID: 10, Type: models.PaymentIntentSubscription, AmountStars: 15, Status: models.PaymentIntentPending, ExpiresAt: time.Now().Add(time.Minute)}
	if err := repo.Create(context.Background(), intent); err != nil {
		t.Fatalf("create intent: %v", err)
	}

	valid, err := repo.IsPendingForPayment(context.Background(), intent.Payload, 10, 15, time.Now())
	if err != nil || !valid {
		t.Fatalf("intent should validate, valid=%v err=%v", valid, err)
	}
	valid, err = repo.IsPendingForPayment(context.Background(), intent.Payload, 10, 14, time.Now())
	if err != nil || valid {
		t.Fatalf("wrong amount must be rejected, valid=%v err=%v", valid, err)
	}

	applied := 0
	processed, err := repo.Process(context.Background(), intent.Payload, 10, 15, "charge-1", func(_ *gorm.DB, _ *models.PaymentIntent) error {
		applied++
		return nil
	})
	if err != nil || !processed || applied != 1 {
		t.Fatalf("first processing = processed:%v applied:%d err:%v", processed, applied, err)
	}
	processed, err = repo.Process(context.Background(), intent.Payload, 10, 15, "charge-1", func(_ *gorm.DB, _ *models.PaymentIntent) error {
		applied++
		return nil
	})
	if err != nil || processed || applied != 1 {
		t.Fatalf("replay = processed:%v applied:%d err:%v", processed, applied, err)
	}
}
