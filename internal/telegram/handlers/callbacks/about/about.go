package about

import (
	"context"
	"fmt"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		user, _ := bot.GetMe(context.Background())
		text, kb := parser.GetMessageTelego("about", map[string]string{
			"ownerUser":  "@SuporteLegendas",
			"botVersion": utils.Version,
			"botId":      fmt.Sprintf("%d", user.ID),
		})

		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			MessageID: update.CallbackQuery.Message.GetMessageID(),
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		_, err := bot.EditMessageText(context.Background(), params)
		if err != nil {
			logger.Error("CALLBACK", "Error editing message: %v", err)
		}

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		return nil
	}
}
