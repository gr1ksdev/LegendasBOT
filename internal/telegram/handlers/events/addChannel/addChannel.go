package addchannel

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/telegram/logs"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func SendAddChannelPromptTelego(bot *telego.Bot, fromID, chatID int64, chatTitle, firstName string) error {
	logger.Bot("AskAddChannel: Solicitação para o canal %d pelo usuário %d", chatID, fromID)

	data := map[string]string{
		"channelName": chatTitle,
		"channelId":   fmt.Sprintf("%d", chatID),
		"firstName":   firstName,
	}

	text, _ := parser.GetMessageTelego("toadd-require-message", data)
	kb := &telego.InlineKeyboardMarkup{
		InlineKeyboard: [][]telego.InlineKeyboardButton{
			{
				{Text: "✅ Sim", CallbackData: fmt.Sprintf("add-yes:%d", chatID), Style: "success"},
				{Text: "❌ Não", CallbackData: fmt.Sprintf("add-not:%d", chatID), Style: "danger"},
			},
		},
	}

	_, err := bot.SendMessage(context.Background(), &telego.SendMessageParams{
		ChatID:      telego.ChatID{ID: fromID},
		Text:        text,
		ReplyMarkup: kb,
		ParseMode:   telego.ModeHTML,
	})
	return err
}

func AskAddChannelHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.MyChatMember == nil {
			return nil
		}

		chatID := update.MyChatMember.Chat.ID
		fromID := update.MyChatMember.From.ID
		chatTitle := update.MyChatMember.Chat.Title
		firstName := update.MyChatMember.From.FirstName

		if chatID == 0 {
			return nil
		}

		// Verificar se o canal já existe no banco
		existing, _ := c.ChannelService.GetChannelByID(context.Background(), chatID)
		if existing != nil {
			logger.Bot("AskAddChannel: Canal %d já existe no banco.", chatID)
			return nil
		}

		return SendAddChannelPromptTelego(ctx.Bot(), fromID, chatID, chatTitle, firstName)
	}
}

func UpdateChannelInfoHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.MyChatMember == nil {
			return nil
		}

		chatID := update.MyChatMember.Chat.ID
		channel, err := c.ChannelService.GetChannelByID(context.Background(), chatID)
		if err != nil {
			return nil
		}

		logger.Bot("UpdateChannelInfo: Atualizando metadados proativos para o canal %d", chatID)

		// Reutiliza a lógica de syncMetadata
		titleChanged := update.MyChatMember.Chat.Title != "" && update.MyChatMember.Chat.Title != channel.Title
		usernameURL := utils.NormalizeTelegramURL("@" + update.MyChatMember.Chat.Username)
		channelURL := utils.NormalizeTelegramURL(channel.InviteURL)
		usernameChanged := update.MyChatMember.Chat.Username != "" && usernameURL != channelURL && !strings.HasPrefix(channelURL, "https://t.me/+")

		if titleChanged || usernameChanged {
			logger.Bot("Metadados do canal %d mudaram, sincronização agendada.", chatID)
		}

		return nil
	}
}

func AddYesHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		bot := ctx.Bot()
		data := update.CallbackQuery.Data
		chatIDStr := strings.TrimPrefix(data, "add-yes:")
		chatID, _ := strconv.ParseInt(chatIDStr, 10, 64)

		logger.Bot("AddYes: Iniciando vínculo do canal %d", chatID)

		// Buscar informações completas do chat
		chat, err := bot.GetChat(context.Background(), &telego.GetChatParams{ChatID: telego.ChatID{ID: chatID}})
		if err != nil {
			logger.Error("BOT", "Erro ao buscar chat %d: %v", chatID, err)

			text, kb := parser.GetMessageTelego("toadd-notfound-permissions-message", nil)
			_, _ = bot.EditMessageText(context.Background(), &telego.EditMessageTextParams{
				ChatID:      update.CallbackQuery.Message.GetChat().ChatID(),
				MessageID:   update.CallbackQuery.Message.GetMessageID(),
				Text:        text,
				ReplyMarkup: kb,
				ParseMode:   telego.ModeHTML,
			})

			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao obter dados do canal.",
				ShowAlert:       true,
			})
			return nil
		}

		inviteURL := chat.InviteLink
		if chat.Username != "" {
			inviteURL = "@" + chat.Username
		}

		// Buscar legendas globais das configurações do servidor
		serverConfig, _ := c.ServerService.GetConfig(context.Background())
		globalDefault := ""
		globalNewPack := ""
		if serverConfig != nil {
			globalDefault = serverConfig.GlobalDefaultCaption
			globalNewPack = serverConfig.GlobalNewPackCaption
		}

		// Substituir placeholders dinâmicos (como username do bot)
		botInfo, _ := bot.GetMe(context.Background())
		if botInfo != nil {
			globalDefault = strings.ReplaceAll(globalDefault, "{usernameBot}", botInfo.Username)
			globalNewPack = strings.ReplaceAll(globalNewPack, "{usernameBot}", botInfo.Username)
		}

		channel, err := c.ChannelService.CreateChannelWithDefaults(context.Background(), chatID, chat.Title, inviteURL, globalNewPack, globalDefault, update.CallbackQuery.From.ID)
		if err != nil {
			logger.Error("BOT", "Erro ao vincular canal: %v", err)

			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao vincular canal no banco de dados.",
				ShowAlert:       true,
			})
			return nil
		}

		// Preparar variáveis para a mensagem de sucesso
		vars := map[string]string{
			"botId":       fmt.Sprintf("%d", botInfo.ID),
			"botUsername": botInfo.Username,
			"firstName":   update.CallbackQuery.From.FirstName,
			"miniAppUrl":  auth.GenerateMiniAppUrl(fmt.Sprintf("%d", update.CallbackQuery.From.ID), chatIDStr),
			"channelId":   chatIDStr,
			"channelName": chat.Title,
		}

		text, kb := parser.GetMessageTelego("toadd-success-message", vars)

		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			MessageID: update.CallbackQuery.Message.GetMessageID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}

		msg, err := bot.EditMessageText(context.Background(), params)

		if err == nil {
			_ = bot.SetMessageReaction(context.Background(), &telego.SetMessageReactionParams{
				ChatID:    msg.GetChat().ChatID(),
				MessageID: msg.GetMessageID(),
				Reaction: []telego.ReactionType{
					&telego.ReactionTypeEmoji{
						Type:  telego.ReactionEmoji,
						Emoji: "🎉",
					},
				},
				IsBig: true,
			})
		}

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "✅ Canal vinculado com sucesso!",
		})

		logs.LogAdminTelego(bot, channel)

		return nil
	}
}

func AddNotHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		bot := ctx.Bot()
		text, kb := parser.GetMessageTelego("toadd-cancel-message", nil)

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
			Text:            "Operação cancelada.",
		})

		return nil
	}
}
