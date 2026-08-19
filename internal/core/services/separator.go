package services

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type SeparatorService struct {
	separatorRepo *repositories.SeparatorRepository
}

func NewSeparatorService(separatorRepo *repositories.SeparatorRepository) *SeparatorService {
	return &SeparatorService{separatorRepo: separatorRepo}
}

func (s *SeparatorService) GetSeparatorByTwoID(ctx context.Context, channelId int64, separatorId string) (*models.Separator, error) {
	sep, err := s.separatorRepo.GetSeparatorByTwoID(ctx, channelId, separatorId)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	return sep, nil
}

func (s *SeparatorService) SaveSeparator(ctx context.Context, separator *models.Separator) error {
	if err := s.separatorRepo.SaveSeparator(ctx, separator); err != nil {
		return errors.Internal(err)
	}
	return nil
}

func (s *SeparatorService) GetSeparatorByOwnerChannelID(ctx context.Context, channelID int64) (*models.Separator, error) {
	sep, err := s.separatorRepo.GetSeparatorByOwnerChannelID(ctx, channelID)
	if err != nil {
		return nil, errors.ErrNotFound
	}
	return sep, nil
}

func (s *SeparatorService) DeleteSeparatorByOwnerChannelId(ctx context.Context, channelID int64) error {
	if err := s.separatorRepo.DeleteSeparatorByOwnerChannelId(ctx, channelID); err != nil {
		return errors.Internal(err)
	}
	return nil
}
