package repositories

import (
	"context"
	"errors"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// EmojiRepository gerencia acesso aos dados de emoji customizado.
type EmojiRepository struct {
	db *gorm.DB
}

func NewEmojiRepository(db *gorm.DB) *EmojiRepository {
	return &EmojiRepository{db: db}
}

// ── CustomEmoji ──

// GetEmoji busca um emoji pelo ID do Telegram.
func (r *EmojiRepository) GetEmoji(ctx context.Context, emojiID string) (*models.CustomEmoji, error) {
	var emoji models.CustomEmoji
	err := r.db.WithContext(ctx).Where("emoji_id = ?", emojiID).First(&emoji).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &emoji, nil
}

// UpsertEmoji insere ou atualiza o arquivo do emoji.
func (r *EmojiRepository) UpsertEmoji(ctx context.Context, emoji *models.CustomEmoji) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "emoji_id"}},
			DoUpdates: clause.AssignmentColumns([]string{"file_data", "file_type", "updated_at"}),
		}).
		Create(emoji).Error
}

// DeleteEmoji remove um emoji do cache.
func (r *EmojiRepository) DeleteEmoji(ctx context.Context, emojiID string) error {
	return r.db.WithContext(ctx).Delete(&models.CustomEmoji{EmojiID: emojiID}).Error
}

// ── UserEmojiAccess ──

// GrantAccess concede acesso de um usuário a um emoji.
func (r *EmojiRepository) GrantAccess(ctx context.Context, userID int64, emojiID string) error {
	access := models.UserEmojiAccess{
		UserID:  userID,
		EmojiID: emojiID,
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "emoji_id"}},
			DoNothing: true, // se já existe, ignora
		}).
		Create(&access).Error
}

// HasAccess verifica se um usuário tem acesso a um emoji.
func (r *EmojiRepository) HasAccess(ctx context.Context, userID int64, emojiID string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).
		Model(&models.UserEmojiAccess{}).
		Where("user_id = ? AND emoji_id = ?", userID, emojiID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// RevokeAccess remove acesso de um usuário a um emoji.
func (r *EmojiRepository) RevokeAccess(ctx context.Context, userID int64, emojiID string) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND emoji_id = ?", userID, emojiID).
		Delete(&models.UserEmojiAccess{}).Error
}

// RevokeAllAccess remove todo acesso de um usuário (útil se desativar conta).
func (r *EmojiRepository) RevokeAllAccess(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Delete(&models.UserEmojiAccess{}).Error
}

// GetAccessedEmojiIDs retorna todos os emoji-ids que um usuário tem acesso.
func (r *EmojiRepository) GetAccessedEmojiIDs(ctx context.Context, userID int64) ([]string, error) {
	var ids []string
	err := r.db.WithContext(ctx).
		Model(&models.UserEmojiAccess{}).
		Where("user_id = ?", userID).
		Pluck("emoji_id", &ids).Error
	if err != nil {
		return nil, err
	}
	return ids, nil
}
