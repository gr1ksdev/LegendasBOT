package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type UserCaptionTemplateRepository struct {
	db *gorm.DB
}

func NewUserCaptionTemplateRepository(db *gorm.DB) *UserCaptionTemplateRepository {
	return &UserCaptionTemplateRepository{db: db}
}

func (r *UserCaptionTemplateRepository) Create(ctx context.Context, tpl *models.UserCaptionTemplate) error {
	return r.db.WithContext(ctx).Create(tpl).Error
}

func (r *UserCaptionTemplateRepository) GetByID(ctx context.Context, id string) (*models.UserCaptionTemplate, error) {
	var tpl models.UserCaptionTemplate
	err := r.db.WithContext(ctx).
		Preload("Buttons").
		Where("id = ?", id).
		First(&tpl).Error
	return &tpl, err
}

func (r *UserCaptionTemplateRepository) GetByUserAndCode(ctx context.Context, userID int64, code string) (*models.UserCaptionTemplate, error) {
	var tpl models.UserCaptionTemplate
	err := r.db.WithContext(ctx).
		Preload("Buttons").
		Where("user_id = ? AND code = ?", userID, code).
		First(&tpl).Error
	return &tpl, err
}

func (r *UserCaptionTemplateRepository) ListByUser(ctx context.Context, userID int64) ([]models.UserCaptionTemplate, error) {
	var templates []models.UserCaptionTemplate
	err := r.db.WithContext(ctx).
		Preload("Buttons").
		Where("user_id = ?", userID).
		Order("created_at DESC").
		Find(&templates).Error
	return templates, err
}

func (r *UserCaptionTemplateRepository) Update(ctx context.Context, id string, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&models.UserCaptionTemplate{}).
		Where("id = ?", id).
		Updates(updates).Error
}

func (r *UserCaptionTemplateRepository) Delete(ctx context.Context, id string, userID int64) error {
	return r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&models.UserCaptionTemplate{}).Error
}

func (r *UserCaptionTemplateRepository) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}
