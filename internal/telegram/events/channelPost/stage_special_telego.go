package channelpost

import (
	"context"
	"github.com/leirbagxis/FreddyBot/internal/container"
)

func StageSpecialFlowsTelego(c *container.AppContainer) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		post := pCtx.Update.ChannelPost
		if post == nil {
			return nil
		}

		handled, err := TryHandleNewPackTelego(context.Background(), pCtx.Bot, *pCtx.Channel, *post)
		if err != nil {
			return err
		}

		if handled {
			pCtx.StopPipeline = true
		}

		return nil
	}
}
