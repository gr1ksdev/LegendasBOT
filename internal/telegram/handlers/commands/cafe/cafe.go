package cafe

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// HandlerTelego responde aos comandos /cafe e /doacao.
func HandlerTelego(_ *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		Send(ctx.Bot(), update.Message.Chat.ChatID(), update.Message.MessageID)
		return nil
	}
}

// Send envia a mensagem de doação para o chat informado.
func Send(bot *telego.Bot, chatID telego.ChatID, replyTo int) {
	text, kb := parser.GetMessageTelego("cafe", map[string]string{
		"botUsername": bot.Username(),
	})

	params := &telego.SendMessageParams{
		ChatID:    chatID,
		Text:      text,
		ParseMode: telego.ModeHTML,
	}
	if replyTo != 0 {
		params.ReplyParameters = &telego.ReplyParameters{MessageID: replyTo}
	}
	if kb != nil {
		params.ReplyMarkup = kb
	}

	if _, err := bot.SendMessage(context.Background(), params); err != nil {
		logger.Error("BOT", "erro ao enviar mensagem de doação: %v", err)
	}
}
