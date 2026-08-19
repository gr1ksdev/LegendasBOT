package profileinfo

import (
	"context"
	"fmt"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
	"gorm.io/gorm"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		userID := update.CallbackQuery.From.ID
		bot := ctx.Bot()

		user, err := c.UserService.GetUserByID(context.Background(), userID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
					MessageID: update.CallbackQuery.Message.GetMessageID(),
					Text:      "❌ Usuário não encontrado no banco de dados.",
				})
				return nil
			}
			logger.Error("BOT", "Erro ao buscar usuário: %v", err)
			return nil
		}

		countChannel, err := c.ChannelService.CountUserChannels(context.Background(), userID)
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
					ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
					MessageID: update.CallbackQuery.Message.GetMessageID(),
					Text:      "❌ Usuário não encontrado no banco de dados.",
				})
				return nil
			}
			logger.Error("BOT", "Erro ao buscar countChannel: %v", err)
			return nil
		}

		data := map[string]string{
			"firstName":    user.FirstName,
			"userId":       fmt.Sprintf("%d", user.UserId),
			"register":     user.CreatedAt.Format("02/01/2006"),
			"countChannel": fmt.Sprintf("%d", countChannel),
		}

		text, kb := parser.GetMessageTelego("profile-info", data)
		if user.IsAdmin {
			adminRow := []telego.InlineKeyboardButton{
				{
					Text: "🛠 Admin Painel",
					WebApp: &telego.WebAppInfo{
						URL: fmt.Sprintf("%s/admin/dash", config.WebAppURL),
					},
				},
			}

			kb.InlineKeyboard = append(kb.InlineKeyboard, adminRow)
		}

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
