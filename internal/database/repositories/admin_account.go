package repositories

import (
	"context"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"gorm.io/gorm"
)

// AdminAccountRepository gerencia o acesso aos dados de contas MTProto de admin.
type AdminAccountRepository struct {
	db *gorm.DB
}

// NewAdminAccountRepository cria um novo repositorio de contas admin.
func NewAdminAccountRepository(db *gorm.DB) *AdminAccountRepository {
	return &AdminAccountRepository{db: db}
}

// List retorna todas as contas admin, ordenadas por data de criacao (mais recentes primeiro).
func (r *AdminAccountRepository) List(ctx context.Context) ([]models.AdminMTProtoAccount, error) {
	var accounts []models.AdminMTProtoAccount
	err := r.db.WithContext(ctx).Order("created_at DESC").Find(&accounts).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return accounts, nil
}

// GetByID busca uma conta pelo ID.
func (r *AdminAccountRepository) GetByID(ctx context.Context, id string) (*models.AdminMTProtoAccount, error) {
	var account models.AdminMTProtoAccount
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &account, nil
}

// Create insere uma nova conta admin.
func (r *AdminAccountRepository) Create(ctx context.Context, account *models.AdminMTProtoAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// Update atualiza uma conta admin existente.
func (r *AdminAccountRepository) Update(ctx context.Context, account *models.AdminMTProtoAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// Delete remove uma conta admin pelo ID.
func (r *AdminAccountRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.AdminMTProtoAccount{ID: id}).Error
}

// UpdateStatus atualiza o status de uma conta admin.
func (r *AdminAccountRepository) UpdateStatus(ctx context.Context, id string, status string) error {
	return r.db.WithContext(ctx).Model(&models.AdminMTProtoAccount{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// UpdateLastUsed atualiza o timestamp de ultimo uso.
func (r *AdminAccountRepository) UpdateLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.AdminMTProtoAccount{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error
}
