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

func AskDeleteChannelHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetSelectedChannel(context.Background(), userId)
		if err != nil {
			logger.Error("BOT", "Erro ao pegar sessão: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			logger.Error("BOT", "Erro ao buscar canal: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		data := map[string]string{
			"title":     channel.Title,
			"channelId": fmt.Sprintf("%d", session),
		}
		c.CacheService.SetDeleteChannel(context.Background(), userId, session)

		text, kb := parser.GetMessageTelego("del", data)
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

func ConfirmDeleteChannelHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetDeleteChannel(context.Background(), userId)
		if err != nil {
			logger.Error("BOT", "Erro ao criar sessão: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			logger.Error("BOT", "Erro ao buscar canal: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		err = c.ChannelService.DisconnectChannel(context.Background(), userId, session)
		if err != nil {
			logger.Error("BOT", "Erro ao excluir canal: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao excluir canal. Tente novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		data := map[string]string{
			"title":     channel.Title,
			"channelId": fmt.Sprintf("%d", session),
		}

		text, kb := parser.GetMessageTelego("confirm-del", data)
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
			Text:            "✅ Canal excluido com sucesso!",
		})

		return nil
	}
}
