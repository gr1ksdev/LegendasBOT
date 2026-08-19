package repositories

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"gorm.io/gorm"
)

type ServerConfigRepository struct {
	db *gorm.DB
}

func NewServerConfigRepository(db *gorm.DB) *ServerConfigRepository {
	return &ServerConfigRepository{db: db}
}

func (r *ServerConfigRepository) GetServerConfig(ctx context.Context) (*models.ServerConfig, error) {
	var config models.ServerConfig
	if err := r.db.WithContext(ctx).First(&config, 1).Error; err != nil {
		return nil, err
	}
	return &config, nil
}

func (r *ServerConfigRepository) UpdateServerConfig(ctx context.Context, config *models.ServerConfig) error {
	return r.db.WithContext(ctx).Save(config).Error
}
