package channelpost

import (
	"context"
	"math/rand"
	"strings"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

var AllowedTelegramReactionEmojis = []string{
	"👍", "👎", "❤️", "🔥", "🥰", "👏", "😁", "🤔", "🤯", "😱",
	"🤬", "😢", "🎉", "🤩", "🤮", "💩", "🙏", "👌", "🕊", "🤡",
	"🥱", "🥴", "😍", "🐳", "❤️‍🔥", "🌚", "🌭", "💯", "🤣", "⚡",
	"🍌", "🏆", "💔", "🤨", "😐", "🍓", "🍾", "💋", "😈", "😴",
	"😭", "🤓", "👻", "👨‍💻", "👀", "🎃", "🙈", "😇", "😨", "🤝",
	"✍️", "🤗", "🫡", "🎅", "🎄", "☃️", "💅", "🤪", "🗿", "🆒",
	"💘", "🙉", "🦄", "😘", "💊", "🙊", "😎", "👾", "😡",
}

func StageNativeReactionsTelego(c *container.AppContainer) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		// 1. Verificar se native reactions está habilitado no canal
		if pCtx.Channel == nil || !pCtx.Channel.NativeReactionsEnabled {
			return nil
		}

		// 2. Determinar qual mensagem reagir
		chatID := pCtx.Update.ChannelPost.Chat.ID
		messageID := pCtx.Update.ChannelPost.MessageID

		// Para media groups: já tratado pelo pipeline, a mensagem principal está em ChannelPost
		if messageID == 0 {
			return nil
		}

		// 3. Selecionar emoji baseado no modo
		var selectedEmoji string
		mode := pCtx.Channel.NativeReactionMode

		if mode == "fixed" && pCtx.Channel.NativeReactions != "" {
			// Modo fixo: usar o primeiro emoji da lista
			parts := strings.Split(pCtx.Channel.NativeReactions, ",")
			for _, p := range parts {
				p = strings.TrimSpace(p)
				if p != "" {
					selectedEmoji = p
					break
				}
			}
		}

		if mode == "random" || selectedEmoji == "" {
			// Modo aleatório: sortear da lista global
			idx := rand.Intn(len(AllowedTelegramReactionEmojis))
			selectedEmoji = AllowedTelegramReactionEmojis[idx]
		}

		if selectedEmoji == "" {
			return nil
		}

		// 4. Delay aleatório de 500ms-2s para parecer natural
		delay := time.Duration(500+rand.Intn(1500)) * time.Millisecond
		time.Sleep(delay)

		// 5. Chamar SetMessageReaction
		reaction := &telego.ReactionTypeEmoji{
			Type:  telego.ReactionEmoji,
			Emoji: selectedEmoji,
		}

		err := pCtx.Bot.SetMessageReaction(context.Background(), &telego.SetMessageReactionParams{
			ChatID:    telego.ChatID{ID: chatID},
			MessageID: messageID,
			Reaction:  []telego.ReactionType{reaction},
			IsBig:     false,
		})

		if err != nil {
			logger.Warn("NATIVE_REACTIONS", "Erro ao reagir à mensagem %d: %v", messageID, err)
			return nil // tolerante a falha — não bloquear pipeline
		}

		logger.Bot("🎯 Reação nativa aplicada: %s na mensagem %d (modo: %s)", selectedEmoji, messageID, mode)
		return nil
	}
}
