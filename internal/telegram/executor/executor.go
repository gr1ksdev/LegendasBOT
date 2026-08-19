// Package executor define a interface TelegramExecutor que abstrai
// as operacoes de edicao/envio no Telegram.
//
// Existem duas implementacoes:
//   - BotAPIExecutor: usa a Telegram Bot API via telego (existente)
//   - MTProtoExecutor: usa o protocolo MTProto via gotd/td (contas conectadas)
//
// A decisao de qual implementacao usar e feita pela factory, com base
// na presenca de uma conta conectada ativa para o usuario.
package executor

import (
	"context"
)

// EditOptions agrupa opcoes opcionais para operacoes de edicao.
type EditOptions struct {
	DisableLinkPreview bool
	// Entities contem o JSON dos MessageEntityDTO a serem aplicados via MTProto.
	// Quando preenchido, o parseMode e ignorado e as entities sao usadas diretamente.
	// Formato: [{"type":"bold","offset":0,"length":5,"url":"",...}]
	Entities string
}

// TelegramExecutor define o contrato para execucao de acoes no Telegram.
// Todas as operacoes disponiveis atualmente no pipeline de channel posts
// devem estar representadas aqui.
type TelegramExecutor interface {
	// EditMessage edita o texto de uma mensagem existente.
	EditMessage(ctx context.Context, chatID int64, messageID int, text string, parseMode string, keyboard *InlineKeyboardMarkup, opts *EditOptions) error

	// EditCaption edita a legenda de uma mensagem de midia existente.
	EditCaption(ctx context.Context, chatID int64, messageID int, caption string, parseMode string, keyboard *InlineKeyboardMarkup, opts *EditOptions) error

	// EditReplyMarkup edita apenas o teclado inline de uma mensagem.
	EditReplyMarkup(ctx context.Context, chatID int64, messageID int, keyboard *InlineKeyboardMarkup) error

	// SendSticker envia um sticker separador para o canal.
	SendSticker(ctx context.Context, chatID int64, stickerID string) error

	// DeleteMessage deleta uma mensagem do canal.
	DeleteMessage(ctx context.Context, chatID int64, messageID int) error

	// SendMessage envia uma nova mensagem de texto para o canal.
	SendMessage(ctx context.Context, chatID int64, text string, parseMode string, keyboard *InlineKeyboardMarkup, opts *EditOptions) error
}

// InlineKeyboardMarkup representa o teclado inline do Telegram.
// E uma representacao generica para nao depender de telego ou gotd.
type InlineKeyboardMarkup struct {
	InlineKeyboard [][]InlineKeyboardButton
}

type InlineKeyboardButton struct {
	Text         string
	URL          string
	CallbackData string
	Style        string
}

// NewEmptyKeyboard cria um InlineKeyboardMarkup vazio.
func NewEmptyKeyboard() *InlineKeyboardMarkup {
	return &InlineKeyboardMarkup{
		InlineKeyboard: make([][]InlineKeyboardButton, 0),
	}
}
