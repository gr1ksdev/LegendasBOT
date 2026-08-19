package channelpost

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"unicode/utf16"

	dbmodels "github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/telegram/executor"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

func ProcessTextDispatchTelego(pCtx *ProcessingContextTelego) error {
	post := pCtx.Update.ChannelPost

	// ── Sem permissao de edicao: so botoes via Bot API ──
	if !pCtx.Permissions.CanEdit {
		if pCtx.FinalKeyboard != nil {
			_, err := pCtx.Bot.EditMessageReplyMarkup(context.Background(), &telego.EditMessageReplyMarkupParams{
				ChatID:      telego.ChatID{ID: post.Chat.ID},
				MessageID:   post.MessageID,
				ReplyMarkup: pCtx.FinalKeyboard,
			})
			return err
		}
		return nil
	}

	// ── PASSO 1: Bot API edita texto com HTML + keyboard ──
	// Fluxo original: Bot API sempre funciona para texto formatado e botoes.
	botAPI := executor.NewBotAPIExecutor(pCtx.Bot)
	botOpts := &executor.EditOptions{
		DisableLinkPreview: pCtx.DisableLinkPreview,
	}
	keyboard := toExecutorKeyboard(pCtx.FinalKeyboard)
	err := botAPI.EditMessage(context.Background(), post.Chat.ID, post.MessageID,
		pCtx.FormattedText, telego.ModeHTML, keyboard, botOpts)
	if err != nil {
		return err
	}

	// ── PASSO 2: Se tem entities (emoji customizado), aplicar via MTProto ──
	// MTProto edita SOMENTE as entities na mensagem ja existente (sem alterar texto nem botoes).
	if pCtx.FinalEntities != "" {
		exec := getExecutorForChannel(pCtx)
		if exec != nil {
			mtprotoOpts := &executor.EditOptions{
				Entities: pCtx.FinalEntities,
			}
			// Texto cru (sem HTML, sem alteracao) + sem keyboard (ja foi aplicado no passo 1)
			editErr := exec.EditMessage(context.Background(), post.Chat.ID, post.MessageID,
				pCtx.FormattedText, "", nil, mtprotoOpts)
			if editErr != nil {
				logger.Warn("MTPROTO", "⚠️ Entities nao aplicadas via MTProto (texto ja editado pelo Bot API): %v", editErr)
			} else {
				logger.Bot("✅ Entities aplicadas via MTProto no texto %d", post.MessageID)
			}
		} else {
			logger.Bot("⚠️ Entities configuradas mas sem executor (emoji nao sera exibido)")
		}
	}

	HandleSeparatorAfterDispatchTelego(pCtx)
	return nil
}

// getExecutorForChannel retorna o executor apropriado para o canal.
func getExecutorForChannel(pCtx *ProcessingContextTelego) executor.TelegramExecutor {
	if pCtx.ExecutorFactory == nil || pCtx.Channel == nil {
		return nil
	}
	return pCtx.ExecutorFactory.ForUser(context.Background(), pCtx.Channel.OwnerID)
}

// toExecutorKeyboard converte *telego.InlineKeyboardMarkup para *executor.InlineKeyboardMarkup.
func toExecutorKeyboard(tk *telego.InlineKeyboardMarkup) *executor.InlineKeyboardMarkup {
	if tk == nil {
		return nil
	}

	rows := make([][]executor.InlineKeyboardButton, len(tk.InlineKeyboard))
	for i, row := range tk.InlineKeyboard {
		buttons := make([]executor.InlineKeyboardButton, len(row))
		for j, btn := range row {
			b := executor.InlineKeyboardButton{
				Text:  btn.Text,
				Style: btn.Style,
			}
			if btn.URL != "" {
				b.URL = btn.URL
			}
			if btn.CallbackData != "" {
				b.CallbackData = btn.CallbackData
			}
			buttons[j] = b
		}
		rows[i] = buttons
	}
	return &executor.InlineKeyboardMarkup{InlineKeyboard: rows}
}

