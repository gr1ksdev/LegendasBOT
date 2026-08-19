package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type CustomCaptionRepository struct {
	db *gorm.DB
}

func NewCustomCaptionRepository(db *gorm.DB) *CustomCaptionRepository {
	return &CustomCaptionRepository{db: db}
}

func (r *CustomCaptionRepository) WithTransaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

func (r *CustomCaptionRepository) CreateCustomCaption(ctx context.Context, caption *models.CustomCaption) error {
	return r.db.WithContext(ctx).Create(caption).Error
}

func (r *CustomCaptionRepository) GetCustomCaptionByID(ctx context.Context, channelID int64, captionID string) (*models.CustomCaption, error) {
	var caption models.CustomCaption
	err := r.db.WithContext(ctx).
		Where("caption_id = ? AND owner_channel_id = ?", captionID, channelID).
		First(&caption).Error
	return &caption, err
}

func (r *CustomCaptionRepository) UpdateCustomCaption(ctx context.Context, channelID int64, captionID string, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.CustomCaption{}).
		Where("caption_id = ? AND owner_channel_id = ?", captionID, channelID).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *CustomCaptionRepository) DeleteAllByChannel(ctx context.Context, channelID int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("owner_caption_id IN (SELECT caption_id FROM custom_captions WHERE owner_channel_id = ?)", channelID).
			Delete(&models.CustomCaptionButton{}).Error; err != nil {
			return err
		}
		if err := tx.Where("owner_channel_id = ?", channelID).
			Delete(&models.CustomCaption{}).Error; err != nil {
			return err
		}
		return nil
	})
}

func (r *CustomCaptionRepository) DeleteCustomCaption(ctx context.Context, channelID int64, captionID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("caption_id = ? AND owner_channel_id = ?", captionID, channelID).
		Delete(&models.CustomCaption{})
	return result.RowsAffected, result.Error
}

func (r *CustomCaptionRepository) CreateCustomCaptionButton(ctx context.Context, button *models.CustomCaptionButton) error {
	return r.db.WithContext(ctx).Create(button).Error
}

func (r *CustomCaptionRepository) GetCustomCaptionButtons(ctx context.Context, captionID string) ([]models.CustomCaptionButton, error) {
	var buttons []models.CustomCaptionButton
	err := r.db.WithContext(ctx).
		Where("owner_caption_id = ?", captionID).
		Order("position_y ASC, position_x ASC").
		Find(&buttons).Error
	return buttons, err
}

func (r *CustomCaptionRepository) UpdateCustomCaptionButton(ctx context.Context, captionID, buttonID string, updates map[string]interface{}) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.CustomCaptionButton{}).
		Where("button_id = ? AND owner_caption_id = ?", buttonID, captionID).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *CustomCaptionRepository) DeleteCustomCaptionButton(ctx context.Context, captionID, buttonID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("button_id = ? AND owner_caption_id = ?", buttonID, captionID).
		Delete(&models.CustomCaptionButton{})
	return result.RowsAffected, result.Error
}

func (r *CustomCaptionRepository) UpdateCustomCaptionLayout(ctx context.Context, captionID string, buttons []struct {
	ID string
	X  int
	Y  int
}) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, b := range buttons {
			if err := tx.Model(&models.CustomCaptionButton{}).
				Where("button_id = ? AND owner_caption_id = ?", b.ID, captionID).
				Updates(map[string]interface{}{
					"position_x": b.X,
					"position_y": b.Y,
				}).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
