package vote

import (
	"context"
	"fmt"
	"strings"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
	"github.com/mymmrac/telego/telegohandler"
)

func HandlerTelego(c *container.AppContainer) telegohandler.Handler {
	return func(ctx *telegohandler.Context, update telego.Update) error {
		if update.CallbackQuery == nil {
			return nil
		}

		bot := ctx.Bot()
		data := update.CallbackQuery.Data
		if !strings.HasPrefix(data, "vote:") {
			return nil
		}

		// Extrair o emoji do callback data
		votedEmoji := strings.TrimPrefix(data, "vote:")

		var chatID int64
		var messageID int
		var inlineMessageID string
		var replyMarkup *telego.InlineKeyboardMarkup

		if update.CallbackQuery.Message != nil {
			msg := update.CallbackQuery.Message
			chatID = msg.GetChat().ID
			messageID = msg.GetMessageID()
			if m, ok := msg.(*telego.Message); ok {
				replyMarkup = m.ReplyMarkup
			}
		} else {
			inlineMessageID = update.CallbackQuery.InlineMessageID
		}

		if chatID == 0 && messageID == 0 && inlineMessageID == "" {
			logger.Warn("VOTE", "Recebido callback de voto sem identificador de mensagem: %v", update.CallbackQuery.ID)
			return nil
		}

		// 1. Registrar/Alternar voto no banco de dados
		added, _, err := c.VoteService.ToggleVote(context.Background(), chatID, messageID, inlineMessageID, update.CallbackQuery.From.ID, votedEmoji)
		if err != nil {
			logger.Error("VOTE", "Erro ao processar voto no banco: %v", err)
			_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
				CallbackQueryID: update.CallbackQuery.ID,
				Text:            "Erro ao computar voto.",
			})
			return nil
		}

		// Feedback visual para o usuário
		feedback := fmt.Sprintf("Voto removido de %s", votedEmoji)
		if added {
			feedback = fmt.Sprintf("Você votou em %s!", votedEmoji)
		}
		_ = bot.AnswerCallbackQuery(context.Background(), &telego.AnswerCallbackQueryParams{
			CallbackQueryID: update.CallbackQuery.ID,
			Text:            feedback,
		})

		// 2. Buscar contagens atualizadas para todos os emojis desta mensagem
		counts, err := c.VoteService.GetVoteCounts(context.Background(), chatID, messageID, inlineMessageID)
		if err != nil {
			logger.Error("VOTE", "Erro ao buscar contagens: %v", err)
			return nil
		}

		ikb := replyMarkup
		updated := false

		// Reconstrução de teclado para mensagens inline (se necessário)
		if ikb == nil && inlineMessageID != "" {
			var sessionID string
			key := fmt.Sprintf("pb_inline_map:%s", inlineMessageID)
			err := c.CacheService.Get(context.Background(), key, &sessionID)
			if err == nil && sessionID != "" {
				state, _ := c.CacheService.GetPostBuilderSession(context.Background(), sessionID)
				if state != nil {
					ikb = &telego.InlineKeyboardMarkup{}
					// Botões de URL
					for _, btn := range state.Buttons {
						ikb.InlineKeyboard = append(ikb.InlineKeyboard, []telego.InlineKeyboardButton{
							{Text: btn.Text, URL: btn.URL},
						})
					}
					// Reações
					if state.Reactions != "" {
						reactions := strings.Split(state.Reactions, ",")
						var reactionRow []telego.InlineKeyboardButton
						for _, r := range reactions {
							emoji := strings.TrimSpace(r)
							if emoji == "" {
								continue
							}

							count := counts[emoji]
							text := emoji
							if count > 0 {
								text = fmt.Sprintf("%s %d", emoji, count)
							}

							reactionRow = append(reactionRow, telego.InlineKeyboardButton{
								Text:         text,
								CallbackData: "vote:" + emoji,
							})
						}
						if len(reactionRow) > 0 {
							ikb.InlineKeyboard = append(ikb.InlineKeyboard, reactionRow)
						}
					}
					updated = true
				}
			}
		}

		if ikb == nil {
			return nil
		}

		// Atualizar o teclado (seja o original ou o reconstruído)
		for i, row := range ikb.InlineKeyboard {
			for j, btn := range row {
				if strings.HasPrefix(btn.CallbackData, "vote:") {
					emoji := strings.TrimPrefix(btn.CallbackData, "vote:")
					count := counts[emoji]

					var newText string
					if count > 0 {
						newText = fmt.Sprintf("%s %d", emoji, count)
					} else {
						newText = emoji
					}

					if ikb.InlineKeyboard[i][j].Text != newText {
						ikb.InlineKeyboard[i][j].Text = newText
						updated = true
					}
				}
			}
		}

		if updated {
			editParams := &telego.EditMessageReplyMarkupParams{
				ReplyMarkup: ikb,
			}
			if inlineMessageID != "" {
				editParams.InlineMessageID = inlineMessageID
			} else {
				editParams.ChatID = telego.ChatID{ID: chatID}
				editParams.MessageID = messageID
			}

			_, err := bot.EditMessageReplyMarkup(context.Background(), editParams)
			if err != nil {
				errStr := err.Error()
				if !strings.Contains(errStr, "message is not modified") {
					logger.Error("VOTE", "Erro ao editar teclado: %v", err)
				}
			}
		}

		return nil
	}
}
