package services

import (
	"context"
	"strings"
	"unicode"

	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

type CaptionService struct {
	channelRepo *repositories.ChannelRepository
	buttonRepo  *repositories.ButtonRepository
	cache       *cache.Service
}

func NewCaptionService(channelRepo *repositories.ChannelRepository, buttonRepo *repositories.ButtonRepository, cache *cache.Service) *CaptionService {
	return &CaptionService{
		channelRepo: channelRepo,
		buttonRepo:  buttonRepo,
		cache:       cache,
	}
}

func (s *CaptionService) isEmoji(s_str string) bool {
	for _, r := range s_str {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

func (s *CaptionService) UpdateDefaultCaption(ctx context.Context, channelID int64, captionData types.CaptionDefaultUpdateRequest) (int64, error) {
	if len(captionData.Caption) > 4096 {
		return 0, errors.BadRequest("Caption muito longa (máximo 4096 caracteres)")
	}

	rowsAffected, err := s.channelRepo.UpdateDefaultCaption(ctx, channelID, captionData.Caption)
	if err != nil {
		return 0, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	logger.Bot("✅ Legenda padrão atualizada com sucesso (Canal: %d)", channelID)

	return rowsAffected, nil
}

func (s *CaptionService) SaveEntitiesCaption(ctx context.Context, channelID int64, text string, entitiesJSON string) error {
	if len(text) > 4096 {
		return errors.BadRequest("Caption muito longa (máximo 4096 caracteres)")
	}
	_, err := s.channelRepo.UpdateDefaultCaptionEntities(ctx, channelID, text, entitiesJSON)
	if err != nil {
		return errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channelID)
	logger.Bot("✅ Legenda com entities salva (Canal: %d, entities: %d bytes)", channelID, len(entitiesJSON))
	return nil
}

func (s *CaptionService) UpdateNewPackCaption(ctx context.Context, channelID int64, captionData types.NewPackCaptionUpdateRequest) (int64, error) {
	caption := captionData.Text()
	if strings.TrimSpace(caption) == "" {
		return 0, errors.BadRequest("Caption obrigatória")
	}
	if len(caption) > 4096 {
		return 0, errors.BadRequest("Caption muito longa (máximo 4096 caracteres)")
	}

	if captionData.NewPackMessagePosition != nil {
		position := strings.TrimSpace(*captionData.NewPackMessagePosition)
		if position != "above" && position != "below" {
			return 0, errors.BadRequest("Posição da mensagem do New Pack inválida")
		}
		*captionData.NewPackMessagePosition = position
	}

	rowsAffected, err := s.channelRepo.UpdateNewPackSettings(ctx, channelID, caption, captionData.NewPackMessageButtons, captionData.NewPackStickerButtons, captionData.NewPackMessagePosition, captionData.NewPackReplyToSticker)
	if err != nil {
		return 0, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	logger.Bot("✅ NewPackCaption atualizada com sucesso (Canal: %d)", channelID)

	return rowsAffected, nil
}

func (s *CaptionService) UpdateReactions(ctx context.Context, channelID int64, reactionsData types.ReactionsUpdateRequest) (int64, error) {
	if reactionsData.Reactions != "" {
		parts := strings.Split(reactionsData.Reactions, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !s.isEmoji(p) {
				return 0, errors.BadRequest("apenas emojis são permitidos como reações")
			}
		}
	}

	rowsAffected, err := s.channelRepo.UpdateReactions(ctx, channelID, reactionsData.Reactions)
	if err != nil {
		return 0, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)

	return rowsAffected, nil
}

func (s *CaptionService) UpdateReactionPosition(ctx context.Context, channelID int64, posData types.ReactionPositionUpdateRequest) (int64, error) {
	occupied, err := s.buttonRepo.IsRowOccupiedByButtons(ctx, channelID, posData.ReactionPosition)
	if err != nil {
		return 0, errors.Internal(err)
	}
	if occupied {
		return 0, errors.BadRequest("esta linha já possui botões e não pode ser usada para reações")
	}

	rowsAffected, err := s.channelRepo.UpdateReactionPosition(ctx, channelID, posData.ReactionPosition)
	if err != nil {
		return 0, errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)

	return rowsAffected, nil
}

func (s *CaptionService) UpdateNativeReactions(ctx context.Context, channelID int64, emojis string) error {
	if emojis != "" {
		parts := strings.Split(emojis, ",")
		for _, p := range parts {
			p = strings.TrimSpace(p)
			if p == "" {
				continue
			}
			if !s.isEmoji(p) {
				return errors.BadRequest("apenas emojis são permitidos para reações nativas")
			}
		}
	}

	if err := s.channelRepo.UpdateNativeReactions(ctx, channelID, emojis); err != nil {
		return errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return nil
}

func (s *CaptionService) UpdateNativeReactionMode(ctx context.Context, channelID int64, mode string) error {
	if mode != "fixed" && mode != "random" {
		return errors.BadRequest("modo deve ser 'fixed' ou 'random'")
	}

	if err := s.channelRepo.UpdateNativeReactionMode(ctx, channelID, mode); err != nil {
		return errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return nil
}

func (s *CaptionService) UpdateNativeReactionsEnabled(ctx context.Context, channelID int64, enabled bool) error {
	if err := s.channelRepo.UpdateNativeReactionsEnabled(ctx, channelID, enabled); err != nil {
		return errors.Internal(err)
	}

	s.cache.InvalidateChannel(ctx, channelID)
	return nil
}
