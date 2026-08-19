package mychannel

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

// MessageEntityDTO representa uma entidade do Telegram serializavel via JSON.
type MessageEntityDTO struct {
	Type          string `json:"type"`
	Offset        int    `json:"offset"`
	Length        int    `json:"length"`
	URL           string `json:"url,omitempty"`
	Language      string `json:"language,omitempty"`
	CustomEmojiID string `json:"customEmojiId,omitempty"`
	UserID        int64  `json:"userId,omitempty"`
}

// entityToDTO converte telego.MessageEntity para DTO serializavel.
func entityToDTO(e telego.MessageEntity) MessageEntityDTO {
	dto := MessageEntityDTO{
		Type:          e.Type,
		Offset:        e.Offset,
		Length:        e.Length,
		URL:           e.URL,
		Language:      e.Language,
		CustomEmojiID: e.CustomEmojiID,
	}
	if e.User != nil {
		dto.UserID = e.User.ID
	}
	return dto
}

// --- "Configurar Legenda" Callback ---

func AskCaptionHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil || update.CallbackQuery.Message == nil {
			return nil
		}

		bot := ctx.Bot()
		userID := update.CallbackQuery.From.ID

		// Verificar se o usuario tem acesso premium (conta conectada ou assinatura)
		if !c.HasPremiumAccess(context.Background(), userID) {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "❌ Voce precisa conectar uma conta Telegram primeiro!",
				ShowAlert:       true,
			})
			return nil
		}

		// Extrair channelId do callback data "setcaption:CHANNEL_ID"
		callbackData := update.CallbackQuery.Data
		parts := strings.Split(callbackData, ":")
		if len(parts) != 2 {
			return nil
		}
		channelIDStr := parts[1]
		channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
		if err != nil {
			logger.Warn("BOT", "channelID invalido no callback setcaption: %s", channelIDStr)
			return nil
		}

		// Verificar se o canal pertence ao usuario
		channel, err := c.ChannelService.GetChannelByTwoID(context.Background(), userID, channelID)
		if err != nil {
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "⌛ Canal nao encontrado ou nao pertence a voce!",
				ShowAlert:       true,
			})
			return nil
		}

		// Salvar estado awaiting no cache
		if err := c.CacheService.SetAwaitingCaption(context.Background(), userID, channel.ID); err != nil {
			logger.Error("BOT", "Erro ao setar awaiting caption: %v", err)
		}
		_ = c.CacheService.SetSelectedChannel(context.Background(), userID, channel.ID)

		channelName := channel.Title
		if channelName == "" {
			channelName = fmt.Sprintf("Canal %d", channel.ID)
		}

		data := map[string]string{
			"channelName": channelName,
			"channelId":   fmt.Sprintf("%d", channel.ID),
		}
		text, kb := parser.GetMessageTelego("ask-caption-message", data)

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

// --- Receber mensagem com legenda ---

func SetCaptionHandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.Message == nil || update.Message.From == nil {
			return nil
		}

		bot := ctx.Bot()
		userID := update.Message.From.ID

		// Recuperar channelId do cache
		channelID, err := c.CacheService.GetAwaitingCaption(context.Background(), userID)
		if err != nil || channelID == 0 {
			return nil
		}

		// Extrair texto e entities da mensagem
		var text string
		var entitiesDTO []MessageEntityDTO

		if update.Message.Text != "" {
			text = update.Message.Text
			for _, e := range update.Message.Entities {
				logger.Bot("🔍 Entity detectada: type=%s offset=%d length=%d customEmojiID=%q textLen=%d",
					e.Type, e.Offset, e.Length, e.CustomEmojiID, len(update.Message.Text))
				entitiesDTO = append(entitiesDTO, entityToDTO(e))
			}
		} else if update.Message.Caption != "" {
			text = update.Message.Caption
			for _, e := range update.Message.CaptionEntities {
				logger.Bot("🔍 Entity detectada (caption): type=%s offset=%d length=%d customEmojiID=%q",
					e.Type, e.Offset, e.Length, e.CustomEmojiID)
				entitiesDTO = append(entitiesDTO, entityToDTO(e))
			}
		} else {
			// Mensagem sem texto viavel
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ Envie uma mensagem de <b>texto</b> com a legenda desejada. Stickers, imagens e outros tipos de midia nao sao suportados.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		// Serializar entities como JSON
		var entitiesJSON string
		if len(entitiesDTO) > 0 {
			jsonBytes, err := json.Marshal(entitiesDTO)
			if err != nil {
				logger.Error("BOT", "Erro ao serializar entities: %v", err)
				_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
					ChatID:    update.Message.Chat.ChatID(),
					Text:      "❌ Erro ao processar a formatacao da mensagem. Tente novamente.",
					ParseMode: telego.ModeHTML,
					ReplyParameters: &telego.ReplyParameters{
						MessageID: update.Message.MessageID,
					},
				})
				return nil
			}
			entitiesJSON = string(jsonBytes)
		}

		// Salvar no banco via CaptionService
		if err := c.CaptionService.SaveEntitiesCaption(context.Background(), channelID, text, entitiesJSON); err != nil {
			logger.Error("BOT", "Erro ao salvar entities caption: %v", err)
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:    update.Message.Chat.ChatID(),
				Text:      "❌ Erro ao salvar a legenda. Tente novamente.",
				ParseMode: telego.ModeHTML,
				ReplyParameters: &telego.ReplyParameters{
					MessageID: update.Message.MessageID,
				},
			})
			return nil
		}

		// Limpar cache
		_ = c.CacheService.DeleteAwaitingCaption(context.Background(), userID)

		// Confirmar ao usuario
		channel, _ := c.ChannelService.GetChannelByTwoID(context.Background(), userID, channelID)
		channelName := "Canal"
		if channel != nil && channel.Title != "" {
			channelName = channel.Title
		}

		data := map[string]string{
			"channelName": channelName,
			"channelId":   fmt.Sprintf("%d", channelID),
		}
		successText, kb := parser.GetMessageTelego("success-save-caption", data)

		_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
			ChatID:    update.Message.Chat.ChatID(),
			Text:      successText,
			ParseMode: telego.ModeHTML,
			ReplyParameters: &telego.ReplyParameters{
				MessageID: update.Message.MessageID,
			},
		})
		if kb != nil {
			_, _ = bot.SendMessage(context.Background(), &telego.SendMessageParams{
				ChatID:      telego.ChatID{ID: userID},
				Text:        "⚙️ O que deseja fazer agora?",
				ParseMode:   telego.ModeHTML,
				ReplyMarkup: kb,
			})
		}

		logger.Bot("📝 Legenda com entities configurada: user=%d, channel=%d, entities=%d", userID, channelID, len(entitiesDTO))
		return nil
	}
}
