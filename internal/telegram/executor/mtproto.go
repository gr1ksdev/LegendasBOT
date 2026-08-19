package executor

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/gotd/log/logzap"
	"github.com/gotd/td/telegram"
	"github.com/gotd/td/tg"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"go.uber.org/zap"
)

// SessionProvider fornece dados de sessao MTProto para um usuario.
type SessionProvider interface {
	// GetSessionAndID retorna os dados de sessao descriptografados e o ID da conta.
	// Retorna (nil, "", nil) se nao houver sessao disponivel.
	GetSessionAndID(ctx context.Context, userID int64) (sessionData []byte, accountID string, err error)
}

// AccessHashProvider fornece o access_hash de um canal para MTProto.
type AccessHashProvider interface {
	// GetAccessHash retorna o access_hash de um canal autorizado para a conta.
	// Retorna (0, nil) se nao estiver disponivel.
	GetAccessHash(ctx context.Context, accountID string, channelID int64) (int64, error)

	// UpdateAccessHash atualiza o access_hash de um canal autorizado.
	UpdateAccessHash(ctx context.Context, accountID string, channelID int64, accessHash int64) error
}

// MTProtoExecutor executa operacoes no Telegram via protocolo MTProto (gotd/td).
// Diferente do BotAPIExecutor, este executor precisa do userID para obter a sessao.
// Use UserExecutor (obtido via ExecutorFactory.ForUser) que faz o roteamento automatico.
type MTProtoExecutor struct {
	appID              int
	appHash            string
	sessionProvider    SessionProvider
	accessHashProvider AccessHashProvider
}

// NewMTProtoExecutor cria um novo executor MTProto.
func NewMTProtoExecutor(appID int, appHash string, sp SessionProvider, ahp AccessHashProvider) *MTProtoExecutor {
	return &MTProtoExecutor{
		appID:              appID,
		appHash:            appHash,
		sessionProvider:    sp,
		accessHashProvider: ahp,
	}
}

// ephemeralSession implementa telegram.SessionStorage para uso temporario.
type ephemeralSession struct {
	data []byte
}

func (e *ephemeralSession) LoadSession(_ context.Context) ([]byte, error) {
	if len(e.data) == 0 {
		return nil, fmt.Errorf("no session data")
	}
	return e.data, nil
}

func (e *ephemeralSession) StoreSession(ctx context.Context, data []byte) error {
	e.data = data
	return nil
}

var _ telegram.SessionStorage = (*ephemeralSession)(nil)

