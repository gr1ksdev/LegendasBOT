package mychannel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func ConfigHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		userID := update.CallbackQuery.From.ID
		bot := ctx.Bot()

		callbackData := update.CallbackQuery.Data
		parts := strings.Split(callbackData, ":")
		if len(parts) != 2 {
			logger.Warn("BOT", "Callback invalido: %s", callbackData)
			return nil
		}

		channelIdString := parts[1]
		channelId, err := strconv.ParseInt(channelIdString, 10, 64)
		if err != nil {
			logger.Error("BOT", "Error parsing channelId: %v", err)
			return nil
		}
		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userID, channelId)
		if err != nil {
			logger.Error("BOT", "Erro ao buscar canal: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Este canal não está vinculado ao bot!",
				ShowAlert:       true,
			})
			return nil
		}

		userIDStr := fmt.Sprintf("%d", userID)

		data := map[string]string{
			"title":     channel.Title,
			"channelId": channelIdString,
			"webAppUrl": auth.GenerateMiniAppUrl(userIDStr, channelIdString),
		}

		// Verificar se usuario tem acesso premium (assinatura ativa ou conta conectada)
		// E se o sistema premium esta habilitado globalmente
		premiumEnabled := c.PremiumFeatureService.IsPremiumEnabled(context.Background())
		hasPremium := premiumEnabled && c.HasPremiumAccess(context.Background(), userID)

		templateName := "config-channel"
		if hasPremium {
			templateName = "config-channel-premium"
		}
		text, kb := parser.GetMessageTelego(templateName, data)

		err = c.CacheService.SetSelectedChannel(context.Background(), userID, channelId)
		if err != nil {
			logger.Error("BOT", "Erro ao criar sessão: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
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

func GroupChannelHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		userID := update.CallbackQuery.From.ID
		bot := ctx.Bot()

		callbackData := update.CallbackQuery.Data
		parts := strings.Split(callbackData, ":")
		if len(parts) != 2 {
			logger.Warn("BOT", "Callback invalido: %s", callbackData)
			return nil
		}

		channelIdString := parts[1]
		channelId, err := strconv.ParseInt(channelIdString, 10, 64)
		if err != nil {
			logger.Error("BOT", "Error parsing channelId: %v", err)
			return nil
		}
		_, err = c.ChannelService.GetChannelByTwoID(context.Background(), userID, channelId)
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
			"channelId": channelIdString,
		}
		text, kb := parser.GetMessageTelego("ask-gc-message", data)

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
