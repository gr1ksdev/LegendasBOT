// Package session gerencia o armazenamento e carregamento de sessoes
// MTProto no PostgreSQL.
//
// Implementa a interface session.Storage do gotd/td para que o cliente
// MTProto possa carregar e salvar sessoes automaticamente.
package session

import (
	"context"
	"fmt"

	"github.com/gotd/td/session"
	"github.com/gotd/td/telegram"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"go.uber.org/zap"
)

// PostgresSessionStorage implementa session.Storage do gotd/td usando
// o banco de dados PostgreSQL.
//
// As sessoes sao armazenadas criptografadas e associadas ao usuario
// do bot que conectou a conta Telegram.
type PostgresSessionStorage struct {
	userID  int64
	session []byte
}

// NewPostgresSessionStorage cria um novo armazenamento de sessao.
// userID: ID do usuario dono da conta.
// sessionData: dados da sessao ja descriptografados (opcional).
func NewPostgresSessionStorage(userID int64, sessionData []byte) *PostgresSessionStorage {
	return &PostgresSessionStorage{
		userID:  userID,
		session: sessionData,
	}
}

// LoadSession carrega a sessao do armazenamento.
// Implementa session.Storage.
func (s *PostgresSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	if len(s.session) == 0 {
		logger.Bot("📭 Nenhuma sessao MTProto encontrada para user=%d", s.userID)
		return nil, session.ErrNotFound
	}
	logger.Bot("📥 Sessao MTProto carregada para user=%d (%d bytes)", s.userID, len(s.session))
	return s.session, nil
}

// StoreSession armazena a sessao no storage.
// Implementa session.Storage.
// NOTA: O armazenamento efetivo e feito pelo ConnectedAccountService.
// Este metodo apenas registra em memoria para uso durante a sessao.
func (s *PostgresSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("cannot store empty session data")
	}
	logger.Bot("📥 Sessao MTProto armazenada para user=%d (%d bytes)", s.userID, len(data))
	s.session = data
	return nil
}

// telegramSessionStorage adapta nosso storage para o formato esperado
// pelo gotd/td telegram.Client.
type telegramSessionStorage struct {
	inner *PostgresSessionStorage
}

func (t *telegramSessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	return t.inner.LoadSession(ctx)
}

func (t *telegramSessionStorage) StoreSession(ctx context.Context, data []byte) error {
	return t.inner.StoreSession(ctx, data)
}

// NewTelegramSessionStorage cria um session.Storage compativel com gotd/td.
func NewTelegramSessionStorage(s *PostgresSessionStorage) telegram.SessionStorage {
	return &telegramSessionStorage{inner: s}
}

// NopLogger e um logger do zap que nao produz saida.
// Usado para evitar dependencia de configuracao do zap no gotd.
var NopLogger = zap.NewNop()
