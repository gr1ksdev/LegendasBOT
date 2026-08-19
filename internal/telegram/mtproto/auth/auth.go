package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/gotd/log/logzap"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/auth"
	"github.com/gotd/td/tg"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// AuthState representa o estado temporario de autenticacao de um usuario.
// Armazenado em Redis com TTL curto (5 minutos).
type AuthState struct {
	UserID        int64  `json:"user_id"`
	PhoneNumber   string `json:"phone_number"`
	PhoneCodeHash string `json:"phone_code_hash"`
	SessionHex    string `json:"session_hex,omitempty"` // hex-encoded session data (DC info)

	HasPassword bool `json:"has_password"`
}

// sessionBytes decodifica SessionHex para []byte.
func (s *AuthState) sessionBytes() ([]byte, error) {
	if s.SessionHex == "" {
		return nil, nil
	}
	return hex.DecodeString(s.SessionHex)
}

// setSession codifica []byte em SessionHex.
func (s *AuthState) setSession(data []byte) {
	if len(data) == 0 {
		s.SessionHex = ""
		return
	}
	s.SessionHex = hex.EncodeToString(data)
}

// Status representa o status da autenticacao para retorno ao frontend.
type Status struct {
	Step        string `json:"step"`
	Error       string `json:"error,omitempty"`
	HasPassword bool   `json:"hasPassword,omitempty"`
}

// Service gerencia o fluxo de autenticacao MTProto.
type Service struct {
	redisClient       *redis.Client
	appID             int
	appHash           string
	connectedAccounts AccountSaver
}

// AccountSaver e a interface para salvar a sessao apos autenticacao.
type AccountSaver interface {
	SaveSession(ctx context.Context, userID int64, telegramUserID int64, username string, firstName string, sessionData []byte) error
}

// NewService cria um novo servico de autenticacao MTProto.
func NewService(redisClient *redis.Client, appID int, appHash string, saver AccountSaver) *Service {
	return &Service{
		redisClient:       redisClient,
		appID:             appID,
		appHash:           appHash,
		connectedAccounts: saver,
	}
}

func (s *Service) redisKey(userID int64) string {
	return fmt.Sprintf("mtproto_auth:%d", userID)
}

func (s *Service) isConfigured() bool {
	return s.appID > 0 && s.appHash != ""
}

// captureSession implementa telegram.SessionStorage para capturar
// os dados de sessao (inclui DC info) durante o fluxo de autenticacao.
type captureSession struct {
	data []byte
}

func (c *captureSession) LoadSession(_ context.Context) ([]byte, error) {
	// Se tem dados, carrega (reconecta no mesmo DC). Senao, conexao nova.
	return c.data, nil
}

func (c *captureSession) StoreSession(_ context.Context, data []byte) error {
	c.data = data
	return nil
}

var _ telegram.SessionStorage = (*captureSession)(nil)

// withAuth executa uma funcao que precisa de um client MTProto autenticavel.
// Cria o client com session storage opcional, executa fn, e retorna os dados de sessao capturados.
// sessionInit são dados de sessão de uma conexão anterior (para manter o mesmo DC).
func (s *Service) withAuth(ctx context.Context, sessionInit []byte, fn func(ctx context.Context, api *tg.Client, authClient *auth.Client) error) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	storage := &captureSession{data: sessionInit}

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

// extractUser extrai ID, username e first name de uma AuthAuthorization.
func extractUser(authResult *tg.AuthAuthorization) (telegramID int64, username string, firstName string) {
	if authResult == nil {
		return 0, "", ""
	}

	userClass := authResult.User
	if userClass == nil {
		return 0, "", ""
	}

	user, ok := userClass.(*tg.User)
	if !ok || user == nil {
		return 0, "", ""
	}

	return user.ID, user.Username, user.FirstName
}

