package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type PermissionsRepository struct {
	db *gorm.DB
}

func NewPermissionsRepository(db *gorm.DB) *PermissionsRepository {
	return &PermissionsRepository{db: db}
}

func (r *PermissionsRepository) UpdateMessagePermission(ctx context.Context, channelID int64, data interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.MessagePermission{}).
		Where("owner_caption_id = (SELECT caption_id FROM default_captions WHERE owner_channel_id = ?)", channelID).
		Updates(data)

	return result.RowsAffected, result.Error
}

func (r *PermissionsRepository) UpdateButtonsPermission(ctx context.Context, channelID int64, data interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.ButtonsPermission{}).
		Where("owner_caption_id = (SELECT caption_id FROM default_captions WHERE owner_channel_id = ?)", channelID).
		Updates(data)

	return result.RowsAffected, result.Error
}

func (r *PermissionsRepository) UpdateReactionsActive(ctx context.Context, channelID int64, active bool) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.MessagePermission{}).
		Where("owner_caption_id = (SELECT caption_id FROM default_captions WHERE owner_channel_id = ?)", channelID).
		Update("reactions", active)

	return result.RowsAffected, result.Error
}
