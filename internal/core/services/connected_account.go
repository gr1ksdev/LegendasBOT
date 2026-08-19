package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/telegram/mtproto/encryption"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

// ConnectedAccountService gerencia o ciclo de vida das contas conectadas.
type ConnectedAccountService struct {
	repo *repositories.ConnectedAccountRepository
}

// NewConnectedAccountService cria um novo servico de contas conectadas.
func NewConnectedAccountService(repo *repositories.ConnectedAccountRepository) *ConnectedAccountService {
	return &ConnectedAccountService{repo: repo}
}

// encryptSession criptografa os dados da sessao antes de persistir.
// Usa a SECRET_KEY do projeto como chave de criptografia (derivada via SHA-256).
func (s *ConnectedAccountService) encryptSession(sessionData []byte) (string, error) {
	key := encryption.DeriveKey(config.SecreteKey)
	encrypted, err := encryption.Encrypt(sessionData, key)
	if err != nil {
		return "", errors.Internal(err)
	}
	return encrypted, nil
}

// decryptSession descriptografa os dados da sessao para uso.
func (s *ConnectedAccountService) decryptSession(encrypted string) ([]byte, error) {
	key := encryption.DeriveKey(config.SecreteKey)
	decrypted, err := encryption.Decrypt(encrypted, key)
	if err != nil {
		return nil, errors.Internal(err)
	}
	return decrypted, nil
}

// SaveSession salva a sessao criptografada no banco.
// Se ja existir uma conta para o usuario, faz update.
func (s *ConnectedAccountService) SaveSession(
	ctx context.Context,
	userID int64,
	telegramUserID int64,
	username string,
	firstName string,
	sessionData []byte,
) (*models.ConnectedAccount, error) {
	if len(sessionData) == 0 {
		return nil, errors.BadRequest("sessão MTProto vazia não pode ser salva")
	}
	encrypted, err := s.encryptSession(sessionData)
	if err != nil {
		return nil, err
	}

	account := &models.ConnectedAccount{
		ID:               uuid.New().String(),
		UserID:           userID,
		TelegramUserID:   telegramUserID,
		Username:         username,
		FirstName:        firstName,
		EncryptedSession: encrypted,
		Enabled:          true,
	}

	// Verifica se ja existe conta para este usuario
	existing, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if existing != nil {
		// Atualizar conta existente
		existing.TelegramUserID = telegramUserID
		existing.Username = username
		existing.FirstName = firstName
		existing.EncryptedSession = encrypted
		existing.Enabled = true
		now := time.Now()
		existing.UpdatedAt = now
		if err := s.repo.Update(ctx, existing); err != nil {
			logger.Error("ACCOUNT", "Erro ao atualizar conta conectada: %v", err)
			return nil, errors.Internal(err)
		}
		logger.Bot("🔄 Conta MTProto atualizada: user=%d, telegram_id=%d", userID, telegramUserID)
		return existing, nil
	}

	if err := s.repo.Create(ctx, account); err != nil {
		logger.Error("ACCOUNT", "Erro ao salvar conta conectada: %v", err)
		return nil, errors.Internal(err)
	}

	logger.Bot("🔗 Nova conta MTProto conectada: user=%d, telegram_id=%d", userID, telegramUserID)
	return account, nil
}

// GetSession carrega e descriptografa a sessao de um usuario.
func (s *ConnectedAccountService) GetSession(ctx context.Context, userID int64) ([]byte, *models.ConnectedAccount, error) {
	account, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, nil, err
	}
	if account == nil || !account.Enabled || account.EncryptedSession == "" {
		return nil, nil, nil
	}

	sessionData, err := s.decryptSession(account.EncryptedSession)
	if err != nil {
		logger.Error("ACCOUNT", "Erro ao descriptografar sessao do usuario %d: %v", userID, err)
		return nil, nil, err
	}

	return sessionData, account, nil
}

// GetAccount retorna os dados publicos da conta conectada (sem sessao).
func (s *ConnectedAccountService) GetAccount(ctx context.Context, userID int64) (*models.ConnectedAccount, error) {
	account, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return account, nil
}

// Disconnect remove a conta conectada de um usuario.
func (s *ConnectedAccountService) Disconnect(ctx context.Context, userID int64) error {
	account, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.ErrNotFound
	}

	if err := s.repo.RemoveAllChannels(ctx, account.ID); err != nil {
		logger.Error("ACCOUNT", "Erro ao remover canais da conta: %v", err)
	}

	if err := s.repo.DeleteByUserID(ctx, userID); err != nil {
		return errors.Internal(err)
	}

	logger.Bot("🔌 Conta MTProto desconectada: user=%d", userID)
	return nil
}

// HasActiveAccount verifica se o usuario possui conta ativa.
func (s *ConnectedAccountService) HasActiveAccount(ctx context.Context, userID int64) bool {
	active, err := s.repo.HasActiveAccount(ctx, userID)
	if err != nil {
		logger.Error("ACCOUNT", "Erro ao verificar conta ativa: %v", err)
		return false
	}
	return active
}

// UpdateLastUsed atualiza o timestamp de ultimo uso.
func (s *ConnectedAccountService) UpdateLastUsed(ctx context.Context, accountID string) {
	if err := s.repo.UpdateLastUsed(ctx, accountID); err != nil {
		logger.Warn("ACCOUNT", "Erro ao atualizar last_used_at: %v", err)
	}
}

// AuthorizeChannel verifica se a conta conectada do usuario esta autorizada
// para o canal especificado.
func (s *ConnectedAccountService) AuthorizeChannel(ctx context.Context, userID int64, channelID int64) (bool, error) {
	account, err := s.repo.GetByUserID(ctx, userID)
	if err != nil {
		return false, err
	}
	if account == nil || !account.Enabled {
		return false, nil
	}

	authorized, err := s.repo.IsChannelAuthorized(ctx, account.ID, channelID)
	if err != nil {
		return false, err
	}
	return authorized, nil
}

// GetSessionAndID retorna a sessao e o ID da conta para o executor MTProto.
// Implementa a interface executor.SessionProvider.
func (s *ConnectedAccountService) GetSessionAndID(ctx context.Context, userID int64) ([]byte, string, error) {
	sessionData, account, err := s.GetSession(ctx, userID)
	if err != nil {
		return nil, "", err
	}
	if account == nil {
		return nil, "", nil
	}
	return sessionData, account.ID, nil
}

// GetAccessHash retorna o access_hash de um canal autorizado para a conta.
// Implementa executor.AccessHashProvider.
func (s *ConnectedAccountService) GetAccessHash(ctx context.Context, accountID string, channelID int64) (int64, error) {
	return s.repo.GetAccessHash(ctx, accountID, channelID)
}

// UpdateAccessHash atualiza o access_hash de um canal autorizado.
// Implementa executor.AccessHashProvider.
func (s *ConnectedAccountService) UpdateAccessHash(ctx context.Context, accountID string, channelID int64, accessHash int64) error {
	return s.repo.UpdateAccessHash(ctx, accountID, channelID, accessHash)
}
