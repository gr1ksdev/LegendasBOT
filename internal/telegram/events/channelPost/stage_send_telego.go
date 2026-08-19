package channelpost

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

func StageSendTelego(c *container.AppContainer) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		logger.Bot("📤 Iniciando envio final Telego (Tipo: %s, Album: %v)", pCtx.MessageType, pCtx.IsMediaGroup)

		err := processWithRetryTelego(pCtx.Ctx, func() error {
			logger.Bot("📤 StageSendTelego: isMediaGroup=%v, type=%s, hasBot=%v, hasExecFactory=%v",
				pCtx.IsMediaGroup, pCtx.MessageType, pCtx.Bot != nil, pCtx.ExecutorFactory != nil)

			if pCtx.IsMediaGroup {
				logger.Bot("📤 StageSendTelego: routing to ProcessMediaGroupDispatchTelego")
				return ProcessMediaGroupDispatchTelego(pCtx)
			}

			switch pCtx.MessageType {
			case MessageTypeText:
				logger.Bot("📤 StageSendTelego: routing to ProcessTextDispatchTelego")
				return ProcessTextDispatchTelego(pCtx)
			case MessageTypeAudio, MessageTypePhoto, MessageTypeVideo, MessageTypeAnimation, MessageTypeDocument:
				logger.Bot("📤 StageSendTelego: routing to ProcessMediaDispatchTelego (type=%s)", pCtx.MessageType)
				return ProcessMediaDispatchTelego(pCtx)
			case MessageTypeSticker:
				logger.Bot("📤 StageSendTelego: routing to ProcessStickerDispatchTelego")
				return ProcessStickerDispatchTelego(pCtx)
			}
			return nil
		})

		if err != nil {
			logger.Error("BOT", "❌ Falha final no envio Telego: %v", err)
			recordChannelPostEvent(c, pCtx, "post_failed", services.ChannelEventStatusError, map[string]any{"album": pCtx.IsMediaGroup}, err)
			return err
		}

		recordChannelPostEvent(c, pCtx, "post_processed", services.ChannelEventStatusSuccess, map[string]any{"album": pCtx.IsMediaGroup, "buttons": len(pCtx.FinalButtons), "has_caption": pCtx.FormattedText != ""}, nil)
		logger.Bot("✅ Postagem Telego concluída com sucesso no canal %d", pCtx.Channel.ID)
		return nil
	}
}

func processWithRetryTelego(ctx context.Context, fn func() error) error {
	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		err := fn()
		if err == nil {
			return nil
		}

		if strings.Contains(err.Error(), "Too Many Requests") || strings.Contains(err.Error(), "429") {
			retryAfter := extractRetryAfter(err.Error())
			if retryAfter == 0 {
				retryAfter = (attempt + 1) * 2
			}
			logger.Bot("⏳ Rate limit Telego, aguardando %d segundos...", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
			continue
		}
		return err
	}
	return fmt.Errorf("failed after %d attempts", maxRetries)
}
