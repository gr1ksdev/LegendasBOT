package repositories

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

var ErrInvalidPaymentIntent = errors.New("invalid payment intent")

type PaymentIntentRepository struct {
	db *gorm.DB
}

func NewPaymentIntentRepository(db *gorm.DB) *PaymentIntentRepository {
	return &PaymentIntentRepository{db: db}
}

func (r *PaymentIntentRepository) Create(ctx context.Context, intent *models.PaymentIntent) error {
	return r.db.WithContext(ctx).Create(intent).Error
}

func (r *PaymentIntentRepository) IsPendingForPayment(ctx context.Context, payload string, userID int64, amount int, now time.Time) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.PaymentIntent{}).
		Where("payload = ? AND user_id = ? AND amount_stars = ? AND status = ? AND expires_at > ?", payload, userID, amount, models.PaymentIntentPending, now).
		Count(&count).Error
	return count == 1, err
}

// Process consome uma intent uma unica vez e executa a atualizacao de
// assinatura na mesma transacao. Uma repeticao com o mesmo charge e idempotente.
func (r *PaymentIntentRepository) Process(ctx context.Context, payload string, userID int64, amount int, chargeID string, apply func(*gorm.DB, *models.PaymentIntent) error) (bool, error) {
	processed := false
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var intent models.PaymentIntent
		if err := tx.Where("payload = ?", payload).First(&intent).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrInvalidPaymentIntent
			}
			return err
		}
		if intent.UserID != userID || intent.AmountStars != amount {
			return ErrInvalidPaymentIntent
		}
		if intent.Status == models.PaymentIntentPaid && intent.ChargeID == chargeID {
			return nil
		}
		if intent.Status != models.PaymentIntentPending || !intent.ExpiresAt.After(time.Now()) {
			return ErrInvalidPaymentIntent
		}

		now := time.Now()
		claim := tx.Model(&models.PaymentIntent{}).
			Where("id = ? AND status = ? AND expires_at > ?", intent.ID, models.PaymentIntentPending, now).
			Updates(map[string]interface{}{"status": models.PaymentIntentProcessing, "charge_id": chargeID})
		if claim.Error != nil {
			return claim.Error
		}
		if claim.RowsAffected != 1 {
			return fmt.Errorf("%w: already claimed", ErrInvalidPaymentIntent)
		}

		if err := apply(tx, &intent); err != nil {
			return err
		}
		if err := tx.Model(&models.PaymentIntent{}).Where("id = ? AND status = ?", intent.ID, models.PaymentIntentProcessing).
			Updates(map[string]interface{}{"status": models.PaymentIntentPaid, "paid_at": now}).Error; err != nil {
			return err
		}
		processed = true
		return nil
	})
	return processed, err
}
