package mychannel

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/container"
	separatorModels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/telegram/executor"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// --- Sticker Separator ---

func AskStickerSeparatorHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetSelectedChannel(context.Background(), userId)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		data := map[string]string{
			"channelName": channel.Title,
			"channelId":   fmt.Sprintf("%d", session),
		}

		text, kb := parser.GetMessageTelego("ask-separator-message", data)
		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			MessageID: update.CallbackQuery.Message.GetMessageID(),
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

func RequireStickerSeparatorHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetSelectedChannel(context.Background(), userId)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		c.CacheService.SetAwaitingStickerSeparator(context.Background(), userId, session)

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", session)
		}

		vars := map[string]string{
			"channelName": channelName,
			"channelId":   fmt.Sprintf("%d", session),
		}

		text, kb := parser.GetMessageTelego("require-separator-message", vars)
		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			MessageID: update.CallbackQuery.Message.GetMessageID(),
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

func SetSeparatorHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.Message.From.ID
		channelId, _ := c.CacheService.GetAwaitingStickerSeparator(context.Background(), userId)
		if channelId == 0 {
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, channelId)
		if err != nil {
			return nil
		}

		separator := &separatorModels.Separator{
			ID:             uuid.NewString(),
			OwnerChannelID: channelId,
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}

		isCustomEmoji := false

		// ── Caso 1: Sticker ──
		if update.Message.Sticker != nil {
			separator.Type = "sticker"
			separator.SeparatorID = update.Message.Sticker.FileID

			file, err := bot.GetFile(context.Background(), &telego.GetFileParams{FileID: separator.SeparatorID})
			if err != nil {
				logger.Error("BOT", "erro ao obter sticker: %v", err)
			}

			if file != nil {
				separator.SeparatorURL = file.FilePath
			}
		}

		// ── Caso 2: Custom Emoji ──
		if update.Message.Text != "" && len(update.Message.Entities) > 0 {
			// Coletar TODAS as entidades custom_emoji (podem ser emojis diferentes)
			var emojiEntities []executor.MessageEntityDTO
			for _, entity := range update.Message.Entities {
				if entity.Type == "custom_emoji" {
					emojiEntities = append(emojiEntities, executor.MessageEntityDTO{
						Type:          entity.Type,
						Offset:        entity.Offset,
						Length:        entity.Length,
						CustomEmojiID: entity.CustomEmojiID,
					})
				}
			}
			if len(emojiEntities) > 0 {
				isCustomEmoji = true
				separator.Type = "custom_emoji"
				separator.EmojiText = update.Message.Text
				separator.EmojiID = emojiEntities[0].CustomEmojiID // primeiro ID para compatibilidade
				emojiEntitiesJSON, err := json.Marshal(emojiEntities)
				if err != nil {
					logger.Error("HANDLER", "erro ao serializar entities: %v", err)
				} else {
					separator.EmojiEntitiesJSON = string(emojiEntitiesJSON)
				}

				logger.Bot("📝 Separador custom_emoji: text=%q entities=%d json=%s",
					separator.EmojiText, len(emojiEntities), separator.EmojiEntitiesJSON)

				// Custom emoji requer conta conectada ou assinatura premium
				if !c.HasPremiumAccess(context.Background(), userId) {
					text, kb := parser.GetMessageTelego("require-connected-account-separator", map[string]string{
						"channelId": fmt.Sprintf("%d", channelId),
					})
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
					if _, err := bot.SendMessage(context.Background(), params); err != nil {
						logger.Warn("BOT", "erro ao enviar mensagem de requisito de conta: %v", err)
					}
					return nil
				}
			}
		}

		// Se nao detectou nem sticker nem custom emoji, sair
		if separator.Type == "" {
			return nil
		}

		if err = c.SeparatorService.SaveSeparator(context.Background(), separator); err != nil {
			text, kb := parser.GetMessageTelego("failed-save-separator", map[string]string{
				"channelId": fmt.Sprintf("%d", channelId),
			})

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
			_, _ = bot.SendMessage(context.Background(), params)
			return nil
		}

		// Invalidar cache do canal para forcar recarga com o Separator
		_ = c.CacheService.InvalidateChannel(context.Background(), channelId)

		c.CacheService.DeleteAwaitingStickerSeparator(context.Background(), userId)

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", channelId)
		}

		successKey := "success-save-separator"
		if isCustomEmoji {
			successKey = "success-save-custom-emoji-separator"
		}

		text, kb := parser.GetMessageTelego(successKey, map[string]string{
			"channelId":   fmt.Sprintf("%d", channelId),
			"channelName": channelName,
		})

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
		_, _ = bot.SendMessage(context.Background(), params)
		return nil
	}
}

func DeleteSeparatorHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetSelectedChannel(context.Background(), userId)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		separator, err := c.SeparatorService.GetSeparatorByOwnerChannelID(context.Background(), channel.ID)
		if separator == nil || err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Você ainda não possui nenhum separador vinculado.",
				ShowAlert:       true,
			})
			return nil
		}

		err = c.SeparatorService.DeleteSeparatorByOwnerChannelId(context.Background(), session)
		if err != nil {
			logger.Error("BOT", "Erro ao excluir separator: %v", err)
			return nil
		}

		// Invalidar cache do canal apos remover o Separator
		_ = c.CacheService.InvalidateChannel(context.Background(), session)

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", session)
		}

		data := map[string]string{
			"channelName": channelName,
			"channelId":   fmt.Sprintf("%d", session),
		}

		text, kb := parser.GetMessageTelego("success-delete-separator", data)
		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			MessageID: update.CallbackQuery.Message.GetMessageID(),
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		_, _ = bot.EditMessageText(context.Background(), params)

		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            "✅ Separador excluido com sucesso!",
		})
		return nil
	}
}

