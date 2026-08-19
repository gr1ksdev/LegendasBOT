package services

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type PermissionsService struct {
	permissionsRepo *repositories.PermissionsRepository
	channelRepo     *repositories.ChannelRepository
	cache           *cache.Service
}

func NewPermissionsService(permissionsRepo *repositories.PermissionsRepository, channelRepo *repositories.ChannelRepository, cache *cache.Service) *PermissionsService {
	return &PermissionsService{
		permissionsRepo: permissionsRepo,
		channelRepo:     channelRepo,
		cache:           cache,
	}
}

func (s *PermissionsService) UpdateMessagePermission(ctx context.Context, channelID int64, data interface{}) (int64, error) {
	rows, err := s.permissionsRepo.UpdateMessagePermission(ctx, channelID, data)
	if err != nil {
		return 0, errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channelID)
	return rows, nil
}

func (s *PermissionsService) UpdateButtonsPermission(ctx context.Context, channelID int64, data interface{}) (int64, error) {
	rows, err := s.permissionsRepo.UpdateButtonsPermission(ctx, channelID, data)
	if err != nil {
		return 0, errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channelID)
	return rows, nil
}

func (s *PermissionsService) UpdateReactionsActive(ctx context.Context, channelID int64, active bool) (int64, error) {
	rows, err := s.permissionsRepo.UpdateReactionsActive(ctx, channelID, active)
	if err != nil {
		return 0, errors.Internal(err)
	}
	s.cache.InvalidateChannel(ctx, channelID)
	return rows, nil
}
