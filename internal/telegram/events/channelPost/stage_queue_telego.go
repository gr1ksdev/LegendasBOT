package channelpost

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
)

func StageQueueTelego(c *container.AppContainer, executionPipeline *PipelineTelego) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		workerCtx := *pCtx
		workerCtx.Ctx = context.Background()
		workerCtx.StopPipeline = false

		messageQueue.AddTelegoToQueue(&workerCtx, executionPipeline)

		pCtx.StopPipeline = true
		return nil
	}
}
