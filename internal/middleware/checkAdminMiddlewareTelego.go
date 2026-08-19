package middleware

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func CheckAdminMiddlewareTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		userID := GetUpdateUserIDTelego(update)

		// Bypass para o owner
		if userID == config.OwnerID {
			return ctx.Next(update)
		}

		user, err := c.UserService.GetUserByID(context.Background(), userID)
		if err != nil || user == nil || !user.IsAdmin {
			// Se for um callback, avisa que não tem permissão
			if update.CallbackQuery != nil {
				_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
					CallbackQueryID: update.CallbackQuery.ID,
					Text:            "⛔ Acesso restrito a administradores.",
					ShowAlert:       true,
				})
			}
			return nil
		}

		return ctx.Next(update)
	}
}