func ProcessMediaDispatchTelego(pCtx *ProcessingContextTelego) error {
	post := pCtx.Update.ChannelPost

	// ── Sem permissao de edicao: so botoes via Bot API ──
	if !pCtx.Permissions.CanEdit {
		if pCtx.FinalKeyboard != nil {
			_, err := pCtx.Bot.EditMessageReplyMarkup(context.Background(), &telego.EditMessageReplyMarkupParams{
				ChatID:      telego.ChatID{ID: post.Chat.ID},
				MessageID:   post.MessageID,
				ReplyMarkup: pCtx.FinalKeyboard,
			})
			return err
		}
		return nil
	}

	// ── PASSO 1: Bot API edita caption com HTML + keyboard ──
	botAPI := executor.NewBotAPIExecutor(pCtx.Bot)
	keyboard := toExecutorKeyboard(pCtx.FinalKeyboard)
	err := botAPI.EditCaption(context.Background(), post.Chat.ID, post.MessageID,
		pCtx.FormattedText, telego.ModeHTML, keyboard, &executor.EditOptions{})
	if err != nil {
		return err
	}

	// ── PASSO 2: Se tem entities, aplicar via MTProto ──
	if pCtx.FinalEntities != "" {
		exec := getExecutorForChannel(pCtx)
		if exec != nil {
			mtprotoOpts := &executor.EditOptions{
				Entities: pCtx.FinalEntities,
			}
			editErr := exec.EditCaption(context.Background(), post.Chat.ID, post.MessageID,
				pCtx.FormattedText, "", nil, mtprotoOpts)
			if editErr != nil {
				logger.Warn("MTPROTO", "⚠️ Entities nao aplicadas via MTProto na midia %d: %v", post.MessageID, editErr)
			} else {
				logger.Bot("✅ Entities aplicadas via MTProto na midia %d", post.MessageID)
			}
		} else {
			logger.Bot("⚠️ Entities configuradas mas sem executor (emoji nao sera exibido)")
		}
	}

	HandleSeparatorAfterDispatchTelego(pCtx)
	return nil
}

func ProcessStickerDispatchTelego(pCtx *ProcessingContextTelego) error {
	post := pCtx.Update.ChannelPost

	if pCtx.FinalKeyboard == nil {
		return nil
	}

	// Stickers: Bot API para reply markup (nao tem caption, nao precisa de entities)
	_, err := pCtx.Bot.EditMessageReplyMarkup(context.Background(), &telego.EditMessageReplyMarkupParams{
		ChatID:      telego.ChatID{ID: post.Chat.ID},
		MessageID:   post.MessageID,
		ReplyMarkup: pCtx.FinalKeyboard,
	})
	if err == nil {
		HandleSeparatorAfterDispatchTelego(pCtx)
	}
	return err
}

func ProcessMediaGroupDispatchTelego(pCtx *ProcessingContextTelego) error {
	if len(pCtx.GroupMessages) == 0 {
		return nil
	}

	logger.Bot("📦 MediaGroup dispatch: type=%s, messages=%d", pCtx.MessageType, len(pCtx.GroupMessages))

	if pCtx.MessageType == MessageTypeAudio || pCtx.MessageType == MessageTypeDocument {
		return dispatchReSendMediaGroupTelego(pCtx)
	}

	// ── Photos/Videos: editar caption da primeira mensagem com legenda ──
	targetMessage := pCtx.GroupMessages[0]
	for _, message := range pCtx.GroupMessages {
		if message.HasCaption {
			targetMessage = message
			break
		}
	}

	// ── PASSO 1: Bot API edita caption com HTML + keyboard ──
	botAPI := executor.NewBotAPIExecutor(pCtx.Bot)
	keyboard := toExecutorKeyboard(pCtx.FinalKeyboard)
	err := botAPI.EditCaption(context.Background(), pCtx.Channel.ID, targetMessage.MessageID,
		pCtx.FormattedText, telego.ModeHTML, keyboard, &executor.EditOptions{})
	if err != nil {
		return err
	}

	// ── PASSO 2: Se tem entities, aplicar via MTProto ──
	if pCtx.FinalEntities != "" {
		exec := getExecutorForChannel(pCtx)
		if exec != nil {
			mtprotoOpts := &executor.EditOptions{
				Entities: pCtx.FinalEntities,
			}
			editErr := exec.EditCaption(context.Background(), pCtx.Channel.ID, targetMessage.MessageID,
				pCtx.FormattedText, "", nil, mtprotoOpts)
			if editErr != nil {
				logger.Warn("MTPROTO", "⚠️ Entities nao aplicadas via MTProto no album %d: %v", targetMessage.MessageID, editErr)
			} else {
				logger.Bot("✅ Entities aplicadas via MTProto no album %s", pCtx.MediaGroupID)
			}
		} else {
			logger.Bot("⚠️ Entities configuradas mas sem executor (emoji nao sera exibido)")
		}
	}

	logger.Bot("✅ Media Group %s (Photos/Videos) processed", pCtx.MediaGroupID)
	HandleSeparatorAfterDispatchTelego(pCtx)
	return nil
}

