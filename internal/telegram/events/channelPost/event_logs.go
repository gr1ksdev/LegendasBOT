package channelpost

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
)

func recordChannelPostEvent(c *container.AppContainer, pCtx *ProcessingContextTelego, eventType, status string, metadata map[string]any, err error) {
	if c == nil || c.ChannelEventService == nil || pCtx == nil {
		return
	}
	post := pCtx.Update.ChannelPost
	var channelID int64
	var channelTitle string
	var messageID int
	var ownerID int64
	if pCtx.Channel != nil {
		channelID = pCtx.Channel.ID
		channelTitle = pCtx.Channel.Title
		ownerID = pCtx.Channel.OwnerID
	}
	if post != nil {
		if channelID == 0 {
			channelID = post.Chat.ID
		}
		if channelTitle == "" {
			channelTitle = post.Chat.Title
		}
		messageID = post.MessageID
	}

	c.ChannelEventService.Record(context.Background(), services.ChannelEventRecordInput{
		ChannelID:         channelID,
		ChannelTitle:      channelTitle,
		OwnerID:           ownerID,
		Source:            services.ChannelEventSourceChannelPost,
		EventType:         eventType,
		Status:            status,
		MessageType:       string(pCtx.MessageType),
		TelegramMessageID: messageID,
		Error:             err,
		Metadata:          metadata,
	})
}
