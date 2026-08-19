package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type UserPostTemplateRepository struct {
	db *gorm.DB
}

func NewUserPostTemplateRepository(db *gorm.DB) *UserPostTemplateRepository {
	return &UserPostTemplateRepository{db: db}
}

func (r *UserPostTemplateRepository) Create(ctx context.Context, template *models.UserPostTemplate) error {
	return r.db.WithContext(ctx).Create(template).Error
}

func (r *UserPostTemplateRepository) GetByID(ctx context.Context, id string) (*models.UserPostTemplate, error) {
	var t models.UserPostTemplate
	if err := r.db.WithContext(ctx).First(&t, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &t, nil
}

func (r *UserPostTemplateRepository) GetByOwner(ctx context.Context, ownerID int64) ([]models.UserPostTemplate, error) {
	var templates []models.UserPostTemplate
	if err := r.db.WithContext(ctx).Where("owner_id = ?", ownerID).Order("created_at desc").Find(&templates).Error; err != nil {
		return nil, err
	}
	return templates, nil
}

func (r *UserPostTemplateRepository) Delete(ctx context.Context, id string, ownerID int64) error {
	return r.db.WithContext(ctx).Where("id = ? AND owner_id = ?", id, ownerID).Delete(&models.UserPostTemplate{}).Error
}

func (r *UserPostTemplateRepository) UpdateName(ctx context.Context, id string, ownerID int64, newName string) error {
	return r.db.WithContext(ctx).Model(&models.UserPostTemplate{}).
		Where("id = ? AND owner_id = ?", id, ownerID).
		Update("name", newName).Error
}
