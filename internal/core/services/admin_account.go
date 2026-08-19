package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/gotd/log/logzap"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/telegram/mtproto/encryption"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// adminAuthRedisPrefix e o prefixo para chaves Redis de auth de admin.
const adminAuthRedisPrefix = "admin_mtproto_auth:"

// adminAuthState representa o estado temporario de autenticacao de uma conta admin.
// Armazenado em Redis com TTL curto (5 minutos).
type adminAuthState struct {
	SessionID     string `json:"session_id"`
	Label         string `json:"label"`
	PhoneNumber   string `json:"phone_number"`
	PhoneCodeHash string `json:"phone_code_hash"`
	SessionHex    string `json:"session_hex,omitempty"`
	HasPassword   bool   `json:"has_password"`
}

func (s *adminAuthState) sessionBytes() ([]byte, error) {
	if s.SessionHex == "" {
		return nil, nil
	}
	return hex.DecodeString(s.SessionHex)
}

func (s *adminAuthState) setSession(data []byte) {
	if len(data) == 0 {
		s.SessionHex = ""
		return
	}
	s.SessionHex = hex.EncodeToString(data)
}

// AdminAccountService gerencia o ciclo de vida das contas MTProto de admin.
type AdminAccountService struct {
	repo    *repositories.AdminAccountRepository
	redis   *redis.Client
	appID   int
	appHash string
}

// NewAdminAccountService cria um novo servico de contas admin.
func NewAdminAccountService(
	repo *repositories.AdminAccountRepository,
	redis *redis.Client,
	appID int,
	appHash string,
) *AdminAccountService {
	return &AdminAccountService{
		repo:    repo,
		redis:   redis,
		appID:   appID,
		appHash: appHash,
	}
}

// isConfigured verifica se as credenciais MTProto estao configuradas.
func (s *AdminAccountService) isConfigured() bool {
	return s.appID > 0 && s.appHash != ""
}

// redisKey gera a chave Redis para um estado de auth admin.
func (s *AdminAccountService) redisKey(sessionID string) string {
	return adminAuthRedisPrefix + sessionID
}

// generateSessionID gera um ID unico para a sessao de auth admin.
func generateSessionID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// ── Gerenciamento de Contas ──

// ListAccounts retorna todas as contas admin.
func (s *AdminAccountService) ListAccounts(ctx context.Context) ([]models.AdminMTProtoAccount, error) {
	return s.repo.List(ctx)
}

// DeleteAccount remove uma conta admin.
func (s *AdminAccountService) DeleteAccount(ctx context.Context, id string) error {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if account == nil {
		return errors.ErrNotFound
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return errors.Internal(err)
	}

	logger.Bot("🗑️ Conta MTProto admin removida: %s (%s)", account.Label, account.Username)
	return nil
}

// ToggleAccount ativa/desativa uma conta admin.
func (s *AdminAccountService) ToggleAccount(ctx context.Context, id string, enabled bool) (*models.AdminMTProtoAccount, error) {
	account, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, errors.ErrNotFound
	}

	account.Enabled = enabled
	if err := s.repo.Update(ctx, account); err != nil {
		return nil, errors.Internal(err)
	}

	status := "desativada"
	if enabled {
		status = "ativada"
	}
	logger.Bot("🔄 Conta MTProto admin %s: %s", account.Label, status)
	return account, nil
}

// ── Autenticacao MTProto ──

// AuthStep representa o passo atual da autenticacao.
type AdminAuthStep struct {
	Step        string `json:"step"`
	SessionID   string `json:"sessionId,omitempty"`
	HasPassword bool   `json:"hasPassword,omitempty"`
	Error       string `json:"error,omitempty"`
}

