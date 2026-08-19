package executor

import (
	"context"
)

// AdminSessionProvider fornece acesso a sessao de uma conta admin gerenciada
// para ser usada pelo PremiumExecutor em nome de usuarios premium.
type AdminSessionProvider interface {
	// GetAdminSession retorna os dados de sessao descriptografados e o ID
	// da primeira conta admin habilitada e conectada disponivel.
	// Retorna (nil, "", nil) se nenhuma conta admin estiver disponivel.
	GetAdminSession(ctx context.Context) (sessionData []byte, accountID string, err error)
}

// PremiumExecutor e um TelegramExecutor vinculado a um usuario premium.
// Quando um admin account session esta disponivel, usa MTProto (conta admin gerenciada)
// para executar as operacoes. Caso contrario, faz fallback para BotAPI.
type PremiumExecutor struct {
	userID  int64
	botAPI  TelegramExecutor
	mtproto *MTProtoExecutor
	adminSP AdminSessionProvider
}

// NewPremiumExecutor cria um executor premium que usa a conta admin gerenciada
// como backend MTProto para o usuario especificado.
func NewPremiumExecutor(userID int64, botAPI TelegramExecutor, mtproto *MTProtoExecutor, adminSP AdminSessionProvider) *PremiumExecutor {
	return &PremiumExecutor{
		userID:  userID,
		botAPI:  botAPI,
		mtproto: mtproto,
		adminSP: adminSP,
	}
}

// tryAdminSession tenta obter uma sessao admin e executa fn via MTProto.
// Se nao houver sessao admin disponivel, retorna false e o caller deve fazer fallback.
func (p *PremiumExecutor) tryAdminSession(ctx context.Context, fn func(sessionData []byte, accountID string) error) (bool, error) {
	sessionData, accountID, err := p.adminSP.GetAdminSession(ctx)
	if err != nil {
		return false, nil // log interna, fallback silencioso
	}
	if sessionData == nil {
		return false, nil
	}
	if err := fn(sessionData, accountID); err != nil {
		return true, err
	}
	return true, nil
}

// --- Implementacao de TelegramExecutor ---

func (p *PremiumExecutor) EditMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	ok, err := p.tryAdminSession(ctx, func(sessionData []byte, accountID string) error {
		return p.mtproto.EditMessageWithSession(ctx, sessionData, accountID, chatID, messageID, text, parseMode, keyboard, opts)
	})
	if ok {
		return err
	}
	return p.botAPI.EditMessage(ctx, chatID, messageID, text, parseMode, keyboard, opts)
}

func (p *PremiumExecutor) EditCaption(
	ctx context.Context,
	chatID int64,
	messageID int,
	caption string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	ok, err := p.tryAdminSession(ctx, func(sessionData []byte, accountID string) error {
		return p.mtproto.EditCaptionWithSession(ctx, sessionData, accountID, chatID, messageID, caption, parseMode, keyboard, opts)
	})
	if ok {
		return err
	}
	return p.botAPI.EditCaption(ctx, chatID, messageID, caption, parseMode, keyboard, opts)
}

func (p *PremiumExecutor) EditReplyMarkup(
	ctx context.Context,
	chatID int64,
	messageID int,
	keyboard *InlineKeyboardMarkup,
) error {
	ok, err := p.tryAdminSession(ctx, func(sessionData []byte, accountID string) error {
		return p.mtproto.EditReplyMarkupWithSession(ctx, sessionData, accountID, chatID, messageID, keyboard)
	})
	if ok {
		return err
	}
	return p.botAPI.EditReplyMarkup(ctx, chatID, messageID, keyboard)
}

func (p *PremiumExecutor) SendSticker(
	ctx context.Context,
	chatID int64,
	stickerID string,
) error {
	return p.botAPI.SendSticker(ctx, chatID, stickerID)
}

func (p *PremiumExecutor) DeleteMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
) error {
	return p.botAPI.DeleteMessage(ctx, chatID, messageID)
}

func (p *PremiumExecutor) SendMessage(
	ctx context.Context,
	chatID int64,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	ok, err := p.tryAdminSession(ctx, func(sessionData []byte, accountID string) error {
		return p.mtproto.SendMessageWithSession(ctx, sessionData, accountID, chatID, text, parseMode, keyboard, opts)
	})
	if ok {
		return err
	}
	return p.botAPI.SendMessage(ctx, chatID, text, parseMode, keyboard, opts)
}

// compile-time check
var _ TelegramExecutor = (*PremiumExecutor)(nil)
