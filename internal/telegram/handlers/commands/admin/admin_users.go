package admin

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func GetAllUsersHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		const chunkSize = 50
		offset := 0
		bot := ctx.Bot()

		for {
			users, total, err := app.UserService.GetAllUsersPaginated(context.Background(), chunkSize, offset)
			if err != nil {
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID: update.Message.Chat.ChatID(),
					Text:   "Erro ao buscar usuários.",
				})
				return nil
			}

			if len(users) == 0 {
				if offset == 0 {
					_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
						ChatID: update.Message.Chat.ChatID(),
						Text:   "Nenhum usuário encontrado.",
					})
				}
				break
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("👥 Total de Usuários: <b>%d</b>\n<blockquote>Página %d</blockquote>\n",
				total, (offset/chunkSize)+1))

			for _, u := range users {
				sb.WriteString(fmt.Sprintf("<a href='tg://user?id=%d'>%s</a> - <code>%d</code>\n", u.UserId, html.EscapeString(u.FirstName), u.UserId))
			}

			_, err = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      sb.String(),
				ParseMode: telego.ModeHTML,
			})
			if err != nil {
				break
			}

			offset += chunkSize
			if int64(offset) >= total {
				break
			}
		}
		return nil
	}
}

func GetInfoUserHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		userIDStr := strings.TrimSpace(update.Message.Text[len("/user"):])
		userID, err := strconv.ParseInt(userIDStr, 10, 64)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ ID inválido: %v", err),
			})
			return nil
		}

		user, err := app.UserService.GetUserByID(context.Background(), userID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ Usuário não encontrado!: %v", err),
			})
			return nil
		}

		channels, _ := app.ChannelService.GetUserChannels(context.Background(), user.UserId)
		header := fmt.Sprintf("👤 <b><a href='tg://user?id=%d'>%s</a></b> (<code>%d</code>)\n📦 Canais: <b>%d</b>\n\n",
			user.UserId,
			html.EscapeString(user.FirstName),
			user.UserId,
			len(channels),
		)

		if len(channels) == 0 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      header + "Usuário ainda não possui canais.",
				ParseMode: telego.ModeHTML,
			})
			return nil
		}

		const chunkSize = 20
		for i := 0; i < len(channels); i += chunkSize {
			chunk := channels[i:utils.MinInt(i+chunkSize, len(channels))]
			var lines []string
			for _, c := range chunk {
				lines = append(lines, fmt.Sprintf("<a href='%s'>%s</a> - <code>%d</code>",
					c.InviteURL,
					html.EscapeString(c.Title),
					c.ID,
				))
			}

			msg := header + strings.Join(lines, "\n")
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      msg,
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
		}
		return nil
	}
}

func SetAdminHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, upt telego.Update) error {
		bot := ctx.Bot()
		if upt.Message == nil {
			return nil
		}

		args := strings.Fields(upt.Message.Text)
		if len(args) < 2 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    upt.Message.Chat.ChatID(),
				Text:      "❌ Uso correto:\n<code>/setadmin [userID]</code>",
				ParseMode: telego.ModeHTML,
			})
			return nil
		}

		userID, err := strconv.ParseInt(args[1], 10, 64)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: upt.Message.Chat.ChatID(),
				Text:   "❌ userID inválido.",
			})
			return nil
		}

		isAdmin, err := app.UserService.UpdateUserAdmin(context.Background(), userID)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: upt.Message.Chat.ChatID(),
				Text:   "❌ Erro ao alterar status de admin.",
			})
			return nil
		}

		var msg string
		if isAdmin {
			msg = fmt.Sprintf("✅ Usuário <code>%d</code> agora é administrador.", userID)
		} else {
			msg = fmt.Sprintf("⚠️ Usuário <code>%d</code> não é mais administrador.", userID)
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    upt.Message.Chat.ChatID(),
			Text:      msg,
			ParseMode: telego.ModeHTML,
		})
		return nil
	}
}
