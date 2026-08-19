package help

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		user, _ := bot.GetMe(context.Background())

		text, kb := parser.GetMessageTelego("help", map[string]string{
			"botUsername": "@" + user.Username,
			"botUser":     user.Username,
		})

		params := &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, _ = bot.SendMessage(context.Background(), params)

		return nil
	}
}

func CallbackHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		user, _ := bot.GetMe(context.Background())

		text, kb := parser.GetMessageTelego("help", map[string]string{
			"botUsername": "@" + user.Username,
			"botUser":     user.Username,
		})

		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID: update.CallbackQuery.Message.GetMessageID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, _ = bot.EditMessageText(context.Background(), params)

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		return nil
	}
}
