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

func TestExpiredSubscriptionDoesNotGrantPremiumAccess(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file:subscription-expiry?mode=memory&cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.AutoMigrate(&models.User{}, &models.Subscription{}, &models.PaymentIntent{}, &models.PremiumFeature{}); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	user := &models.User{UserId: 42, FirstName: "Premium", Features: `{"managedPremiumAccount":true}`}
	if err := db.Create(user).Error; err != nil {
		t.Fatalf("create user: %v", err)
	}
	if err := db.Create(&models.Subscription{ID: "expired", UserID: 42, Status: models.SubscriptionActive, CurrentPeriodStart: time.Now().AddDate(0, 0, -31), CurrentPeriodEnd: time.Now().Add(-time.Minute)}).Error; err != nil {
		t.Fatalf("create subscription: %v", err)
	}
	if err := db.Create(&models.PremiumFeature{Key: "managed_premium_account", Name: "Managed", Enabled: true}).Error; err != nil {
		t.Fatalf("create premium feature: %v", err)
	}

	featureSvc := NewPremiumFeatureService(repositories.NewPremiumFeatureRepository(db))
	svc := NewSubscriptionService(repositories.NewSubscriptionRepository(db), repositories.NewPaymentIntentRepository(db), repositories.NewUserRepository(db), nil, featureSvc)
	status, err := svc.GetStatus(context.Background(), 42)
	if err != nil {
		t.Fatalf("get status: %v", err)
	}
	if status.HasSubscription {
		t.Fatal("expired subscription must not grant premium access before maintenance runs")
	}
	if svc.UserHasFeature(context.Background(), 42, "managed_premium_account") {
		t.Fatal("stale feature projection must not grant access after expiration")
	}
	if err := svc.ExpireSubscriptions(context.Background()); err != nil {
		t.Fatalf("expire subscriptions: %v", err)
	}
	sub, err := repositories.NewSubscriptionRepository(db).FindByUserID(context.Background(), 42)
	if err != nil || sub.Status != models.SubscriptionExpired {
		t.Fatalf("subscription status after expiration = %+v, err=%v", sub, err)
	}
}

func TestCreateInvoiceRejectsUnboundedChannelCount(t *testing.T) {
	svc := &SubscriptionService{}
	if _, err := svc.CreateInvoice(context.Background(), 1, false, maxPremiumChannels+1); err == nil {
		t.Fatal("expected excessive channel count to be rejected before pricing")
	}
}