func dispatchReSendMediaGroupTelego(pCtx *ProcessingContextTelego) error {
	// Obter executor para aplicar entities via MTProto apos re-envio
	haveEntities := pCtx.FinalEntities != ""
	var exec executor.TelegramExecutor
	if haveEntities {
		exec = getExecutorForChannel(pCtx)
		if exec == nil {
			logger.Bot("⚠️ MediaGroup: entities configuradas mas sem executor (fallback Bot API)")
		}
	}

	for i, m := range pCtx.GroupMessages {
		if i > 0 {
			time.Sleep(time.Duration(200+i*150) * time.Millisecond)
		}

		var newMsg *telego.Message
		var err error

		if pCtx.MessageType == MessageTypeAudio {
			params := &telego.SendAudioParams{
				ChatID:    telego.ChatID{ID: pCtx.Channel.ID},
				Audio:     telego.InputFile{FileID: m.FileID},
				Caption:   pCtx.FormattedText,
				ParseMode: telego.ModeHTML,
			}
			if pCtx.FinalKeyboard != nil {
				params.ReplyMarkup = pCtx.FinalKeyboard
			}
			newMsg, err = pCtx.Bot.SendAudio(context.Background(), params)
		} else {
			params := &telego.SendDocumentParams{
				ChatID:    telego.ChatID{ID: pCtx.Channel.ID},
				Document:  telego.InputFile{FileID: m.FileID},
				Caption:   pCtx.FormattedText,
				ParseMode: telego.ModeHTML,
			}
			if pCtx.FinalKeyboard != nil {
				params.ReplyMarkup = pCtx.FinalKeyboard
			}
			newMsg, err = pCtx.Bot.SendDocument(context.Background(), params)
		}

		if err != nil {
			logger.Error("BOT", "❌ Failed to re-send media %d in group %s: %v", m.MessageID, pCtx.MediaGroupID, err)
			continue
		}

		// Se temos entities, aplicar via MTProto na mensagem recem-enviada
		if haveEntities && exec != nil && newMsg != nil {
			opts := &executor.EditOptions{
				Entities: pCtx.FinalEntities,
			}
			// Texto cru (sem HTML) — entities serao aplicadas separadamente
			editErr := exec.EditMessage(context.Background(), pCtx.Channel.ID, newMsg.MessageID,
				pCtx.FormattedText, "", nil, opts)
			if editErr != nil {
				logger.Error("MTPROTO", "❌ Falha ao editar legenda com entities no audio %d: %v", newMsg.MessageID, editErr)
			} else {
				logger.Bot("✅ Entities aplicadas via MTProto no audio %d", newMsg.MessageID)
			}
		}

		time.Sleep(200 * time.Millisecond)
		_ = pCtx.Bot.DeleteMessage(context.Background(), &telego.DeleteMessageParams{
			ChatID:    telego.ChatID{ID: pCtx.Channel.ID},
			MessageID: m.MessageID,
		})
	}

	logger.Bot("✅ Media Group %s (Re-sent) processed", pCtx.MediaGroupID)
	HandleSeparatorAfterDispatchTelego(pCtx)
	return nil
}

func HandleSeparatorAfterDispatchTelego(pCtx *ProcessingContextTelego) {
	if pCtx.Channel == nil || pCtx.Channel.Separator == nil {
		hasChannel := pCtx.Channel != nil
		hasSep := hasChannel && pCtx.Channel.Separator != nil
		logger.Bot("🔹 Separator: nao configurado (channel=%v, separator=%v)", hasChannel, hasSep)
		return
	}

	sep := pCtx.Channel.Separator
	logger.Bot("🔹 Separator: type=%s, agendando envio em 1s", sep.Type)

	// Obter executor para possivel uso com custom emoji
	var exec executor.TelegramExecutor
	if pCtx.ExecutorFactory != nil && sep.Type == "custom_emoji" {
		exec = getExecutorForChannel(pCtx)
	}

	time.AfterFunc(1*time.Second, func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error("SEPARATOR", "Panic recuperado no timer do separador: %v", r)
			}
		}()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		err := ProcessSeparatorTelego(ctx, pCtx.Bot, pCtx.Channel, nil, exec)
		if err != nil {
			logger.Warn("SEPARATOR", "erro ao enviar: %v", err)
		} else {
			logger.Bot("🔹 Separator: enviado com sucesso")
		}
	})
}