// --- Transfer Access ---

func AskTransferAccessHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetSelectedChannel(context.Background(), userId)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", session)
		}

		data := map[string]string{
			"channelName": channelName,
			"channelId":   fmt.Sprintf("%d", session),
		}
		_ = c.CacheService.SetTransferChannel(context.Background(), userId, session)

		text, kb := parser.GetMessageTelego("ask-paccess-message", data)
		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			MessageID: update.CallbackQuery.Message.GetMessageID(),
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

func TransferAcessHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userId := update.CallbackQuery.From.ID
		session, err := c.CacheService.GetTransferChannel(context.Background(), userId)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Seção Expirada. Selecione o canal novamente!",
				ShowAlert:       true,
			})
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, session)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal não encontrado ou não pertence a você!",
				ShowAlert:       true,
			})
			return nil
		}

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", session)
		}

		user, _ := c.UserService.GetUserByID(context.Background(), userId)
		data := map[string]string{
			"channelName": channelName,
			"channelId":   fmt.Sprintf("%d", session),
			"ownerId":     fmt.Sprintf("%d", user.UserId),
			"ownerName":   user.FirstName,
		}

		text, kb := parser.GetMessageTelego("require-paccess-message", data)
		params := &telego.EditMessageTextParams{
			ChatID:    update.CallbackQuery.Message.GetChat().ChatID(),
			Text:      text,
			ParseMode: telego.ModeHTML,
			MessageID: update.CallbackQuery.Message.GetMessageID(),
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

func SetTransferAccessHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		bot := ctx.Bot()
		botInfo, _ := bot.GetMe(context.Background())
		userId := update.Message.From.ID

		channelId, err := c.CacheService.GetTransferChannel(context.Background(), userId)
		if err != nil {
			return nil
		}

		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userId, channelId)
		if err != nil || channel == nil {
			return nil
		}

		newOwnerID, err := strconv.ParseInt(update.Message.Text, 10, 64)
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ ID de usuário inválido! Tente novamente com um ID válido.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		if newOwnerID == userId {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ O novo dono precisa ser diferente de voce.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		newOwner, err := bot.GetChat(context.Background(), &telego.GetChatParams{ChatID: telego.ChatID{ID: newOwnerID}})
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ O novo dono precisa iniciar o bot pelo menos uma vez.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		if newOwnerID == botInfo.ID {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ O novo dono não pode ser eu.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		admins, err := bot.GetChatAdministrators(context.Background(), &telego.GetChatAdministratorsParams{
			ChatID: telego.ChatID{ID: channelId},
		})
		if err != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ Erro ao consultar administradores do canal.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		isAdmin := false
		for _, admin := range admins {
			status := admin.MemberStatus()
			if status == telego.MemberStatusAdministrator {
				if a, ok := admin.(*telego.ChatMemberAdministrator); ok && a.User.ID == newOwnerID {
					isAdmin = true
					break
				}
			}
			if status == telego.MemberStatusCreator {
				if a, ok := admin.(*telego.ChatMemberOwner); ok && a.User.ID == newOwnerID {
					isAdmin = true
					break
				}
			}
		}

		if !isAdmin {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ O novo dono precisa ser administrador do canal.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		_ = c.SeparatorService.DeleteSeparatorByOwnerChannelId(context.Background(), userId)
		err = c.ChannelService.UpdateOwnerChannel(context.Background(), channelId, userId, newOwnerID)
		if err != nil {
			return nil
		}

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", channelId)
		}

		channelIDStr := fmt.Sprintf("%d", channelId)
		newOwnerIDStr := fmt.Sprintf("%d", newOwnerID)

		data := map[string]string{
			"channelId":    channelIDStr,
			"channelName":  channelName,
			"newOwnerName": newOwner.LastName,
			"newOwnerId":   newOwnerIDStr,
			"miniAppUrl":   auth.GenerateMiniAppUrl(newOwnerIDStr, channelIDStr),
		}

		textOld, kbOld := parser.GetMessageTelego("success-old-paccess-message", data)
		paramsOld := &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      textOld,
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		}
		if kbOld != nil {
			paramsOld.ReplyMarkup = kbOld
		}
		_, _ = bot.SendMessage(context.Background(), paramsOld)

		textNew, kbNew := parser.GetMessageTelego("success-new-paccess-message", data)
		paramsNew := &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: newOwnerID},
			Text:      textNew,
			ParseMode: telego.ModeHTML,
		}
		if kbNew != nil {
			paramsNew.ReplyMarkup = kbNew
		}
		_, _ = bot.SendMessage(context.Background(), paramsNew)

		_, _ = c.CacheService.DeleteAllUserSessionsBySuffix(context.Background(), userId)
		return nil
	}
}
