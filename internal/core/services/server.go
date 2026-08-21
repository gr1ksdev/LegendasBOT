package services

import (
	"context"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type ServerService struct {
	serverRepo   *repositories.ServerConfigRepository
	mu           sync.RWMutex
	cachedConfig *models.ServerConfig
	cachedAt     time.Time
}

func NewServerService(serverRepo *repositories.ServerConfigRepository) *ServerService {
	return &ServerService{serverRepo: serverRepo}
}

func (s *ServerService) GetConfig(ctx context.Context) (*models.ServerConfig, error) {
	s.mu.RLock()
	if s.cachedConfig != nil && time.Since(s.cachedAt) < 15*time.Second {
		cfg := *s.cachedConfig
		s.mu.RUnlock()
		return &cfg, nil
	}
	s.mu.RUnlock()

	config, err := s.serverRepo.GetServerConfig(ctx)
	if err != nil {
		return nil, errors.Internal(err)
	}

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return config, nil
}

func (s *ServerService) UpdateConfig(ctx context.Context, maintenance, forceJoin bool, globalDefaultCaption, globalNewPackCaption string, fixedPostBuilderEnabled bool, fixedPostBuilderKey, fixedPostBuilderPayload string, logRetentionDays int, logsEnabled bool) (*models.ServerConfig, error) {
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
	config.LogsEnabled = logsEnabled

	if err := s.serverRepo.UpdateServerConfig(ctx, config); err != nil {
		return nil, errors.Internal(err)
	}

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return config, nil
}

func (s *ServerService) IsLogsEnabled() bool {
	if s == nil || s.serverRepo == nil {
		return true
	}

	s.mu.RLock()
	if s.cachedConfig != nil && time.Since(s.cachedAt) < 15*time.Second {
		enabled := s.cachedConfig.LogsEnabled
		s.mu.RUnlock()
		return enabled
	}
	s.mu.RUnlock()

	config, err := s.serverRepo.GetServerConfig(context.Background())
	if err != nil || config == nil {
		return true
	}

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return config.LogsEnabled
}

func (s *ServerService) GetLogRetentionDays() int {
	if s == nil || s.serverRepo == nil {
		return 30
	}

	s.mu.RLock()
	if s.cachedConfig != nil && time.Since(s.cachedAt) < 15*time.Second {
		days := s.cachedConfig.LogRetentionDays
		s.mu.RUnlock()
		if days <= 0 {
			return 30
		}
		return days
	}
	s.mu.RUnlock()

	config, err := s.serverRepo.GetServerConfig(context.Background())
	if err != nil || config == nil || config.LogRetentionDays <= 0 {
		return 30
	}

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return config.LogRetentionDays
}

func (s *ServerService) SaveConfig(ctx context.Context, config *models.ServerConfig) error {
	if err := s.serverRepo.UpdateServerConfig(ctx, config); err != nil {
		return errors.Internal(err)
	}

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return nil
}

func (s *ServerService) GetMaintenance(ctx context.Context) (bool, error) {
	s.mu.RLock()
	if s.cachedConfig != nil && time.Since(s.cachedAt) < 15*time.Second {
		m := s.cachedConfig.Maintence
		s.mu.RUnlock()
		return m, nil
	}
	s.mu.RUnlock()

	config, err := s.serverRepo.GetServerConfig(ctx)
	if err != nil {
		return false, errors.Internal(err)
	}

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

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

	s.mu.Lock()
	s.cachedConfig = config
	s.cachedAt = time.Now()
	s.mu.Unlock()

	return config.Maintence, nil
}
