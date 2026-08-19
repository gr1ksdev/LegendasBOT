package repositories

import (
	"context"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type AutoDeleteRepository struct {
	db *gorm.DB
}

func NewAutoDeleteRepository(db *gorm.DB) *AutoDeleteRepository {
	return &AutoDeleteRepository{db: db}
}

func (r *AutoDeleteRepository) Create(ctx context.Context, item *models.AutoDeletePost) error {
	return r.db.WithContext(ctx).Create(item).Error
}

func (r *AutoDeleteRepository) GetDuePosts(ctx context.Context, now time.Time) ([]models.AutoDeletePost, error) {
	var posts []models.AutoDeletePost
	err := r.db.WithContext(ctx).
		Where("status = 'pending' AND delete_at <= ?", now).
		Order("delete_at ASC").
		Limit(50).
		Find(&posts).Error
	return posts, err
}

func (r *AutoDeleteRepository) MarkDeleted(ctx context.Context, id uint, deletedAt time.Time) error {
	return r.db.WithContext(ctx).
		Model(&models.AutoDeletePost{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "deleted",
			"deleted_at": deletedAt,
		}).Error
}

func (r *AutoDeleteRepository) MarkFailed(ctx context.Context, id uint, lastError string) error {
	return r.db.WithContext(ctx).
		Model(&models.AutoDeletePost{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     "failed",
			"last_error": lastError,
		}).Error
}
