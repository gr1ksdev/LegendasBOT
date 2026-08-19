package executor

import (
	"context"
)

// UserExecutor e um TelegramExecutor vinculado a um usuario especifico.
// Quando o executor interno e um MTProtoExecutor, usa metodos user-bound.
// Caso contrario, delega diretamente para o BotAPIExecutor.
type UserExecutor struct {
	userID  int64
	mtproto *MTProtoExecutor // pode ser nil
	botAPI  TelegramExecutor // sempre disponivel
}

// NewUserExecutor cria um wrapper que vincula um usuario a um executor.
// botAPI e sempre necessario; mtproto e opcional.
func NewUserExecutor(userID int64, botAPI TelegramExecutor, mtproto *MTProtoExecutor) *UserExecutor {
	return &UserExecutor{
		userID:  userID,
		botAPI:  botAPI,
		mtproto: mtproto,
	}
}

// HasMTProto retorna true se este executor tem suporte MTProto.
func (u *UserExecutor) HasMTProto() bool {
	return u.mtproto != nil
}

// UserID retorna o ID do usuario vinculado.
func (u *UserExecutor) UserID() int64 {
	return u.userID
}

// --- Implementacao de TelegramExecutor ---

func (u *UserExecutor) EditMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	if u.mtproto != nil {
		return u.mtproto.EditMessageForUser(ctx, u.userID, chatID, messageID, text, parseMode, keyboard, opts)
	}
	return u.botAPI.EditMessage(ctx, chatID, messageID, text, parseMode, keyboard, opts)
}

func (u *UserExecutor) EditCaption(
	ctx context.Context,
	chatID int64,
	messageID int,
	caption string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	if u.mtproto != nil {
		return u.mtproto.EditCaptionForUser(ctx, u.userID, chatID, messageID, caption, parseMode, keyboard, opts)
	}
	return u.botAPI.EditCaption(ctx, chatID, messageID, caption, parseMode, keyboard, opts)
}

func (u *UserExecutor) EditReplyMarkup(
	ctx context.Context,
	chatID int64,
	messageID int,
	keyboard *InlineKeyboardMarkup,
) error {
	if u.mtproto != nil {
		return u.mtproto.EditReplyMarkupForUser(ctx, u.userID, chatID, messageID, keyboard)
	}
	return u.botAPI.EditReplyMarkup(ctx, chatID, messageID, keyboard)
}

func (u *UserExecutor) SendSticker(
	ctx context.Context,
	chatID int64,
	stickerID string,
) error {
	// Stickers sempre via Bot API por simplicidade
	return u.botAPI.SendSticker(ctx, chatID, stickerID)
}

func (u *UserExecutor) DeleteMessage(
	ctx context.Context,
	chatID int64,
	messageID int,
) error {
	return u.botAPI.DeleteMessage(ctx, chatID, messageID)
}

func (u *UserExecutor) SendMessage(
	ctx context.Context,
	chatID int64,
	text string,
	parseMode string,
	keyboard *InlineKeyboardMarkup,
	opts *EditOptions,
) error {
	if u.mtproto != nil {
		return u.mtproto.SendMessageForUser(ctx, u.userID, chatID, text, parseMode, keyboard, opts)
	}
	return u.botAPI.SendMessage(ctx, chatID, text, parseMode, keyboard, opts)
}

// compile-time check
var _ TelegramExecutor = (*UserExecutor)(nil)
