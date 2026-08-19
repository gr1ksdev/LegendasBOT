package postbuilder

import (
	"context"

	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
)

func recordPostBuilderEvent(c *container.AppContainer, eventType, status string, actorID int64, channelID int64, sessionID string, metadata map[string]any, err error) {
	if c == nil || c.ChannelEventService == nil {
		return
	}
	input := services.ChannelEventRecordInput{
		ChannelID: channelID,
		ActorID:   actorID,
		Source:    services.ChannelEventSourcePostBuilder,
		EventType: eventType,
		Status:    status,
		SessionID: sessionID,
		Error:     err,
		Metadata:  metadata,
	}
	if channelID != 0 {
		if channel, loadErr := c.ChannelService.GetChannelWithRelations(context.Background(), channelID); loadErr == nil && channel != nil {
			input.ChannelTitle = channel.Title
			input.OwnerID = channel.OwnerID
		}
	}
	c.ChannelEventService.Record(context.Background(), input)
}
