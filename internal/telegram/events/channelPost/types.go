package channelpost

import (
	"time"
)

type MessageType string

const (
	MessageTypeText      MessageType = "text"
	MessageTypeAudio     MessageType = "audio"
	MessageTypeSticker   MessageType = "sticker"
	MessageTypePhoto     MessageType = "photo"
	MessageTypeVideo     MessageType = "video"
	MessageTypeAnimation MessageType = "animation"
	MessageTypeDocument  MessageType = "document"
)

const (
	CleanupTimeout = 30 * time.Minute
	CacheTTL       = 10 * time.Minute
)

type PermissionCheckResult struct {
	CanEdit           bool
	CanAddButtons     bool
	CanEditButtons    bool
	CanAddReactions   bool
	CanUseLinkPreview bool
	Reason            string
}

type ProcessedGroup struct {
	Timestamp time.Time
}

type PermissionMap map[string]interface{}
