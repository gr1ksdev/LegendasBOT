package middleware

import (
	"context"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func CheckBlacklistMiddlewareTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, upt telego.Update) error {
		// Ignorar postagens de canal aqui, pois elas são tratadas no handler de channelPost
		if upt.ChannelPost != nil || upt.EditedChannelPost != nil {
			return ctx.Next(upt)
		}

		userID := GetUpdateUserIDTelego(upt)
		if userID == 0 {
			return ctx.Next(upt)
		}

		// Owner nunca é bloqueado
		if userID == config.OwnerID {
			return ctx.Next(upt)
		}

		user, err := c.UserService.GetUserByID(context.Background(), userID)
		if err != nil || user == nil {
			return ctx.Next(upt)
		}

		if !user.IsBlacklisted {
			return ctx.Next(upt)
		}

		// Se está na blacklist, apenas /start e /ouvidoria são permitidos
		if upt.Message != nil && (strings.HasPrefix(upt.Message.Text, "/start") || strings.HasPrefix(upt.Message.Text, "/ouvidoria")) {
			return ctx.Next(upt)
		}

		// Bloquear outras interações
		if upt.CallbackQuery != nil {
			_ = ctx.Bot().AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: upt.CallbackQuery.ID,
				Text:            "❌ Você está na blacklist e seus comandos estão bloqueados.",
				ShowAlert:       true,
			})
			return nil
		}

		// Para outras mensagens, podemos enviar um aviso silencioso ou apenas ignorar
		return nil
	}
}
