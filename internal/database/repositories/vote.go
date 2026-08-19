package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type VoteRepository struct {
	db *gorm.DB
}

func NewVoteRepository(db *gorm.DB) *VoteRepository {
	return &VoteRepository{db: db}
}

// ToggleVote adds a vote if it doesn't exist, or removes it if it does.
// Only one emoji can be voted per user per message.
// Returns true if added, false if removed, and error if any.
func (r *VoteRepository) ToggleVote(ctx context.Context, chatID int64, messageID int, inlineMessageID string, userID int64, emoji string) (bool, int64, error) {
	var added bool
	var count int64

	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existingVote models.Vote

		// 1. Procurar qualquer voto deste usuário nesta mensagem
		query := tx.Where("user_id = ?", userID)
		if inlineMessageID != "" {
			query = query.Where("inline_message_id = ?", inlineMessageID)
		} else {
			query = query.Where("chat_id = ? AND message_id = ?", chatID, messageID)
		}

		err := query.First(&existingVote).Error

		if err == nil {
			// Usuário já tem um voto nesta mensagem
			if existingVote.Emoji == emoji {
				// Clicou no mesmo emoji -> Remover (Toggle OFF)
				if err := tx.Delete(&existingVote).Error; err != nil {
					return err
				}
				added = false
			} else {
				// Clicou em emoji diferente -> Atualizar (Transferência)
				existingVote.Emoji = emoji
				if err := tx.Save(&existingVote).Error; err != nil {
					return err
				}
				added = true
			}
		} else if err == gorm.ErrRecordNotFound {
			// Não tem voto ainda -> Criar novo
			newVote := models.Vote{
				ChatID:          chatID,
				MessageID:       messageID,
				InlineMessageID: inlineMessageID,
				UserID:          userID,
				Emoji:           emoji,
			}
			if err := tx.Create(&newVote).Error; err != nil {
				return err
			}
			added = true
		} else {
			return err
		}

		// 2. Contar votos para o emoji alvo
		countQuery := tx.Model(&models.Vote{}).Where("emoji = ?", emoji)
		if inlineMessageID != "" {
			countQuery = countQuery.Where("inline_message_id = ?", inlineMessageID)
		} else {
			countQuery = countQuery.Where("chat_id = ? AND message_id = ?", chatID, messageID)
		}

		if err := countQuery.Count(&count).Error; err != nil {
			return err
		}

		return nil
	})

	return added, count, err
}

func (r *VoteRepository) GetVoteCounts(ctx context.Context, chatID int64, messageID int, inlineMessageID string) (map[string]int64, error) {
	type Result struct {
		Emoji string
		Count int64
	}
	var results []Result
	query := r.db.WithContext(ctx).Model(&models.Vote{}).Select("emoji, count(*) as count")

	if inlineMessageID != "" {
		query = query.Where("inline_message_id = ?", inlineMessageID)
	} else {
		query = query.Where("chat_id = ? AND message_id = ?", chatID, messageID)
	}

	err := query.Group("emoji").Scan(&results).Error

	if err != nil {
		return nil, err
	}

	counts := make(map[string]int64)
	for _, res := range results {
		counts[res.Emoji] = res.Count
	}
	return counts, nil
}
