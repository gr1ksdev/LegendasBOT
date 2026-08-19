package repositories

import (
	"context"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"gorm.io/gorm"
)

// ConnectedAccountRepository gerencia o acesso aos dados de contas conectadas.
type ConnectedAccountRepository struct {
	db *gorm.DB
}

// NewConnectedAccountRepository cria um novo repositorio.
func NewConnectedAccountRepository(db *gorm.DB) *ConnectedAccountRepository {
	return &ConnectedAccountRepository{db: db}
}

// Create insere uma nova conta conectada.
func (r *ConnectedAccountRepository) Create(ctx context.Context, account *models.ConnectedAccount) error {
	return r.db.WithContext(ctx).Create(account).Error
}

// GetByUserID busca uma conta conectada pelo ID do usuario do bot.
func (r *ConnectedAccountRepository) GetByUserID(ctx context.Context, userID int64) (*models.ConnectedAccount, error) {
	var account models.ConnectedAccount
	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &account, nil
}

// GetByTelegramUserID busca uma conta pelo ID do Telegram.
func (r *ConnectedAccountRepository) GetByTelegramUserID(ctx context.Context, telegramUserID int64) (*models.ConnectedAccount, error) {
	var account models.ConnectedAccount
	err := r.db.WithContext(ctx).Where("telegram_user_id = ?", telegramUserID).First(&account).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, errors.Internal(err)
	}
	return &account, nil
}

// Update atualiza uma conta conectada existente.
func (r *ConnectedAccountRepository) Update(ctx context.Context, account *models.ConnectedAccount) error {
	return r.db.WithContext(ctx).Save(account).Error
}

// Delete remove uma conta conectada e suas autorizacoes de canais (cascade).
func (r *ConnectedAccountRepository) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.ConnectedAccount{ID: id}).Error
}

// DeleteByUserID remove a conta conectada de um usuario.
func (r *ConnectedAccountRepository) DeleteByUserID(ctx context.Context, userID int64) error {
	return r.db.WithContext(ctx).Where("user_id = ?", userID).Delete(&models.ConnectedAccount{}).Error
}

// UpdateLastUsed atualiza o timestamp de ultimo uso.
func (r *ConnectedAccountRepository) UpdateLastUsed(ctx context.Context, id string) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&models.ConnectedAccount{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error
}

// HasActiveAccount verifica se o usuario possui uma conta ativa.
func (r *ConnectedAccountRepository) HasActiveAccount(ctx context.Context, userID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ConnectedAccount{}).
		Where("user_id = ? AND enabled = ? AND encrypted_session <> ?", userID, true, "").
		Count(&count).Error
	if err != nil {
		return false, errors.Internal(err)
	}
	return count > 0, nil
}

// CountAll retorna o numero total de contas conectadas.
func (r *ConnectedAccountRepository) CountAll(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ConnectedAccount{}).Count(&count).Error
	if err != nil {
		return 0, errors.Internal(err)
	}
	return count, nil
}

// --- Metodos para ConnectedAccountChannel ---

// AddChannel autoriza uma conta conectada a atuar em um canal.
func (r *ConnectedAccountRepository) AddChannel(ctx context.Context, accChannel *models.ConnectedAccountChannel) error {
	return r.db.WithContext(ctx).Create(accChannel).Error
}

// GetChannels retorna todos os canais autorizados para uma conta.
func (r *ConnectedAccountRepository) GetChannels(ctx context.Context, accountID string) ([]models.ConnectedAccountChannel, error) {
	var channels []models.ConnectedAccountChannel
	err := r.db.WithContext(ctx).
		Where("connected_account_id = ? AND enabled = ?", accountID, true).
		Find(&channels).Error
	if err != nil {
		return nil, errors.Internal(err)
	}
	return channels, nil
}

// IsChannelAuthorized verifica se a conta esta autorizada para um canal.
func (r *ConnectedAccountRepository) IsChannelAuthorized(ctx context.Context, accountID string, channelID int64) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&models.ConnectedAccountChannel{}).
		Where("connected_account_id = ? AND channel_id = ? AND enabled = ?", accountID, channelID, true).
		Count(&count).Error
	if err != nil {
		return false, errors.Internal(err)
	}
	return count > 0, nil
}

// GetAccessHash retorna o access_hash de um canal autorizado para a conta.
func (r *ConnectedAccountRepository) GetAccessHash(ctx context.Context, accountID string, channelID int64) (int64, error) {
	var channel models.ConnectedAccountChannel
	err := r.db.WithContext(ctx).
		Where("connected_account_id = ? AND channel_id = ? AND enabled = ?", accountID, channelID, true).
		First(&channel).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil
		}
		return 0, errors.Internal(err)
	}
	return channel.AccessHash, nil
}

// UpdateAccessHash atualiza o access_hash de um canal autorizado.
func (r *ConnectedAccountRepository) UpdateAccessHash(ctx context.Context, accountID string, channelID int64, accessHash int64) error {
	return r.db.WithContext(ctx).
		Model(&models.ConnectedAccountChannel{}).
		Where("connected_account_id = ? AND channel_id = ?", accountID, channelID).
		Update("access_hash", accessHash).Error
}

// RemoveChannel remove a autorizacao de um canal.
func (r *ConnectedAccountRepository) RemoveChannel(ctx context.Context, accountID string, channelID int64) error {
	return r.db.WithContext(ctx).
		Where("connected_account_id = ? AND channel_id = ?", accountID, channelID).
		Delete(&models.ConnectedAccountChannel{}).Error
}

// RemoveAllChannels remove todas as autorizacoes de canais de uma conta.
func (r *ConnectedAccountRepository) RemoveAllChannels(ctx context.Context, accountID string) error {
	return r.db.WithContext(ctx).
		Where("connected_account_id = ?", accountID).
		Delete(&models.ConnectedAccountChannel{}).Error
}