func ProcessSeparatorTelego(ctx context.Context, b *telego.Bot, channel *dbmodels.Channel, post *telego.Message, exec executor.TelegramExecutor) error {
	if channel == nil || channel.Separator == nil {
		return nil
	}

	if post != nil && post.MediaGroupID != "" && post.Audio != nil {
		return nil
	}

	var chatID int64
	if post != nil {
		chatID = post.Chat.ID
	} else {
		chatID = channel.ID
	}

	// ── Anti-loop: rate limit por canal ──
	// Se enviou um separador para este canal nos ultimos 3 segundos, ignora.
	// Isso evita que o ChannelPost do proprio separador dispare outro envio.
	if lastTime, ok := BotLastSeparatorSent.Load(chatID); ok {
		elapsed := time.Since(lastTime.(time.Time))
		if elapsed < 3*time.Second {
			logger.Bot("🔹 Separator: ignorado (rate limit, ultimo envio ha %.1fs para canal %d)", elapsed.Seconds(), chatID)
			return nil
		}
	}
	// Marca ANTES do envio para fechar a janela o mais cedo possivel
	BotLastSeparatorSent.Store(chatID, time.Now())

	sep := channel.Separator

	// ── Sticker: via Bot API ──
	if sep.Type == "" || sep.Type == "sticker" {
		if sep.SeparatorID == "" {
			return nil
		}

		sendCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		maxRetries := 2
		for attempt := 0; attempt < maxRetries; attempt++ {
			_, err := b.SendSticker(sendCtx, &telego.SendStickerParams{
				ChatID:  telego.ChatID{ID: chatID},
				Sticker: telego.InputFile{FileID: sep.SeparatorID},
			})
			if err == nil {
				return nil
			}

			lower := strings.ToLower(err.Error())
			if strings.Contains(lower, "too many requests") || strings.Contains(lower, "429") {
				retryAfter := extractRetryAfter(err.Error())
				if retryAfter <= 0 {
					retryAfter = (attempt + 1) * 2
				}
				time.Sleep(time.Duration(retryAfter) * time.Second)
				continue
			}
			return err
		}
		return fmt.Errorf("failed after %d attempts", maxRetries)
	}

	// ── Custom Emoji: via MTProto (ou Bot API como fallback) ──
	if sep.Type == "custom_emoji" {
		if sep.EmojiText == "" || sep.EmojiID == "" {
			return nil
		}

		// Usar entities salvas (EmojiEntitiesJSON) ou construir do EmojiID unico
		var entitiesJSONStr string

		if sep.EmojiEntitiesJSON != "" {
			// Ja temos o JSON completo das entities (multiplos emojis diferentes)
			entitiesJSONStr = sep.EmojiEntitiesJSON
			logger.Bot("🔹 Separador custom_emoji: usando entities salvas (%s)", entitiesJSONStr)
		} else {
			// Fallback: construir entity unica a partir do EmojiID (dados antigos)
			emojiLength := len(utf16.Encode([]rune(sep.EmojiText)))
			entitiesDTO := []executor.MessageEntityDTO{
				{
					Type:          "custom_emoji",
					Offset:        0,
					Length:        emojiLength,
					CustomEmojiID: sep.EmojiID,
				},
			}
			entitiesJSON, err := json.Marshal(entitiesDTO)
			if err != nil {
				return fmt.Errorf("marshal entity: %w", err)
			}
			entitiesJSONStr = string(entitiesJSON)
			logger.Bot("🔹 Separador custom_emoji: text=%q utf16_len=%d emoji_id=%s entities=%s (fallback single)",
				sep.EmojiText, emojiLength, sep.EmojiID, entitiesJSONStr)
		}

		opts := &executor.EditOptions{
			Entities: entitiesJSONStr,
		}

		if exec != nil {
			// Enviar via MTProto (conta conectada)
			logger.Bot("🔹 Separador: enviando via MTProto (exec=%T)", exec)
			sendErr := exec.SendMessage(context.Background(), chatID, sep.EmojiText, "", nil, opts)
			if sendErr != nil {
				logger.Warn("MTPROTO", "⚠️ Custom emoji separator via MTProto falhou: %v, tentando Bot API", sendErr)
				// Fallback: Bot API
				botAPIOpts := &executor.EditOptions{
					Entities: entitiesJSONStr,
				}
				botAPI := executor.NewBotAPIExecutor(b)
				err := botAPI.SendMessage(context.Background(), chatID, sep.EmojiText, "", nil, botAPIOpts)
				if err != nil {
					logger.Warn("SEPARATOR", "⚠️ Bot API fallback also failed: %v", err)
				}
				return err
			}
			logger.Bot("✅ Custom emoji separator enviado via MTProto")
			return nil
		}

		// Fallback: Bot API sem MTProto
		logger.Bot("🔹 Separador: enviando via Bot API (sem MTProto)")
		botAPI := executor.NewBotAPIExecutor(b)
		err := botAPI.SendMessage(context.Background(), chatID, sep.EmojiText, "", nil, opts)
		if err != nil {
			logger.Warn("SEPARATOR", "⚠️ Bot API custom emoji falhou: %v", err)
		}
		return err
	}

	return nil
}