// WithSession cria um cliente MTProto ephemeral a partir de dados de sessao explicitos,
// executa fn e retorna. Util para cenarios onde a sessao nao esta vinculada a um userID
// (ex: conta admin gerenciada).
func (e *MTProtoExecutor) WithSession(ctx context.Context, sessionData []byte, accountID string, fn func(ctx context.Context, api *tg.Client, accountID string) error) error {
	if sessionData == nil {
		return fmt.Errorf("no session data")
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	client := telegram.NewClient(e.appID, e.appHash, telegram.Options{
		Logger:         logzap.New(zap.NewNop()),
		SessionStorage: &ephemeralSession{data: sessionData},
	})

	return client.Run(ctx, func(ctx context.Context) error {
		return fn(ctx, client.API(), accountID)
	})
}

// withClient cria um cliente MTProto ephemeral, executa fn e retorna.
func (e *MTProtoExecutor) withClient(ctx context.Context, userID int64, fn func(ctx context.Context, api *tg.Client, accountID string) error) error {
	sessionData, accountID, err := e.sessionProvider.GetSessionAndID(ctx, userID)
	if err != nil {
		return fmt.Errorf("get session: %w", err)
	}
	if sessionData == nil {
		return fmt.Errorf("no session data for user %d", userID)
	}
	return e.WithSession(ctx, sessionData, accountID, fn)
}

// botAPIToMTProtoChannelID converte um channel ID da Bot API (ex: -1001234567890)
// para o formato usado pelo MTProto (ex: 1234567890).
// A Bot API adiciona prefixo -100 para canais/supergrupos.
func botAPIToMTProtoChannelID(chatID int64) int64 {
	// IDs de canais/supergrupos na Bot API: -100XXXXXXXXXX (< -1000000000000)
	if chatID < -1000000000000 {
		return -chatID - 1000000000000
	}
	// IDs de usuarios/grupos comuns sao usados diretamente
	return chatID
}

// resolvePeer cria um InputPeerClass para o canal.
// Usa access_hash do banco (Bot API ID) ou tenta resolver via channels.getChannels.
// Converte o ID para o formato MTProto automaticamente.
func (e *MTProtoExecutor) resolvePeer(ctx context.Context, api *tg.Client, accountID string, chatID int64) (tg.InputPeerClass, error) {
	// Converter ID para formato MTProto (canal/supergrupo: remover prefixo -100)
	mtprotoID := botAPIToMTProtoChannelID(chatID)
	logger.Bot("resolvePeer: chatID=%d mtprotoID=%d", chatID, mtprotoID)

	accessHash, err := e.accessHashProvider.GetAccessHash(ctx, accountID, chatID)
	if err != nil {
		return nil, fmt.Errorf("get access hash: %w", err)
	}

	if accessHash == 0 {
		channels, err := api.ChannelsGetChannels(ctx, []tg.InputChannelClass{
			&tg.InputChannel{
				ChannelID:  mtprotoID,
				AccessHash: 0,
			},
		})
		if err == nil {
			chats := channels.GetChats()
			for _, ch := range chats {
				if c, ok := ch.(*tg.Channel); ok && c.ID == mtprotoID {
					accessHash = c.AccessHash
					logger.Bot("AccessHash resolvido via channels.getChannels: %d", accessHash)
					break
				}
			}
		} else {
			logger.Warn("MTPROTO", "channels.getChannels falhou, tentando com AccessHash=0: %v", err)
		}

		// Salvar access_hash para uso futuro (se conseguiu resolver)
		if accessHash != 0 {
			if saveErr := e.accessHashProvider.UpdateAccessHash(ctx, accountID, chatID, accessHash); saveErr != nil {
				logger.Error("MTPROTO", "Erro ao salvar access_hash: %v", saveErr)
			}
		} else {
			logger.Warn("MTPROTO", "AccessHash=0 para canal %d (pode falhar se o bot nao for admin)", mtprotoID)
		}
	}

	return &tg.InputPeerChannel{
		ChannelID:  mtprotoID,
		AccessHash: accessHash,
	}, nil
}

// convertKeyboard converte InlineKeyboardMarkup generico para ReplyMarkupClass do gotd.
func convertKeyboard(keyboard *InlineKeyboardMarkup) tg.ReplyMarkupClass {
	if keyboard == nil || len(keyboard.InlineKeyboard) == 0 {
		return nil
	}

	rows := make([]tg.KeyboardButtonRow, len(keyboard.InlineKeyboard))
	for i, row := range keyboard.InlineKeyboard {
		buttons := make([]tg.KeyboardButtonClass, len(row))
		for j, btn := range row {
			if btn.URL != "" {
				buttons[j] = &tg.KeyboardButtonURL{
					Text: btn.Text,
					URL:  btn.URL,
				}
			} else if btn.CallbackData != "" {
				buttons[j] = &tg.KeyboardButtonCallback{
					Text: btn.Text,
					Data: []byte(btn.CallbackData),
				}
			} else {
				buttons[j] = &tg.KeyboardButton{Text: btn.Text}
			}
		}
		rows[i] = tg.KeyboardButtonRow{Buttons: buttons}
	}

	return &tg.ReplyInlineMarkup{Rows: rows}
}

// --- Callback factories (compartilhados entre User-bound e Session-based) ---

func (e *MTProtoExecutor) editMessageCallback(chatID int64, messageID int, text string, opts *EditOptions, keyboard *InlineKeyboardMarkup) func(ctx context.Context, api *tg.Client, accountID string) error {
	return func(ctx context.Context, api *tg.Client, accountID string) error {
		peer, err := e.resolvePeer(ctx, api, accountID, chatID)
		if err != nil {
			return fmt.Errorf("resolve peer: %w", err)
		}

		req := &tg.MessagesEditMessageRequest{
			Peer: peer,
			ID:   messageID,
		}
		req.SetMessage(text)

		// Entities
		if opts != nil && opts.Entities != "" {
			entities, err := entitiesJSONToGotd(opts.Entities)
			if err != nil {
				return fmt.Errorf("parse entities: %w", err)
			}
			if len(entities) > 0 {
				req.SetEntities(entities)
			}
		}

		// No webpage
		if opts != nil && opts.DisableLinkPreview {
			req.NoWebpage = true
			req.Flags.Set(0)
		}

		// Reply markup
		if kb := convertKeyboard(keyboard); kb != nil {
			req.SetReplyMarkup(kb)
		}

		_, err = api.MessagesEditMessage(ctx, req)
		return err
	}
}

func (e *MTProtoExecutor) sendMessageCallback(chatID int64, text string, opts *EditOptions, keyboard *InlineKeyboardMarkup) func(ctx context.Context, api *tg.Client, accountID string) error {
	return func(ctx context.Context, api *tg.Client, accountID string) error {
		peer, err := e.resolvePeer(ctx, api, accountID, chatID)
		if err != nil {
			return fmt.Errorf("resolve peer: %w", err)
		}

		var randomID int64
		_ = binary.Read(rand.Reader, binary.LittleEndian, &randomID)

		req := &tg.MessagesSendMessageRequest{
			Peer:     peer,
			Message:  text,
			RandomID: randomID,
		}

		if opts != nil && opts.Entities != "" {
			entities, err := entitiesJSONToGotd(opts.Entities)
			if err != nil {
				return fmt.Errorf("parse entities: %w", err)
			}
			if len(entities) > 0 {
				req.SetEntities(entities)
			}
		}

		if kb := convertKeyboard(keyboard); kb != nil {
			req.SetReplyMarkup(kb)
		}

		_, err = api.MessagesSendMessage(ctx, req)
		return err
	}
}

func (e *MTProtoExecutor) editReplyMarkupCallback(chatID int64, messageID int, keyboard *InlineKeyboardMarkup) func(ctx context.Context, api *tg.Client, accountID string) error {
	return func(ctx context.Context, api *tg.Client, accountID string) error {
		peer, err := e.resolvePeer(ctx, api, accountID, chatID)
		if err != nil {
			return err
		}

		req := &tg.MessagesEditMessageRequest{
			Peer: peer,
			ID:   messageID,
		}

		if kb := convertKeyboard(keyboard); kb != nil {
			req.ReplyMarkup = kb
			req.Flags.Set(2)
		}

		_, err = api.MessagesEditMessage(ctx, req)
		return err
	}
}

// --- Metodos User-bound (chamados pelo UserExecutor) ---

func (e *MTProtoExecutor) EditMessageForUser(
	ctx context.Context,
	userID int64,
	chatID int64,
	messageID int,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	return e.withClient(ctx, userID, e.editMessageCallback(chatID, messageID, text, opts, keyboard))
}

func (e *MTProtoExecutor) EditCaptionForUser(
	ctx context.Context,
	userID int64,
	chatID int64,
	messageID int,
	caption string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	return e.EditMessageForUser(ctx, userID, chatID, messageID, caption, parseMode, keyboard, opts)
}

func (e *MTProtoExecutor) SendMessageForUser(
	ctx context.Context,
	userID int64,
	chatID int64,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	return e.withClient(ctx, userID, e.sendMessageCallback(chatID, text, opts, keyboard))
}

func (e *MTProtoExecutor) EditReplyMarkupForUser(
	ctx context.Context,
	userID int64,
	chatID int64,
	messageID int,
	keyboard *InlineKeyboardMarkup,
) error {
	return e.withClient(ctx, userID, e.editReplyMarkupCallback(chatID, messageID, keyboard))
}

// --- Metodos Session-based (chamados pelo PremiumExecutor) ---

func (e *MTProtoExecutor) EditMessageWithSession(
	ctx context.Context,
	sessionData []byte,
	accountID string,
	chatID int64,
	messageID int,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	return e.WithSession(ctx, sessionData, accountID, e.editMessageCallback(chatID, messageID, text, opts, keyboard))
}

func (e *MTProtoExecutor) EditCaptionWithSession(
	ctx context.Context,
	sessionData []byte,
	accountID string,
	chatID int64,
	messageID int,
	caption string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	return e.EditMessageWithSession(ctx, sessionData, accountID, chatID, messageID, caption, parseMode, keyboard, opts)
}

func (e *MTProtoExecutor) SendMessageWithSession(
	ctx context.Context,
	sessionData []byte,
	accountID string,
	chatID int64,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	return e.WithSession(ctx, sessionData, accountID, e.sendMessageCallback(chatID, text, opts, keyboard))
}

func (e *MTProtoExecutor) EditReplyMarkupWithSession(
	ctx context.Context,
	sessionData []byte,
	accountID string,
	chatID int64,
	messageID int,
	keyboard *InlineKeyboardMarkup,
) error {
	return e.WithSession(ctx, sessionData, accountID, e.editReplyMarkupCallback(chatID, messageID, keyboard))
}
