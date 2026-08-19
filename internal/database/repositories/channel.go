package repositories

import (
	"context"
	"errors"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type ChannelRepository struct {
	db *gorm.DB
}

func NewChannelRepository(db *gorm.DB) *ChannelRepository {
	return &ChannelRepository{db: db}
}

func (r *ChannelRepository) CountUserChannels(ctx context.Context, userID int64) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.Channel{}).
		Where("owner_id = ?", userID).
		Count(&count).Error
	return count, err
}

func (r *ChannelRepository) GetChannelByTwoID(ctx context.Context, userId, channelId int64) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.WithContext(ctx).
		Joins("DefaultCaption").
		Joins("DefaultCaption.MessagePermission").
		Joins("DefaultCaption.ButtonsPermission").
		Joins("Separator").
		Preload("Buttons").
		Preload("CustomCaptions").
		Preload("CustomCaptions.Buttons").
		Where("channels.owner_id = ? AND channels.id = ?", userId, channelId).
		First(&channel).Error

	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *ChannelRepository) GetChannelByUserID(ctx context.Context, userId int64) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.WithContext(ctx).
		Joins("DefaultCaption").
		Joins("DefaultCaption.MessagePermission").
		Joins("DefaultCaption.ButtonsPermission").
		Joins("Separator").
		Preload("Buttons").
		Preload("CustomCaptions").
		Preload("CustomCaptions.Buttons").
		Where("channels.owner_id = ?", userId).
		First(&channel).Error

	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *ChannelRepository) GetChannelByID(ctx context.Context, channelId int64) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.WithContext(ctx).
		Joins("DefaultCaption").
		Joins("DefaultCaption.MessagePermission").
		Joins("DefaultCaption.ButtonsPermission").
		Joins("Separator").
		Preload("Owner").
		Preload("Owner.Channels").
		Preload("Buttons").
		Preload("CustomCaptions").
		Preload("CustomCaptions.Buttons").
		Where("channels.id = ?", channelId).
		First(&channel).Error

	if err != nil {
		return nil, err
	}
	return &channel, nil
}

func (r *ChannelRepository) GetChannelByIDLight(ctx context.Context, channelId int64) (*models.Channel, error) {
	var channel models.Channel
	err := r.db.WithContext(ctx).
		Where("id = ?", channelId).
		First(&channel).Error
	return &channel, err
}

func (r *ChannelRepository) CreateChannel(ctx context.Context, channel *models.Channel) error {
	return r.db.WithContext(ctx).Create(channel).Error
}

func (r *ChannelRepository) UpdateChannel(ctx context.Context, channel *models.Channel) error {
	return r.db.WithContext(ctx).Save(channel).Error
}

func (r *ChannelRepository) UpdateOwnerChannel(ctx context.Context, channelID, oldOwnerID, newOwnerID int64) error {
	var channel models.Channel
	err := r.db.WithContext(ctx).
		Where("id = ? AND owner_id = ?", channelID, oldOwnerID).
		First(&channel).Error
	if err != nil {
		return err
	}

	return r.db.WithContext(ctx).
		Model(&channel).
		Updates(map[string]any{
			"owner_id":      newOwnerID,
			"token_version": gorm.Expr("token_version + 1"),
		}).Error
}

func (r *ChannelRepository) DeleteChannelWithRelations(ctx context.Context, userId, channelId int64) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// 1. Verificar se o canal existe e pertence ao usuário (ou se é admin bypassando userId se necessário)
		// Aqui mantemos a lógica original de filtragem por userId e channelId
		var channel models.Channel
		if err := tx.Where("owner_id = ? AND id = ?", userId, channelId).First(&channel).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("channel not found")
			}
			return err
		}

		// 2. Limpar dados dependentes manualmente para evitar violações de FK no Postgres
		// GORM OnDelete:CASCADE nem sempre é refletido no banco se as tabelas já existiam.

		// Limpar Botões
		if err := tx.Where("owner_channel_id = ?", channelId).Delete(&models.Button{}).Error; err != nil {
			return err
		}

		// Limpar Separadores
		if err := tx.Where("owner_channel_id = ?", channelId).Delete(&models.Separator{}).Error; err != nil {
			return err
		}

		// Limpar Custom Captions e seus botões
		var customCaptions []models.CustomCaption
		if err := tx.Where("owner_channel_id = ?", channelId).Find(&customCaptions).Error; err == nil {
			for _, cc := range customCaptions {
				if err := tx.Where("custom_caption_id = ?", cc.CaptionID).Delete(&models.CustomCaptionButton{}).Error; err != nil {
					return err
				}
			}
			if err := tx.Where("owner_channel_id = ?", channelId).Delete(&models.CustomCaption{}).Error; err != nil {
				return err
			}
		}

		// Limpar Default Caption e suas permissões
		var defaultCaption models.DefaultCaption
		if err := tx.Where("owner_channel_id = ?", channelId).First(&defaultCaption).Error; err == nil {
			if err := tx.Where("owner_caption_id = ?", defaultCaption.CaptionID).Delete(&models.MessagePermission{}).Error; err != nil {
				return err
			}
			if err := tx.Where("owner_caption_id = ?", defaultCaption.CaptionID).Delete(&models.ButtonsPermission{}).Error; err != nil {
				return err
			}
			if err := tx.Delete(&defaultCaption).Error; err != nil {
				return err
			}
		}

		// 3. Por fim, deletar o canal
		if err := tx.Delete(&channel).Error; err != nil {
			return err
		}

		return nil
	})
}

