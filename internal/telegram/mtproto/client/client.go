// Package client gerencia os clientes MTProto (gotd/td).
//
// Fornece um wrapper seguro para criar, autenticar e gerenciar
// clientes MTProto associados a contas de usuarios.
package client

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/gotd/log/logzap"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"go.uber.org/zap"
)

// Client representa um cliente MTProto conectado para um usuario.
type Client struct {
	userID      int64
	apiID       int
	apiHash     string
	sessionData []byte

	mu        sync.RWMutex
	client    *telegram.Client
	cancelFn  context.CancelFunc
	connected bool
	lastUsed  time.Time
}

// NewClient cria um novo wrapper de cliente MTProto.
func NewClient(userID int64, apiID int, apiHash string, sessionData []byte) *Client {
	return &Client{
		userID:      userID,
		apiID:       apiID,
		apiHash:     apiHash,
		sessionData: sessionData,
	}
}

// Connect estabelece a conexao MTProto usando os dados de sessao fornecidos.
// Cria um contexto de longa duracao para o cliente.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.connected {
		return nil
	}

	logger.Bot("🔌 Conectando cliente MTProto para user=%d", c.userID)

	// Criar cliente gotd/td com sessao
	client := telegram.NewClient(c.apiID, c.apiHash, telegram.Options{
		SessionStorage: &memorySessionStorage{data: c.sessionData},
		Logger:         logzap.New(zap.NewNop()),
	})

	// Contexto para controlar o ciclo de vida
	clientCtx, cancel := context.WithCancel(ctx)

	// Iniciar o cliente em background
	go func() {
		logger.Bot("🔄 Iniciando MTProto client.Run() para user=%d", c.userID)
		if err := client.Run(clientCtx, func(ctx context.Context) error {
			<-ctx.Done()
			return ctx.Err()
		}); err != nil && err != context.Canceled {
			logger.Error("MTPROTO", "Erro no MTProto client.Run para user=%d: %v", c.userID, err)
			c.mu.Lock()
			c.connected = false
			c.mu.Unlock()
		}
	}()

	// Aguardar um pouco para a conexao estabelecer
	time.Sleep(500 * time.Millisecond)

	c.client = client
	c.cancelFn = cancel
	c.connected = true
	c.lastUsed = time.Now()

	logger.Bot("✅ Cliente MTProto conectado para user=%d", c.userID)
	return nil
}

// Disconnect encerra a conexao MTProto.
func (c *Client) Disconnect() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected {
		return
	}

	logger.Bot("🔌 Desconectando cliente MTProto para user=%d", c.userID)

	if c.cancelFn != nil {
		c.cancelFn()
	}

	c.client = nil
	c.connected = false
}

// IsConnected retorna true se o cliente esta conectado.
func (c *Client) IsConnected() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.connected
}

// API retorna o acesso a API Telegram via MTProto.
func (c *Client) API() *tg.Client {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.client == nil {
		return nil
	}
	return c.client.API()
}

// LastUsed retorna quando o cliente foi usado pela ultima vez.
func (c *Client) LastUsed() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastUsed
}

// memorySessionStorage implementa session.Storage em memoria.
// Usado para clientes temporarios durante autenticacao.
type memorySessionStorage struct {
	data []byte
}

func (m *memorySessionStorage) LoadSession(ctx context.Context) ([]byte, error) {
	if len(m.data) == 0 {
		return nil, fmt.Errorf("no session data")
	}
	return m.data, nil
}

func (m *memorySessionStorage) StoreSession(ctx context.Context, data []byte) error {
	m.data = data
	return nil
}

// Ensure telegram.SessionStorage interface is implemented.
var _ telegram.SessionStorage = (*memorySessionStorage)(nil)
