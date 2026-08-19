package tutorial

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		bot := ctx.Bot()
		chatID := update.Message.From.ID

		topic, err := bot.CreateForumTopic(context.Background(), &telego.CreateForumTopicParams{
			ChatID:            telego.ChatID{ID: chatID},
			Name:              "📚 Tutoriais",
			IconCustomEmojiID: "5334882760735598374",
		})
		if err != nil {
			logger.Error("BOT", "Erro ao criar tópico de tutorial: %v", err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID: telego.ChatID{ID: chatID},
				Text:   "Não consegui criar a aba de tutoriais aqui no PV. Verifique se tópicos no privado estão habilitados para este bot.",
			})
			return nil
		}

		threadID := topic.MessageThreadID

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:          telego.ChatID{ID: chatID},
			MessageThreadID: threadID,
			Text:            "Bem-vindo ao tutorial! Vou te mandar os vídeos aqui 👇",
		})

		return nil
	}
}
