package suporte

import (
	"context"
	"fmt"
	"html"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		bot := ctx.Bot()
		textInput := update.Message.Text
		command := "/ouvidoria"

		message := strings.TrimSpace(textInput[len(command):])

		if len(message) <= 0 {
			text, kb := parser.GetMessageTelego("support-usage", map[string]string{})
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:      update.Message.Chat.ChatID(),
				Text:        text,
				ReplyMarkup: kb,
				ParseMode:   telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		firstName := html.EscapeString(utils.RemoveHTMLTags(update.Message.From.FirstName))
		safeMessage := html.EscapeString(message)

		user, _ := bot.GetMe(context.Background())

		data := map[string]string{
			"firstName":   firstName,
			"userId":      fmt.Sprintf("%d", update.Message.From.ID),
			"botId":       fmt.Sprintf("%d", user.ID),
			"userMessage": safeMessage,
		}

		adminText, _ := parser.GetMessageTelego("support-msg-admin", data)
		_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: config.OwnerID},
			Text:      adminText,
			ParseMode: telego.ModeHTML,
		})
		if err != nil {
			logger.Error("BOT", "Erro ao enviar mensagem de ouvidoria pro admin: %v", err)
		}

		text, kb := parser.GetMessageTelego("support-sent", data)
		params := &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		_, err = bot.SendMessage(context.Background(), params)
		if err != nil {
			logger.Error("BOT", "Erro ao enviar confirmação de ouvidoria pro usuário %d: %v", update.Message.From.ID, err)
		}

		return nil
	}
}
