package mychannel

import (
	"context"
	"fmt"

	"github.com/leirbagxis/FreddyBot/internal/container"
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

		userID := update.CallbackQuery.From.ID
		bot := ctx.Bot()

		channels, err := c.ChannelService.GetUserChannels(context.Background(), userID)
		if err != nil || len(channels) == 0 {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Você ainda não possui nenhum canal vinculado.",
				ShowAlert:       true,
			})
			return nil
		}

		text, _ := parser.GetMessageTelego("profile-user-channels", map[string]string{})

		var finalButtons [][]parser.Button

		// Adiciona os botões dinâmicos
		for _, channel := range channels {
			finalButtons = append(finalButtons, []parser.Button{
				{
					Text:         channel.Title,
					CallbackData: fmt.Sprintf("config:%d", channel.ID),
				},
			})
		}

		// Recarrega botões padrão do parser
		_, homeButtons := parser.GetMessageTelego("profile-user-channels", map[string]string{})
		if homeButtons != nil {
			for _, row := range homeButtons.InlineKeyboard {
				var newRow []parser.Button
				for _, btn := range row {
					cbData := ""
					if btn.CallbackData != "" {
						cbData = btn.CallbackData
					}
					url := ""
					if btn.URL != "" {
						url = btn.URL
					}

					newRow = append(newRow, parser.Button{
						Text:         btn.Text,
						CallbackData: cbData,
						URL:          url,
					})
				}
				finalButtons = append(finalButtons, newRow)
			}
		}

		replyMarkup := parser.BuildInlineKeyboardTelego(finalButtons)

		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID: update.CallbackQuery.Message.GetMessageID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
		}
		if replyMarkup != nil {
			params.ReplyMarkup = replyMarkup
		}
		_, err = bot.EditMessageText(context.Background(), params)

		if err != nil {
			logger.Error("BOT", "Erro ao editar mensagem: %v", err)
		}

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		return nil
	}
}
