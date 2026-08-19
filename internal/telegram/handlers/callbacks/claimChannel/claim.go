package claimchannel

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

func HandlerTelego(c *container.AppContainer) telegohandler.InlineQueryHandler {
	return func(ctx *telegohandler.Context, inlineQuery telego.InlineQuery) error {
		query := inlineQuery.Query
		from := inlineQuery.From

		cmd := strings.Split(query, " ")
		if len(cmd) == 0 || strings.ToLower(cmd[0]) != "claim" {
			return nil
		}

		bot := ctx.Bot()

		if len(cmd) < 2 {
			article := buildErrorArticleTelego(from.ID, "invalid", "🔎 ID Inválido", "⚠️ O ID deve ser um número válido.")
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results:       []telego.InlineQueryResult{article},
				CacheTime:     0,
			})
			return nil
		}

		channelIdRaw := strings.TrimSpace(cmd[1])
		channelIdRaw = strings.Map(func(r rune) rune {
			if r >= 0x2066 && r <= 0x2069 {
				return -1
			}
			return r
		}, channelIdRaw)

		channelID, err := strconv.ParseInt(channelIdRaw, 10, 64)
		if err != nil {
			article := buildErrorArticleTelego(from.ID, "invalid_format", "🔎 ID Inválido", "⚠️ O ID deve ser um número válido.")
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results:       []telego.InlineQueryResult{article},
				CacheTime:     0,
			})
			return nil
		}

		channel, _ := c.ChannelService.GetChannelByID(context.Background(), channelID)
		if channel == nil {
			article := buildErrorArticleTelego(from.ID, "not_found", "⚠️ Canal não encontrado", "🚫 Este canal não está cadastrado no sistema.")
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results:       []telego.InlineQueryResult{article},
				CacheTime:     0,
			})
			return nil
		}

		admins, err := bot.GetChatAdministrators(context.Background(), &telego.GetChatAdministratorsParams{
			ChatID: telego.ChatID{ID: channelID},
		})
		if err != nil {
			article := buildErrorArticleTelego(from.ID, "admin_error", "⚠️ Erro ao acessar canal", "🚫 O bot pode não estar no canal ou o ID está errado.")
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results:       []telego.InlineQueryResult{article},
				CacheTime:     0,
			})
			return nil
		}

		isOwner := false
		for _, admin := range admins {
			if admin.MemberStatus() == telego.MemberStatusCreator {
				if owner, ok := admin.(*telego.ChatMemberOwner); ok && owner.User.ID == from.ID {
					isOwner = true
					break
				}
			}
		}
		if !isOwner {
			article := buildErrorArticleTelego(from.ID, "not_owner", "🚫 Sem Permissão", fmt.Sprintf("⚠️ Você não é o criador/dono deste canal (%d).", channelID))
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results:       []telego.InlineQueryResult{article},
				CacheTime:     0,
			})
			return nil
		}

		ownerUser, err := c.UserService.GetUserByID(context.Background(), channel.OwnerID)
		if err != nil {
			article := buildErrorArticleTelego(from.ID, "user_error", "❌ Erro ao buscar proprietário", "Erro ao buscar usuário proprietário.")
			_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
				InlineQueryID: inlineQuery.ID,
				Results:       []telego.InlineQueryResult{article},
				CacheTime:     0,
			})
			return nil
		}

		session, err := c.SessionManager.CreateClaimSession(context.Background(), channelID, channel.OwnerID, from.ID)
		if err != nil {
			logger.Error("BOT", "Erro ao criar cache: %v", err)
			return nil
		}

		vars := map[string]string{
			"channelId":   fmt.Sprint(channelID),
			"channelName": channel.Title,
			"ownerId":     fmt.Sprint(ownerUser.UserId),
			"ownerName":   ownerUser.FirstName,
			"sessionKey":  session.Key,
		}

		text, kb := parser.GetMessageTelego("claim-ownership-require-message", vars)

		article := &telego.InlineQueryResultArticle{
			Type:        "article",
			ID:          fmt.Sprintf("claim_success_%d", from.ID),
			Title:       "✅ Canal Encontrado",
			Description: fmt.Sprintf("Canal %d - Confirme a propriedade", channelID),
			InputMessageContent: &telego.InputTextMessageContent{
				MessageText: text,
				ParseMode:   telego.ModeHTML,
			},
			ReplyMarkup: kb,
		}

		_ = bot.AnswerInlineQuery(context.Background(), &telego.AnswerInlineQueryParams{
			InlineQueryID: inlineQuery.ID,
			Results:       []telego.InlineQueryResult{article},
			CacheTime:     0,
		})
		return nil
	}
}

func AcceptClaimHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		bot := ctx.Bot()
		callback := update.CallbackQuery
		from := callback.From

		callbackData := callback.Data
		parts := strings.Split(callbackData, ":")
		if len(parts) != 2 {
			logger.Warn("BOT", "Callback invalido: %s", callbackData)
			return nil
		}

		getSession, err := c.SessionManager.GetChannelSession(context.Background(), parts[1])
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Tempo Esgotado. Faça o processo novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByID(context.Background(), getSession.ChannelID)
		if err != nil {
			logger.Error("BOT", "Erro ao obter info do canal: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Erro ao obter informações do canal!",
				ShowAlert:       true,
			})
			return nil
		}

		_ = c.SessionManager.DeleteChannelSession(context.Background(), parts[1])

		err = c.ChannelService.UpdateOwnerChannel(context.Background(), getSession.ChannelID, getSession.OwnerID, getSession.NewOwnerID)
		if err != nil {
			logger.Error("BOT", "Erro ao transferir posse do canal: %v", err)
			return nil
		}

		userId := fmt.Sprintf("%d", getSession.NewOwnerID)
		channelId := fmt.Sprintf("%d", channel.ID)

		data := map[string]string{
			"channelId":   fmt.Sprintf("%d", getSession.ChannelID),
			"channelName": channel.Title,
			"miniAppUrl":  auth.GenerateMiniAppUrl(userId, channelId),
		}

		textNew, kbNew := parser.GetMessageTelego("success-new-paccess-message", data)
		params := &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: from.ID},
			Text:      textNew,
			ParseMode: telego.ModeHTML,
		}
		if kbNew != nil {
			params.ReplyMarkup = kbNew
		}
		_, _ = bot.SendMessage(context.Background(), params)

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
		})

		return nil
	}
}

func buildErrorArticleTelego(userID int64, idSuffix, title, message string) *telego.InlineQueryResultArticle {
	return &telego.InlineQueryResultArticle{
		Type:  "article",
		ID:    fmt.Sprintf("error_%s_%d", idSuffix, userID),
		Title: title,
		InputMessageContent: &telego.InputTextMessageContent{
			MessageText: message,
			ParseMode:   telego.ModeHTML,
		},
	}
}
