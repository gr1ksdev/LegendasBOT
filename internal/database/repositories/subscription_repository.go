package repositories

import (
	"context"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"gorm.io/gorm"
)

// SubscriptionRepository gerencia o acesso aos dados de assinaturas.
type SubscriptionRepository struct {
	db *gorm.DB
}

// NewSubscriptionRepository cria um novo repositorio de assinaturas.
func NewSubscriptionRepository(db *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// WithTx devolve um repositorio preso a uma transacao do chamador.
func (r *SubscriptionRepository) WithTx(tx *gorm.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: tx}
}

// FindByUserID busca a assinatura de um usuario.
// Retorna (nil, nil) se nao existir.
func (r *SubscriptionRepository) FindByUserID(ctx context.Context, userID int64) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &sub, nil
}

// Create insere uma nova assinatura.
func (r *SubscriptionRepository) Create(ctx context.Context, sub *models.Subscription) error {
	if err := r.db.WithContext(ctx).Create(sub).Error; err != nil {
		return errors.Internal(err)
	}
	return nil
}

// Update atualiza uma assinatura existente.
func (r *SubscriptionRepository) Update(ctx context.Context, sub *models.Subscription) error {
	if err := r.db.WithContext(ctx).Save(sub).Error; err != nil {
		return errors.Internal(err)
	}
	return nil
}

// Delete remove uma assinatura pelo ID.
func (r *SubscriptionRepository) Delete(ctx context.Context, id string) error {
	if err := r.db.WithContext(ctx).Delete(&models.Subscription{ID: id}).Error; err != nil {
		return errors.Internal(err)
	}
	return nil
}

// FindExpired busca assinaturas com periodo expirado que ainda estao como active.
func (r *SubscriptionRepository) FindExpired(ctx context.Context) ([]models.Subscription, error) {
	var subs []models.Subscription
	err := r.db.WithContext(ctx).
		Where("status = ? AND current_period_end < ?", models.SubscriptionActive, time.Now()).
		Find(&subs).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return subs, nil
}

// FindAll retorna todas as assinaturas ordenadas por criacao (mais recentes primeiro).
func (r *SubscriptionRepository) FindAll(ctx context.Context) ([]models.Subscription, error) {
	var subs []models.Subscription
	err := r.db.WithContext(ctx).Order("created_at desc").Find(&subs).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return subs, nil
}

// FindByChargeID busca uma subscription pelo TelegramPaymentID (charge_id).
func (r *SubscriptionRepository) FindByChargeID(ctx context.Context, chargeID string) (*models.Subscription, error) {
	var sub models.Subscription
	err := r.db.WithContext(ctx).Where("telegram_payment_id = ?", chargeID).First(&sub).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &sub, nil
}

// FindRefundByChargeID busca um refund pelo TelegramPaymentChargeID.
func (r *SubscriptionRepository) FindRefundByChargeID(ctx context.Context, chargeID string) (*models.Refund, error) {
	var refund models.Refund
	err := r.db.WithContext(ctx).Where("telegram_payment_charge_id = ?", chargeID).First(&refund).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &refund, nil
}

// CreateRefund insere um novo registro de reembolso.
func (r *SubscriptionRepository) CreateRefund(ctx context.Context, refund *models.Refund) error {
	if err := r.db.WithContext(ctx).Create(refund).Error; err != nil {
		return errors.Internal(err)
	}
	return nil
}

// FindRefundsByUserID busca todos os reembolsos de um usuario.
func (r *SubscriptionRepository) FindRefundsByUserID(ctx context.Context, userID int64) ([]models.Refund, error) {
	var refunds []models.Refund
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).Order("created_at desc").Find(&refunds).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return refunds, nil
}

// FindDueForRenewal busca assinaturas ativas que estao proximas do fim (7 dias ou menos).
func (r *SubscriptionRepository) FindDueForRenewal(ctx context.Context) ([]models.Subscription, error) {
	var subs []models.Subscription
	cutoff := time.Now().AddDate(0, 0, 7)
	err := r.db.WithContext(ctx).
		Where("status = ? AND current_period_end <= ? AND cancel_at_period_end = ?",
			models.SubscriptionActive, cutoff, false).
		Find(&subs).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return subs, nil
}