func (r *ChannelRepository) GetAllChannelsByUserID(ctx context.Context, userID int64) ([]models.Channel, error) {
	var channels []models.Channel
	err := r.db.WithContext(ctx).
		Where("owner_id = ?", userID).
		Order("updated_at ASC").
		Find(&channels).Error
	return channels, err
}

func (r *ChannelRepository) GetAllChannels(ctx context.Context) ([]models.Channel, error) {
	var channels []models.Channel
	err := r.db.WithContext(ctx).
		Order("updated_at DESC").
		Find(&channels).Error
	return channels, err
}

func (r *ChannelRepository) GetAllChannelsPaginated(ctx context.Context, limit, offset int) ([]models.Channel, int64, error) {
	var channels []models.Channel
	var total int64
	db := r.db.WithContext(ctx).Model(&models.Channel{})
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := db.Limit(limit).Offset(offset).Order("updated_at DESC").Find(&channels).Error
	return channels, total, err
}

func (r *ChannelRepository) GetChannelButtons(ctx context.Context, channelId int64) ([]models.Button, error) {
	var buttons []models.Button
	err := r.db.WithContext(ctx).
		Where("owner_channel_id = ?", channelId).
		Order("position_y ASC, position_x ASC").
		Find(&buttons).Error
	return buttons, err
}

func (r *ChannelRepository) UpdateDefaultCaption(ctx context.Context, channelID int64, caption string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.DefaultCaption{}).
		Where("owner_channel_id = ?", channelID).
		Updates(map[string]interface{}{
			"caption":      caption,
			"entities":     "",    // limpa entities ao salvar via dashboard
			"use_entities": false, // desabilita path de entities
		})
	return result.RowsAffected, result.Error
}

func (r *ChannelRepository) UpdateDefaultCaptionEntities(ctx context.Context, channelID int64, caption string, entities string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.DefaultCaption{}).
		Where("owner_channel_id = ?", channelID).
		Updates(map[string]interface{}{
			"caption":      caption,
			"entities":     entities,
			"use_entities": true,
		})
	return result.RowsAffected, result.Error
}

func (r *ChannelRepository) UpdateNewPackSettings(ctx context.Context, channelID int64, caption string, messageButtons, stickerButtons *bool, messagePosition *string, replyToSticker *bool) (int64, error) {
	updates := map[string]interface{}{
		"new_pack_caption": caption,
	}
	if messageButtons != nil {
		updates["new_pack_message_buttons"] = *messageButtons
	}
	if stickerButtons != nil {
		updates["new_pack_sticker_buttons"] = *stickerButtons
	}
	if messagePosition != nil {
		updates["new_pack_message_position"] = *messagePosition
	}
	if replyToSticker != nil {
		updates["new_pack_reply_to_sticker"] = *replyToSticker
	}

	result := r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Updates(updates)
	return result.RowsAffected, result.Error
}

func (r *ChannelRepository) UpdateReactions(ctx context.Context, channelID int64, reactions string) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Update("reactions", reactions)
	return result.RowsAffected, result.Error
}

func (r *ChannelRepository) UpdateReactionPosition(ctx context.Context, channelID int64, position int) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Update("reaction_position", position)
	return result.RowsAffected, result.Error
}

func (r *ChannelRepository) UpdateDynamicLinks(ctx context.Context, channelID int64, settings map[string]any) (int64, error) {
	result := r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Updates(settings)
	return result.RowsAffected, result.Error
}

func (r *ChannelRepository) UpdateNativeReactions(ctx context.Context, channelID int64, emojis string) error {
	return r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Update("native_reactions", emojis).Error
}

func (r *ChannelRepository) UpdateNativeReactionMode(ctx context.Context, channelID int64, mode string) error {
	return r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Update("native_reaction_mode", mode).Error
}

func (r *ChannelRepository) UpdateNativeReactionsEnabled(ctx context.Context, channelID int64, enabled bool) error {
	return r.db.WithContext(ctx).Model(&models.Channel{}).
		Where("id = ?", channelID).
		Update("native_reactions_enabled", enabled).Error
}
