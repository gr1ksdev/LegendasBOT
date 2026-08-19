package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"gorm.io/gorm"
)

// PremiumFeatureRepository gerencia o acesso aos dados de features premium.
type PremiumFeatureRepository struct {
	db *gorm.DB
}

// NewPremiumFeatureRepository cria um novo repositorio de features premium.
func NewPremiumFeatureRepository(db *gorm.DB) *PremiumFeatureRepository {
	return &PremiumFeatureRepository{db: db}
}

// List retorna todas as features premium cadastradas.
func (r *PremiumFeatureRepository) List(ctx context.Context) ([]models.PremiumFeature, error) {
	var features []models.PremiumFeature
	err := r.db.WithContext(ctx).Order("created_at ASC").Find(&features).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return features, nil
}

// ListEnabled retorna apenas as features que estao habilitadas globalmente.
func (r *PremiumFeatureRepository) ListEnabled(ctx context.Context) ([]models.PremiumFeature, error) {
	var features []models.PremiumFeature
	err := r.db.WithContext(ctx).Where("enabled = ?", true).Order("created_at ASC").Find(&features).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return features, nil
}

// GetByKey busca uma feature pela chave.
func (r *PremiumFeatureRepository) GetByKey(ctx context.Context, key string) (*models.PremiumFeature, error) {
	var feature models.PremiumFeature
	err := r.db.WithContext(ctx).Where("key = ?", key).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &feature, nil
}

// Upsert cria ou atualiza uma feature premium.
func (r *PremiumFeatureRepository) Upsert(ctx context.Context, feature *models.PremiumFeature) error {
	return r.db.WithContext(ctx).Save(feature).Error
}

// Toggle alterna o estado enabled de uma feature.
func (r *PremiumFeatureRepository) Toggle(ctx context.Context, key string, enabled bool) error {
	result := r.db.WithContext(ctx).
		Model(&models.PremiumFeature{}).
		Where("key = ?", key).
		Update("enabled", enabled)
	if result.Error != nil {
		return errors.Internal(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}

// IsEnabled verifica rapidamente se uma feature esta habilitada.
func (r *PremiumFeatureRepository) IsEnabled(ctx context.Context, key string) (bool, error) {
	var feature models.PremiumFeature
	err := r.db.WithContext(ctx).Select("enabled").Where("key = ?", key).First(&feature).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return false, nil
		}
		return false, errors.Internal(err)
	}
	return feature.Enabled, nil
}

// UpdatePrice atualiza o preco de uma feature.
func (r *PremiumFeatureRepository) UpdatePrice(ctx context.Context, key string, price int) error {
	result := r.db.WithContext(ctx).
		Model(&models.PremiumFeature{}).
		Where("key = ?", key).
		Update("price", price)
	if result.Error != nil {
		return errors.Internal(result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.ErrNotFound
	}
	return nil
}
