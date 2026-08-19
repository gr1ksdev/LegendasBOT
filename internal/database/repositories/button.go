package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type ButtonRepository struct {
	db *gorm.DB
}

func NewButtonRepository(db *gorm.DB) *ButtonRepository {
	return &ButtonRepository{db: db}
}

func (r *ButtonRepository) CreateButton(ctx context.Context, button *models.Button) error {
	return r.db.WithContext(ctx).Create(button).Error
}

func (r *ButtonRepository) UpdateButton(ctx context.Context, channelID int64, buttonID, name, url, style string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Button{}).
		Where("button_id = ? AND owner_channel_id = ?", buttonID, channelID).
		Updates(map[string]interface{}{
			"name_button": name,
			"button_url":  url,
			"style":       style,
		})
	return result.RowsAffected, result.Error
}

func (r *ButtonRepository) DeleteButton(ctx context.Context, channelID int64, buttonID string) (int64, error) {
	result := r.db.WithContext(ctx).
		Where("button_id = ? AND owner_channel_id = ?", buttonID, channelID).
		Delete(&models.Button{})
	return result.RowsAffected, result.Error
}

func (r *ButtonRepository) UpdateButtonsLayout(ctx context.Context, channelID int64, buttons []struct {
	ID string
	X  int
	Y  int
}, reactionPosition int) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, b := range buttons {
			if err := tx.Model(&models.Button{}).
				Where("button_id = ? AND owner_channel_id = ?", b.ID, channelID).
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

func (r *ButtonRepository) IsRowOccupiedByButtons(ctx context.Context, channelID int64, y int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.Button{}).
		Where("owner_channel_id = ? AND position_y = ?", channelID, y).
		Count(&count).Error
	return count > 0, err
}
