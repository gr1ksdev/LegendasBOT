package services

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type ServerService struct {
	serverRepo *repositories.ServerConfigRepository
}

func NewServerService(serverRepo *repositories.ServerConfigRepository) *ServerService {
	return &ServerService{serverRepo: serverRepo}
}

func (s *ServerService) GetConfig(ctx context.Context) (*models.ServerConfig, error) {
	config, err := s.serverRepo.GetServerConfig(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return config, nil
}

func (s *ServerService) UpdateConfig(ctx context.Context, maintenance, forceJoin bool, globalDefaultCaption, globalNewPackCaption string, fixedPostBuilderEnabled bool, fixedPostBuilderKey, fixedPostBuilderPayload string, logRetentionDays int) (*models.ServerConfig, error) {
	config, err := s.serverRepo.GetServerConfig(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}

	config.Maintence = maintenance
	config.ForceJoin = forceJoin
	config.GlobalDefaultCaption = globalDefaultCaption
	config.GlobalNewPackCaption = globalNewPackCaption
	config.FixedPostBuilderEnabled = fixedPostBuilderEnabled
	config.FixedPostBuilderKey = fixedPostBuilderKey
	config.FixedPostBuilderPayload = fixedPostBuilderPayload
	if logRetentionDays <= 0 {
		logRetentionDays = 30
	}
	config.LogRetentionDays = logRetentionDays

	if err := s.serverRepo.UpdateServerConfig(ctx, config); err != nil {
		return nil, errors.Internal(err)
	}
	return config, nil
}

func (s *ServerService) GetLogRetentionDays() int {
	if s == nil || s.serverRepo == nil {
		return 30
	}
	config, err := s.serverRepo.GetServerConfig(context.Background())
	if err != nil || config == nil || config.LogRetentionDays <= 0 {
		return 30
	}
	return config.LogRetentionDays
}

func (s *ServerService) SaveConfig(ctx context.Context, config *models.ServerConfig) error {
	if err := s.serverRepo.UpdateServerConfig(ctx, config); err != nil {
		return errors.Internal(err)
	}
	return nil
}

func (s *ServerService) GetMaintenance(ctx context.Context) (bool, error) {
	config, err := s.serverRepo.GetServerConfig(ctx)
	if err != nil {
		return false, errors.Internal(err)
	}
	return config.Maintence, nil
}

func (s *ServerService) ToggleMaintenance(ctx context.Context) (bool, error) {
	config, err := s.serverRepo.GetServerConfig(ctx)
	if err != nil {
		return false, errors.Internal(err)
	}

	config.Maintence = !config.Maintence
	if err := s.serverRepo.UpdateServerConfig(ctx, config); err != nil {
		return false, errors.Internal(err)
	}
	return config.Maintence, nil
}
