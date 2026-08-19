package services

import (
	"context"

	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/mymmrac/telego"
)

type ChannelService struct {
	channelRepo   *repositories.ChannelRepository
	userRepo      *repositories.UserRepository
	separatorRepo *repositories.SeparatorRepository
	cache         *cache.Service
	bot           *telego.Bot
}

func NewChannelService(channelRepo *repositories.ChannelRepository, userRepo *repositories.UserRepository, separatorRepo *repositories.SeparatorRepository, cache *cache.Service, bot *telego.Bot) *ChannelService {
	return &ChannelService{
		channelRepo:   channelRepo,
		userRepo:      userRepo,
		separatorRepo: separatorRepo,
		cache:         cache,
		bot:           bot,
	}
}

func (s *ChannelService) GetAllChannels(ctx context.Context) ([]models.Channel, error) {
	channels, err := s.channelRepo.GetAllChannels(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return channels, nil
}

func (s *ChannelService) GetUserChannels(ctx context.Context, userID int64) ([]models.Channel, error) {
	channels, err := s.channelRepo.GetAllChannelsByUserID(ctx, userID)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return channels, nil
}

func (s *ChannelService) GetChannelByID(ctx context.Context, channelID int64) (*models.Channel, error) {
	// 1. Tenta buscar do cache (L1 ou L2)
	if s.cache != nil {
		channel, err := s.cache.GetChannel(ctx, channelID)
		if err == nil && channel != nil {
			return channel, nil
		}
	}

	// 2. Se não estiver no cache, busca do repositório
	channel, err := s.channelRepo.GetChannelByID(ctx, channelID)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	// 3. Salva no cache para a próxima vez
	if s.cache != nil {
		_ = s.cache.SetChannel(ctx, channel)
	}

	return channel, nil
}

func (s *ChannelService) CreateChannel(ctx context.Context, channel *models.Channel) error {
	if err := s.channelRepo.CreateChannel(ctx, channel); err != nil {
		return errors.Internal(err)
	}
	return nil
}

func (s *ChannelService) TransferChannel(ctx context.Context, actorID, channelID, newOwnerID int64, isPrivileged bool) (*models.Channel, error) {
	channel, err := s.channelRepo.GetChannelByIDLight(ctx, channelID)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	if !isPrivileged && channel.OwnerID != actorID {
		return nil, errors.ErrForbidden
	}

	err = s.channelRepo.UpdateOwnerChannel(ctx, channelID, channel.OwnerID, newOwnerID)
	if err != nil {
		return nil, errors.Internal(err)
	}
	if s.cache != nil {
		_ = s.cache.InvalidateChannel(ctx, channelID)
	}
	return channel, nil
}

func (s *ChannelService) CountUserChannels(ctx context.Context, userID int64) (int64, error) {
	count, err := s.channelRepo.CountUserChannels(ctx, userID)
	if err != nil {
		return 0, errors.Internal(err)
	}
	return count, nil
}

func (s *ChannelService) GetChannelWithRelations(ctx context.Context, channelID int64) (*models.Channel, error) {
	return s.GetChannelByID(ctx, channelID)
}

func (s *ChannelService) UpdateChannelBasicInfoAndFirstButton(ctx context.Context, channel *models.Channel) error {
	if err := s.channelRepo.UpdateChannel(ctx, channel); err != nil {
		return errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channel.ID)
	return nil
}

func (s *ChannelService) UpdateOwnerChannel(ctx context.Context, channelID, oldOwnerID, newOwnerID int64) error {
	_, err := s.TransferChannel(ctx, oldOwnerID, channelID, newOwnerID, false)
	return err
}

func (s *ChannelService) GetChannelByTwoID(ctx context.Context, userID, channelID int64) (*models.Channel, error) {
	// 1. Tenta buscar do cache
	if s.cache != nil {
		channel, err := s.cache.GetChannel(ctx, channelID)
		if err == nil && channel != nil {
			// Valida se o dono é o mesmo
			if channel.OwnerID == userID {
				return channel, nil
			}
		}
	}

	// 2. Busca do repo se não estiver no cache ou se o dono for diferente (segurança)
	channel, err := s.channelRepo.GetChannelByTwoID(ctx, userID, channelID)
	if err != nil {
		return nil, errors.ErrNotFound
	}

	// 3. Salva no cache
	if s.cache != nil {
		_ = s.cache.SetChannel(ctx, channel)
	}

	return channel, nil
}

func (s *ChannelService) GetAllChannelsPaginated(ctx context.Context, limit, offset int) ([]models.Channel, int64, error) {
	channels, total, err := s.channelRepo.GetAllChannelsPaginated(ctx, limit, offset)
	if err != nil {
		return nil, 0, errors.Internal(err)
	}
	return channels, total, nil
}

func (s *ChannelService) CreateChannelWithDefaults(ctx context.Context, channelID int64, title, inviteURL, newPackCaption, defaultCaption string, ownerID int64) (*models.Channel, error) {
	inviteURL = utils.NormalizeTelegramURL(inviteURL)
	captionID := uuid.NewString()
	newPackMessageButtons := true
	newPackStickerButtons := true
	newPackMessagePosition := "above"
	newPackReplyToSticker := false
	channel := &models.Channel{
		ID:                     channelID,
		Title:                  title,
		InviteURL:              inviteURL,
		NewPackCaption:         newPackCaption,
		NewPackMessageButtons:  &newPackMessageButtons,
		NewPackStickerButtons:  &newPackStickerButtons,
		NewPackMessagePosition: &newPackMessagePosition,
		NewPackReplyToSticker:  &newPackReplyToSticker,
		OwnerID:                ownerID,
		DefaultCaption: &models.DefaultCaption{
			CaptionID: captionID,
			Caption:   defaultCaption,
			MessagePermission: &models.MessagePermission{
				MessagePermissionID: uuid.NewString(),
				OwnerCaptionID:      captionID,
			},
			ButtonsPermission: &models.ButtonsPermission{
				ButtonsPermissionID: uuid.NewString(),
				OwnerCaptionID:      captionID,
			},
		},
		Buttons: []models.Button{
			{
				ButtonID:   uuid.NewString(),
				NameButton: title,
				ButtonURL:  inviteURL,
				PositionX:  0,
				PositionY:  0,
			},
		},
	}

	if err := s.CreateChannel(ctx, channel); err != nil {
		return nil, err
	}
	return channel, nil
}

func (s *ChannelService) DeleteChannel(ctx context.Context, userID int64, channelID int64) error {
	err := s.channelRepo.DeleteChannelWithRelations(ctx, userID, channelID)
	if err != nil {
		return errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channelID)
	// Limpa todas as sessões do usuário (RAM e Redis) para evitar inconsistências
	_, _ = s.cache.DeleteAllUserSessionsBySuffix(ctx, userID)
	return nil
}

func (s *ChannelService) DisconnectChannel(ctx context.Context, userID int64, channelID int64) error {
	// 1. Send farewell message to the channel
	farewellMsg := "Ate breve, bye 👋"

	// Tenta enviar a mensagem, mas não bloqueia se falhar (o bot pode já ter sido removido manualmente)
	if s.bot != nil {
		_, _ = s.bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID: telego.ChatID{ID: channelID},
			Text:   farewellMsg,
		})

		// 2. Leave the channel
		_ = s.bot.LeaveChat(context.Background(), &telego.LeaveChatParams{
			ChatID: telego.ChatID{ID: channelID},
		})
	}

	// 3. Delete from DB
	return s.DeleteChannel(ctx, userID, channelID)
}

func (s *ChannelService) UpdateDynamicLinks(ctx context.Context, channelID int64, settings map[string]any) error {
	_, err := s.channelRepo.UpdateDynamicLinks(ctx, channelID, settings)
	if err != nil {
		return errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channelID)
	return nil
}