// SendCode inicia o fluxo de autenticacao enviando o codigo para o telefone.
func (s *Service) SendCode(ctx context.Context, userID int64, phoneNumber string) (*Status, error) {
	logger.Bot("📱 Enviando codigo MTProto para telefone do usuario %d", userID)
	if !s.isConfigured() {
		return &Status{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}

	if len(phoneNumber) < 8 || len(phoneNumber) > 20 {
		return &Status{Step: "error", Error: "Número de telefone inválido"}, nil
	}

	var phoneCodeHash string
	var authSessionData []byte

	{
		var err error
		authSessionData, err = s.withAuth(ctx, nil, func(ctx context.Context, api *tg.Client, authClient *auth.Client) error {
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
			logger.Error("MTPROTO", "Erro ao enviar codigo para user %d: %v", userID, err)
			return &Status{Step: "error", Error: "Erro ao enviar código. Verifique as credenciais MTProto e o número de telefone."}, nil
		}
	}

	state := &AuthState{
		UserID:        userID,
		PhoneNumber:   phoneNumber,
		PhoneCodeHash: phoneCodeHash,
	}
	state.setSession(authSessionData)
	if err := s.saveState(ctx, state); err != nil {
		return &Status{Step: "error", Error: "Erro interno ao salvar estado"}, nil
	}

	return &Status{Step: "code"}, nil
}

// VerifyCode verifica o codigo SMS enviado para o telefone.
func (s *Service) VerifyCode(ctx context.Context, userID int64, code string) (*Status, error) {
	logger.Bot("🔐 Verificando codigo MTProto para usuario %d", userID)
	if !s.isConfigured() {
		return &Status{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}

	state, err := s.loadState(ctx, userID)
	if err != nil {
		return &Status{Step: "error", Error: "Sessão expirada. Inicie a conexão novamente."}, nil
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
		sessionData, err = s.withAuth(ctx, initData, func(ctx context.Context, api *tg.Client, authClient *auth.Client) error {
			authResult, signInErr := authClient.SignIn(ctx, state.PhoneNumber, code, state.PhoneCodeHash)
			if signInErr != nil {
				if errors.Is(signInErr, auth.ErrPasswordAuthNeeded) {
					needPassword = true
					return nil
				}
				return signInErr
			}

			tgUserID, username, firstName = extractUser(authResult)
			return nil
		})
		if err != nil {
			logger.Error("MTPROTO", "Erro ao verificar codigo para user %d: %v", userID, err)
			return &Status{Step: "error", Error: "Código inválido ou expirado"}, nil
		}
	}

	if needPassword {
		state.HasPassword = true
		if saveErr := s.saveState(ctx, state); saveErr != nil {
			logger.Error("MTPROTO", "Erro ao salvar estado 2FA: %v", saveErr)
		}
		return &Status{Step: "password", HasPassword: true}, nil
	}

	if err := s.connectedAccounts.SaveSession(ctx, userID, tgUserID, username, firstName, sessionData); err != nil {
		logger.Error("MTPROTO", "Erro ao salvar sessao para user %d: %v", userID, err)
		return &Status{Step: "error", Error: "Erro ao salvar sessão"}, nil
	}

	_ = s.deleteState(ctx, userID)

	return &Status{Step: "done"}, nil
}

// VerifyPassword verifica a senha 2FA.
func (s *Service) VerifyPassword(ctx context.Context, userID int64, password string) (*Status, error) {
	logger.Bot("🔐 Verificando senha 2FA para usuario %d", userID)
	if !s.isConfigured() {
		return &Status{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}

	state, err := s.loadState(ctx, userID)
	if err != nil {
		return &Status{Step: "error", Error: "Sessão expirada. Inicie a conexão novamente."}, nil
	}
	_ = state

	var (
		tgUserID    int64
		username    string
		firstName   string
		sessionData []byte
	)

	{
		initData, _ := state.sessionBytes()
		sessionData, err = s.withAuth(ctx, initData, func(ctx context.Context, api *tg.Client, authClient *auth.Client) error {
			authResult, passErr := authClient.Password(ctx, password)
			if passErr != nil {
				return passErr
			}

			tgUserID, username, firstName = extractUser(authResult)
			return nil
		})
		if err != nil {
			logger.Error("MTPROTO", "Erro ao verificar senha para user %d: %v", userID, err)
			return &Status{Step: "error", Error: "Senha inválida"}, nil
		}
	}

	if err := s.connectedAccounts.SaveSession(ctx, userID, tgUserID, username, firstName, sessionData); err != nil {
		logger.Error("MTPROTO", "Erro ao salvar sessao para user %d: %v", userID, err)
		return &Status{Step: "error", Error: "Erro ao salvar sessão"}, nil
	}

	_ = s.deleteState(ctx, userID)

	return &Status{Step: "done"}, nil
}

// GetStatus retorna o status atual da autenticacao.
func (s *Service) GetStatus(ctx context.Context, userID int64) (*Status, error) {
	if !s.isConfigured() {
		return &Status{Step: "error", Error: "Conexão MTProto indisponível. Configure MTPROTO_APP_ID e MTPROTO_APP_HASH."}, nil
	}
	state, err := s.loadState(ctx, userID)
	if err != nil {
		return &Status{Step: "phone"}, nil
	}

	if state.HasPassword {
		return &Status{Step: "password", HasPassword: true}, nil
	}

	return &Status{Step: "code"}, nil
}

func (s *Service) saveState(ctx context.Context, state *AuthState) error {
	data, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("failed to marshal auth state: %w", err)
	}
	return s.redisClient.Set(ctx, s.redisKey(state.UserID), data, 5*time.Minute).Err()
}

func (s *Service) loadState(ctx context.Context, userID int64) (*AuthState, error) {
	data, err := s.redisClient.Get(ctx, s.redisKey(userID)).Bytes()
	if err != nil {
		if err == redis.Nil {
			return nil, fmt.Errorf("auth session expired or not found")
		}
		return nil, fmt.Errorf("failed to load auth state: %w", err)
	}

	var state AuthState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to unmarshal auth state: %w", err)
	}
	return &state, nil
}

func (s *Service) deleteState(ctx context.Context, userID int64) error {
	return s.redisClient.Del(ctx, s.redisKey(userID)).Err()
}
