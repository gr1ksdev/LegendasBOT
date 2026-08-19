package middleware

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func CheckMaintenanceMiddlewareTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, upt telego.Update) error {
		if upt.ChannelPost != nil || upt.EditedChannelPost != nil {
			return ctx.Next(upt)
		}

		// 1. Get maintenance status first
		maintenance, err := c.ServerService.GetMaintenance(context.Background())
		if err != nil {
			logger.Error("MID", "Erro ao buscar status de manutenção: %v", err)
			return ctx.Next(upt)
		}

		// 2. If not in maintenance, allow everything
		if !maintenance {
			return ctx.Next(upt)
		}

		// 3. If in maintenance, check if user is admin or owner
		userID := GetUpdateUserIDTelego(upt)
		if userID != 0 {
			// Owner sempre passa
			if userID == config.OwnerID {
				return ctx.Next(upt)
			}

			// Admins também passam
			user, err := c.UserService.GetUserByID(context.Background(), userID)
			if err == nil && user != nil && user.IsAdmin {
				return ctx.Next(upt)
			}
		}

		// 4. Send maintenance response
		sendMaintenanceResponseTelego(ctx, upt, userID)
		return nil
	}
}

func sendMaintenanceResponseTelego(ctx *telegohandler.Context, upt telego.Update, userID int64) {
	b := ctx.Bot()
	switch {
	case upt.Message != nil:
		_, _ = b.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: userID},
			Text:      "⚙️ <b>Manutenção em andamento</b>\n\nO bot está temporariamente indisponível para melhorias no sistema.\n\n⏳ <i>Voltaremos em breve.</i>\n\nObrigado pela sua paciência 💙",
			ParseMode: telego.ModeHTML,
		})

	case upt.CallbackQuery != nil:
		_ = b.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: upt.CallbackQuery.ID,
			Text:            "🚧 O bot está em manutenção no momento.",
			ShowAlert:       true,
			CacheTime:       0,
		})

	case upt.InlineQuery != nil:
		_ = b.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
			InlineQueryID: upt.InlineQuery.ID,
			Results: []telego.InlineQueryResult{
				&telego.InlineQueryResultArticle{
					Type:  "article",
					ID:    "maintenance",
					Title: "⚙️ Manutenção em andamento",
					InputMessageContent: &telego.InputTextMessageContent{
						MessageText: "O bot está em manutenção no momento. Tente novamente mais tarde.",
					},
				},
			},
			CacheTime: 0,
		})
	}
}