// StartAuth inicia o fluxo de autenticacao para uma nova conta admin.
func (s *AdminAccountService) StartAuth(ctx context.Context, label string, phoneNumber string) (*AdminAuthStep, error) {
	logger.Bot("📱 Iniciando auth MTProto admin: label=%s phone=%s", label, utils.MaskPhone(phoneNumber))
	if !s.isConfigured() {
		return &AdminAuthStep{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}

	if len(phoneNumber) < 8 || len(phoneNumber) > 20 {
		return &AdminAuthStep{Step: "error", Error: "Número de telefone inválido"}, nil
	}

	sessionID := generateSessionID()
	var phoneCodeHash string

	{
		var authSessionData []byte
		var err error

		// Usar withAuth do pacote auth (reimplementado localmente para admin)
		authSessionData, phoneCodeHash, err = s.sendCode(ctx, phoneNumber)
		if err != nil {
			logger.Error("ADMIN_ACCOUNT", "Erro ao enviar codigo: %v", err)
			return &AdminAuthStep{Step: "error", Error: "Erro ao enviar código. Verifique as credenciais MTProto."}, nil
		}

		state := &adminAuthState{
			SessionID:     sessionID,
			Label:         label,
			PhoneNumber:   phoneNumber,
			PhoneCodeHash: phoneCodeHash,
		}
		state.setSession(authSessionData)
		if err := s.saveState(ctx, state); err != nil {
			return &AdminAuthStep{Step: "error", Error: "Erro interno ao salvar estado"}, nil
		}
	}

	return &AdminAuthStep{Step: "code", SessionID: sessionID}, nil
}

// VerifyCode verifica o codigo SMS durante o fluxo de autenticacao admin.
func (s *AdminAccountService) VerifyCode(ctx context.Context, sessionID string, code string) (*AdminAuthStep, error) {
	logger.Bot("🔐 Verificando codigo MTProto admin (session=%s)", sessionID)
	if !s.isConfigured() {
		return &AdminAuthStep{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}

	state, err := s.loadState(ctx, sessionID)
	if err != nil {
		return &AdminAuthStep{Step: "error", Error: "Sessão expirada. Inicie a conexão novamente."}, nil
	}

	var (
		tgUserID     int64
		username     string
		firstName    string
		needPassword bool
		sessionData  []byte
	)

	{
		initData, _ := state.sessionBytes()
		sessionData, tgUserID, username, firstName, needPassword, err = s.verifyCode(ctx, state.PhoneNumber, code, state.PhoneCodeHash, initData)
		if err != nil {
			logger.Error("ADMIN_ACCOUNT", "Erro ao verificar codigo: %v", err)
			return &AdminAuthStep{Step: "error", Error: "Código inválido ou expirado"}, nil
		}
	}

	if needPassword {
		state.HasPassword = true
		if saveErr := s.saveState(ctx, state); saveErr != nil {
			logger.Error("ADMIN_ACCOUNT", "Erro ao salvar estado 2FA: %v", saveErr)
		}
		return &AdminAuthStep{Step: "password", HasPassword: true, SessionID: sessionID}, nil
	}

	// Salvar conta no banco
	account, err := s.createAccount(ctx, state.Label, state.PhoneNumber, tgUserID, username, firstName, sessionData)
	if err != nil {
		return &AdminAuthStep{Step: "error", Error: "Erro ao salvar sessão"}, nil
	}

	_ = s.deleteState(ctx, sessionID)

	logger.Bot("🔗 Nova conta MTProto admin conectada: %s (@%s, id=%d)", account.Label, account.Username, account.TelegramUserID)
	return &AdminAuthStep{Step: "done"}, nil
}

// VerifyPassword verifica a senha 2FA durante o fluxo de autenticacao admin.
func (s *AdminAccountService) VerifyPassword(ctx context.Context, sessionID string, password string) (*AdminAuthStep, error) {
	logger.Bot("🔐 Verificando senha 2FA MTProto admin (session=%s)", sessionID)
	if !s.isConfigured() {
		return &AdminAuthStep{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}

	state, err := s.loadState(ctx, sessionID)
	if err != nil {
		return &AdminAuthStep{Step: "error", Error: "Sessão expirada. Inicie a conexão novamente."}, nil
	}

	var (
		tgUserID    int64
		username    string
		firstName   string
		sessionData []byte
	)

	{
		initData, _ := state.sessionBytes()
		sessionData, tgUserID, username, firstName, err = s.verifyPassword(ctx, password, initData)
		if err != nil {
			logger.Error("ADMIN_ACCOUNT", "Erro ao verificar senha: %v", err)
			return &AdminAuthStep{Step: "error", Error: "Senha inválida"}, nil
		}
	}

	account, err := s.createAccount(ctx, state.Label, state.PhoneNumber, tgUserID, username, firstName, sessionData)
	if err != nil {
		return &AdminAuthStep{Step: "error", Error: "Erro ao salvar sessão"}, nil
	}

	_ = s.deleteState(ctx, sessionID)

	logger.Bot("🔗 Nova conta MTProto admin conectada: %s (@%s, id=%d)", account.Label, account.Username, account.TelegramUserID)
	return &AdminAuthStep{Step: "done"}, nil
}

// GetAuthStatus retorna o status atual da autenticacao para uma sessao.
func (s *AdminAccountService) GetAuthStatus(ctx context.Context, sessionID string) (*AdminAuthStep, error) {
	if !s.isConfigured() {
		return &AdminAuthStep{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}
	state, err := s.loadState(ctx, sessionID)
	if err != nil {
		return &AdminAuthStep{Step: "done", SessionID: sessionID}, nil
	}

	if state.HasPassword {
		return &AdminAuthStep{Step: "password", HasPassword: true, SessionID: sessionID}, nil
	}

	return &AdminAuthStep{Step: "code", SessionID: sessionID}, nil
}

// ── Metodos internos MTProto ──

// captureSession implementa telegram.SessionStorage para capturar os dados de sessao.
type adminCaptureSession struct {
	data []byte
}

func (c *adminCaptureSession) LoadSession(_ context.Context) ([]byte, error) {
	return c.data, nil
}

func (c *adminCaptureSession) StoreSession(_ context.Context, data []byte) error {
	c.data = data
	return nil
}

var _ telegram.SessionStorage = (*adminCaptureSession)(nil)

// withAuth executa uma funcao que precisa de um client MTProto autenticavel.
func (s *AdminAccountService) withAuth(ctx context.Context, sessionInit []byte, fn func(ctx context.Context, api *tg.Client, authClient *auth.Client) error) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	storage := &adminCaptureSession{data: sessionInit}

	client := telegram.NewClient(s.appID, s.appHash, telegram.Options{
		Logger:         logzap.New(zap.NewNop()),
		SessionStorage: storage,
	})

	err := client.Run(ctx, func(ctx context.Context) error {
		authClient := auth.NewClient(client.API(), rand.Reader, s.appID, s.appHash)
		return fn(ctx, client.API(), authClient)
	})

	return storage.data, err
}

// sendCode envia o codigo de autenticacao para o telefone.
func (s *AdminAccountService) sendCode(ctx context.Context, phoneNumber string) ([]byte, string, error) {
	var phoneCodeHash string

	sessionData, err := s.withAuth(ctx, nil, func(ctx context.Context, api *tg.Client, authClient *auth.Client) error {
		sentCode, err := authClient.SendCode(ctx, phoneNumber, auth.SendCodeOptions{})
		if err != nil {
			return fmt.Errorf("send code: %w", err)
		}

		code, ok := sentCode.(*tg.AuthSentCode)
		if !ok {
			return fmt.Errorf("unexpected sent code type %T", sentCode)
		}

		phoneCodeHash = code.PhoneCodeHash
		return nil
	})
	if err != nil {
		return nil, "", err
	}

	return sessionData, phoneCodeHash, nil
}

// verifyCode verifica o codigo SMS.
func (s *AdminAccountService) verifyCode(ctx context.Context, phoneNumber, code, phoneCodeHash string, initData []byte) ([]byte, int64, string, string, bool, error) {
	var (
		tgUserID     int64
		username     string
		firstName    string
		needPassword bool
	)

	sessionData, err := s.withAuth(ctx, initData, func(ctx context.Context, api *tg.Client, authClient *auth.Client) error {
		authResult, signInErr := authClient.SignIn(ctx, phoneNumber, code, phoneCodeHash)
		if signInErr != nil {
			if signInErr == auth.ErrPasswordAuthNeeded {
				needPassword = true
				return nil
			}
			return signInErr
		}

		if authResult == nil || authResult.User == nil {
			return fmt.Errorf("no user in auth result")
		}

		user, ok := authResult.User.(*tg.User)
		if !ok || user == nil {
			return fmt.Errorf("unexpected user type")
		}

		tgUserID = user.ID
		username = user.Username
		firstName = user.FirstName
		return nil
	})
	if err != nil {
		return nil, 0, "", "", false, err
	}

	return sessionData, tgUserID, username, firstName, needPassword, nil
}

// verifyPassword verifica a senha 2FA.
func (s *AdminAccountService) verifyPassword(ctx context.Context, password string, initData []byte) ([]byte, int64, string, string, error) {
	var (
		tgUserID  int64
		username  string
		firstName string
	)

	sessionData, err := s.withAuth(ctx, initData, func(ctx context.Context, api *tg.Client, authClient *auth.Client) error {
		authResult, passErr := authClient.Password(ctx, password)
		if passErr != nil {
			return passErr
		}

		if authResult == nil || authResult.User == nil {
			return fmt.Errorf("no user in auth result")
		}

		user, ok := authResult.User.(*tg.User)
		if !ok || user == nil {
			return fmt.Errorf("unexpected user type")
		}

		tgUserID = user.ID
		username = user.Username
		firstName = user.FirstName
		return nil
	})
	if err != nil {
		return nil, 0, "", "", err
	}

	return sessionData, tgUserID, username, firstName, nil
}

// ── Persistencia ──

// encryptSession criptografa os dados da sessao.
func (s *AdminAccountService) encryptSession(sessionData []byte) (string, error) {
	key := encryption.DeriveKey(config.SecreteKey)
	encrypted, err := encryption.Encrypt(sessionData, key)
	if err != nil {
		return "", errors.Internal(err)
	}
	return encrypted, nil
}

// decryptSession descriptografa os dados da sessao para uso no executor MTProto.
func (s *AdminAccountService) decryptSession(encrypted string) ([]byte, error) {
	key := encryption.DeriveKey(config.SecreteKey)
	decrypted, err := encryption.Decrypt(encrypted, key)
	if err != nil {
		logger.Error("ADMIN_ACCOUNT", "Erro ao descriptografar sessao admin: %v", err)
		return nil, errors.Internal(err)
	}
	return decrypted, nil
}

// GetAdminSession retorna os dados de sessao descriptografados e o ID da primeira
// conta admin habilitada e conectada disponivel. Utilizado pelo PremiumExecutor
// para executar operacoes MTProto em nome de usuarios premium.
// Retorna (nil, "", nil) se nenhuma conta admin estiver disponivel.
func (s *AdminAccountService) GetAdminSession(ctx context.Context) ([]byte, string, error) {
	accounts, err := s.repo.List(ctx)
	if err != nil {
		logger.Error("ADMIN_ACCOUNT", "Erro ao listar contas admin: %v", err)
		return nil, "", err
	}

	for _, acc := range accounts {
		if acc.Enabled && acc.Status == "connected" && acc.EncryptedSession != "" {
			sessionData, err := s.decryptSession(acc.EncryptedSession)
			if err != nil {
				logger.Error("ADMIN_ACCOUNT", "Erro ao descriptografar sessao da conta %s: %v", acc.ID, err)
				continue
			}
			if len(sessionData) > 0 {
				return sessionData, acc.ID, nil
			}
		}
	}

	return nil, "", nil
}

// createAccount salva uma nova conta admin no banco.
func (s *AdminAccountService) createAccount(ctx context.Context, label, phoneNumber string, tgUserID int64, username, firstName string, sessionData []byte) (*models.AdminMTProtoAccount, error) {
	if len(sessionData) == 0 {
		return nil, errors.BadRequest("sessão MTProto vazia não pode ser salva")
	}
	encrypted, err := s.encryptSession(sessionData)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	account := &models.AdminMTProtoAccount{
		ID:               uuid.New().String(),
		Label:            label,
		PhoneNumber:      phoneNumber,
		TelegramUserID:   tgUserID,
		Username:         username,
		FirstName:        firstName,
		EncryptedSession: encrypted,
		Enabled:          true,
		Status:           "connected",
		LastUsedAt:       &now,
	}

	if err := s.repo.Create(ctx, account); err != nil {
		logger.Error("ADMIN_ACCOUNT", "Erro ao salvar conta admin: %v", err)
		return nil, errors.Internal(err)
	}

	return account, nil
}

// ── Redis State Management ──

func (s *AdminAccountService) saveState(ctx context.Context, state *adminAuthState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal admin auth state: %w", err)
	}
	return s.redis.Set(ctx, s.redisKey(state.SessionID), data, 5*time.Minute).Err()
}

func (s *AdminAccountService) loadState(ctx context.Context, sessionID string) (*adminAuthState, error) {
	data, err := s.redis.Get(ctx, s.redisKey(sessionID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("admin auth session expired or not found")
		}
		return nil, fmt.Errorf("failed to load admin auth state: %w", err)
	}

	var state adminAuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal admin auth state: %w", err)
	}
	return &state, nil
}

func (s *AdminAccountService) deleteState(ctx context.Context, sessionID string) error {
	return s.redis.Del(ctx, s.redisKey(sessionID)).Err()
}
