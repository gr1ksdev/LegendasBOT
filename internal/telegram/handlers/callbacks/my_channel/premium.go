package mychannel

import (
	"context"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// PremiumFeaturesHandlerTelego mostra a tela de recursos premium.
// Callback: "premium-features:{channelId}"
func PremiumFeaturesHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userID := update.CallbackQuery.From.ID

		// Extrair channelId do callback "premium-features:CHANNEL_ID"
		callbackData := update.CallbackQuery.Data
		parts := strings.Split(callbackData, ":")
		if len(parts) != 2 {
			logger.Warn("BOT", "Callback premium-features invalido: %s", callbackData)
			return nil
		}
		channelIDStr := parts[1]

		// Verificar se o canal pertence ao usuario
		channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err != nil {
			logger.Error("BOT", "Error parsing channelId: %v", err)
			return nil
		}
		_, err = c.ChannelService.GetChannelByTwoID(context.Background(), userID, channelID)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal nao encontrado ou nao pertence a voce!",
				ShowAlert:       true,
			})
			return nil
		}

		// Verificar se premium esta habilitado globalmente
		premiumEnabled := c.PremiumFeatureService.IsPremiumEnabled(context.Background())
		if !premiumEnabled {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⭐ O sistema premium está desativado no momento.",
				ShowAlert:       true,
			})
			return nil
		}

		// Verificar acesso premium
		hasPremium := c.HasPremiumAccess(context.Background(), userID)

		if !hasPremium {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⭐ Voce precisa de uma assinatura premium para acessar este recurso!",
				ShowAlert:       true,
			})
			return nil
		}

		data := map[string]string{
			"channelId": channelIDStr,
		}
		text, kb := parser.GetMessageTelego("premium-features", data)

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
