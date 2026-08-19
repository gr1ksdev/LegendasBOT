package channelpost

import (
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

func StageDecorateTelego(c *container.AppContainer) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		// 1. Determine which buttons/reactions to use
		hashtag := extractHashtag(pCtx.OriginalCaption)
		custom := findCustomCaption(pCtx.Channel, hashtag)

		// 2. Build Keyboard
		pCtx.FinalKeyboard = CreateInlineKeyboardTelego(pCtx.FinalButtons, custom, pCtx.Channel, pCtx.MessageType)

		if pCtx.FinalKeyboard != nil {
			logger.Bot("🎹 Teclado construído com %d linhas", len(pCtx.FinalKeyboard.InlineKeyboard))
			recordChannelPostEvent(c, pCtx, "buttons_applied", services.ChannelEventStatusInfo, map[string]any{"rows": len(pCtx.FinalKeyboard.InlineKeyboard), "buttons": len(pCtx.FinalButtons)}, nil)
		} else {
			logger.Bot("⏭️ Nenhum teclado necessário")
		}

		return nil
	}
}
