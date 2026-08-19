package admin

import (
	"context"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func NoticeCommandHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil {
			return nil
		}

		text := update.Message.Text
		if text == "" {
			return nil
		}

		lines := strings.Split(text, "\n")
		if len(lines) <= 1 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ A mensagem de aviso está vazia.",
			})
			return nil
		}

		noticeText := strings.TrimSpace(strings.Join(lines[1:], "\n"))

		users, err := app.UserService.GetAllUsers(context.Background())
		if err != nil || len(users) == 0 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Nenhum usuário encontrado.",
			})
			return nil
		}

		var failedUsers []string
		sentCount := 0

		for _, user := range users {
			_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: user.UserId},
				Text:      noticeText,
				ParseMode: telego.ModeHTML,
			})
			if err != nil {
				logger.Error("ADMIN", "Erro ao enviar aviso para %d - %s: %v", user.UserId, user.FirstName, err)
				failedUsers = append(failedUsers, fmt.Sprintf("<code>%d</code> - %s", user.UserId, html.EscapeString(user.FirstName)))
			} else {
				sentCount++
			}
		}

		var finalMsg strings.Builder
		finalMsg.WriteString(fmt.Sprintf("📨 Aviso enviado para <b>%d</b> usuários.\n", sentCount))

		if len(failedUsers) > 0 {
			finalMsg.WriteString(fmt.Sprintf("\n❌ Falhou para %d usuários:\n", len(failedUsers)))
			finalMsg.WriteString(strings.Join(failedUsers, "\n"))
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      finalMsg.String(),
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		return nil
	}
}

func NoticeChannelsHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		user, _ := bot.GetMe(context.Background())

		data := map[string]string{
			"botUsername": user.Username,
		}

		text, kb := parser.GetMessageTelego("publi", data)

		channels, err := app.ChannelService.GetAllChannels(context.Background())
		if err != nil || len(channels) == 0 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Nenhum canal encontrado.",
			})
			return nil
		}

		var failedChannels []string
		sentCount := 0

		for _, ch := range channels {
			_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:      telego.ChatID{ID: ch.ID},
				Text:        text,
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: kb,
			})
			if err != nil {
				logger.Error("ADMIN", "Erro ao enviar aviso para canal %d - %s: %v", ch.ID, ch.Title, err)
				failedChannels = append(failedChannels, fmt.Sprintf("<code>%d</code> - %s", ch.ID, html.EscapeString(ch.Title)))
			} else {
				sentCount++
			}
		}

		var resultMsg strings.Builder
		resultMsg.WriteString(fmt.Sprintf("📨 Aviso enviado para <b>%d</b> canais.\n", sentCount))

		if len(failedChannels) > 0 {
			resultMsg.WriteString(fmt.Sprintf("\n❌ Falhou para %d canais:\n", len(failedChannels)))
			resultMsg.WriteString(strings.Join(failedChannels, "\n"))
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      resultMsg.String(),
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		return nil
	}
}

func SendMessageToIdHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil || update.Message.Text == "" {
			return nil
		}

		lines := strings.Split(update.Message.Text, "\n")
		if len(lines) < 2 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Uso inválido. Envie no formato:\n/send <id>\n<mensagem>",
			})
			return nil
		}

		idStr := strings.TrimSpace(strings.TrimPrefix(lines[0], "/send"))
		targetID, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   fmt.Sprintf("❌ ID inválido: %v", err),
			})
			return nil
		}

		message := strings.Join(lines[1:], "\n")
		message = strings.TrimSpace(message)

		if message == "" {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Mensagem vazia.",
			})
			return nil
		}

		_, err = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: targetID},
			Text:      message,
			ParseMode: telego.ModeHTML,
		})
		if err != nil {
			logger.Error("ADMIN", "Erro ao enviar mensagem para %d: %v", targetID, err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      fmt.Sprintf("❌ Falha ao enviar para <code>%d</code>: %v", targetID, err),
				ParseMode: telego.ModeHTML,
			})
			return nil
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      fmt.Sprintf("✅ Mensagem enviada para <code>%d</code> com sucesso.", targetID),
			ParseMode: telego.ModeHTML,
		})
		return nil
	}
}

func NoticeUsersReplyHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil {
			return nil
		}

		if update.Message.ReplyToMessage == nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Responda a uma message para enviar o aviso aos usuários.",
			})
			return nil
		}

		noticeText := update.Message.ReplyToMessage.Text
		if noticeText == "" {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ A mensagem respondida está vazia.",
			})
			return nil
		}

		users, err := app.UserService.GetAllUsers(context.Background())
		if err != nil || len(users) == 0 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Nenhum usuário encontrado.",
			})
			return nil
		}

		var failedUsers []string
		sentCount := 0

		for _, user := range users {
			_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: user.UserId},
				Text:      noticeText,
				ParseMode: telego.ModeHTML,
			})
			if err != nil {
				logger.Error("ADMIN", "Erro ao enviar aviso para %d - %s: %v", user.UserId, user.FirstName, err)
				failedUsers = append(failedUsers, fmt.Sprintf("<code>%d</code> - %s", user.UserId, html.EscapeString(user.FirstName)))
			} else {
				sentCount++
			}
		}

		var finalMsg strings.Builder
		finalMsg.WriteString(fmt.Sprintf("📨 Aviso enviado para <b>%d</b> usuários.\n", sentCount))

		if len(failedUsers) > 0 {
			finalMsg.WriteString(fmt.Sprintf("\n❌ Falhou para %d usuários:\n", len(failedUsers)))
			finalMsg.WriteString(strings.Join(failedUsers, "\n"))
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      finalMsg.String(),
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		return nil
	}
}

func NoticeChannelsReplyHandlerTelego(app *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		bot := ctx.Bot()
		if update.Message == nil {
			return nil
		}

		if update.Message.ReplyToMessage == nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Responda a uma mensagem para enviar o aviso aos canais.",
			})
			return nil
		}

		noticeText := update.Message.ReplyToMessage.Text
		if noticeText == "" {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ A mensagem respondida está vazia.",
			})
			return nil
		}

		channels, err := app.ChannelService.GetAllChannels(context.Background())
		if err != nil || len(channels) == 0 {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: update.Message.Chat.ChatID(),
				Text:   "❌ Nenhum canal encontrado.",
			})
			return nil
		}

		var failedChannels []string
		sentCount := 0

		for _, ch := range channels {
			_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    telego.ChatID{ID: ch.ID},
				Text:      noticeText,
				ParseMode: telego.ModeHTML,
			})
			if err != nil {
				logger.Error("ADMIN", "Erro ao enviar aviso para canal %d - %s: %v", ch.ID, ch.Title, err)
				failedChannels = append(failedChannels, fmt.Sprintf("<code>%d</code> - %s", ch.ID, html.EscapeString(ch.Title)))
			} else {
				sentCount++
			}
		}

		var resultMsg strings.Builder
		resultMsg.WriteString(fmt.Sprintf("📨 Aviso enviado para <b>%d</b> canais.\n", sentCount))

		if len(failedChannels) > 0 {
			resultMsg.WriteString(fmt.Sprintf("\n❌ Falhou para %d canais:\n", len(failedChannels)))
			resultMsg.WriteString(strings.Join(failedChannels, "\n"))
		}

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      resultMsg.String(),
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		return nil
	}
}
