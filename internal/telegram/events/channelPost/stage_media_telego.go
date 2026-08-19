package channelpost

import (
	"context"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

func StageMediaGroupingTelego(c *container.AppContainer, executionPipeline *PipelineTelego) StageTelego {
	return func(pCtx *ProcessingContextTelego) error {
		post := pCtx.Update.ChannelPost
		if post.MediaGroupID == "" {
			return nil
		}

		mediaGroupID := post.MediaGroupID
		mgm := GetMediaGroupManagerTelego()

		group, loaded := mgm.GetMediaGroup(mediaGroupID)
		if !loaded {
			group = &MediaGroupTelego{
				Messages:           make([]MediaMessageTelego, 0, 10),
				Processed:          false,
				MessageEditAllowed: true,
				ChatID:             post.Chat.ID,
			}
			mgm.SetMediaGroup(mediaGroupID, group)
		}

		group.mu.Lock()
		defer group.mu.Unlock()

		if group.Processed || mgm.IsProcessed(mediaGroupID) {
			pCtx.StopPipeline = true
			return nil
		}

		fileID := getFileIDTelego(post)

		group.Messages = append(group.Messages, MediaMessageTelego{
			MessageID:       post.MessageID,
			FileID:          fileID,
			HasCaption:      post.Caption != "",
			Caption:         post.Caption,
			CaptionEntities: post.CaptionEntities,
		})

		if group.Timer != nil {
			group.Timer.Stop()
		}

		timeout := time.Duration(800+len(group.Messages)*200) * time.Millisecond
		if timeout > 2*time.Second {
			timeout = 2 * time.Second
		}

		group.Timer = time.AfterFunc(timeout, func() {
			group.mu.Lock()
			if group.Processed {
				group.mu.Unlock()
				return
			}
			group.Processed = true
			msgs := group.Messages
			group.mu.Unlock()

			mgm.MarkProcessed(mediaGroupID)
			mgm.DeleteMediaGroup(mediaGroupID)

			logger.Bot("📸 Media group ready Telego: %s (%d messages)", mediaGroupID, len(msgs))

			groupCtx := &ProcessingContextTelego{
				Ctx:             context.Background(),
				Bot:             pCtx.Bot,
				Update:          pCtx.Update,
				MessageType:     pCtx.MessageType,
				Channel:         pCtx.Channel,
				Permissions:     pCtx.Permissions,
				IsMediaGroup:    true,
				MediaGroupID:    mediaGroupID,
				GroupMessages:   msgs,
				Pipeline:        executionPipeline,
				ExecutorFactory: pCtx.ExecutorFactory,
			}

			messageQueue.AddTelegoToQueue(groupCtx, executionPipeline)
		})

		pCtx.StopPipeline = true
		return nil
	}
}

func getFileIDTelego(post *telego.Message) string {
	if post == nil {
		return ""
	}
	switch {
	case post.Audio != nil:
		return post.Audio.FileID
	case post.Document != nil:
		return post.Document.FileID
	case post.Video != nil:
		return post.Video.FileID
	case post.Animation != nil:
		return post.Animation.FileID
	case post.Photo != nil && len(post.Photo) > 0:
		return post.Photo[len(post.Photo)-1].FileID
	}
	return ""
}
