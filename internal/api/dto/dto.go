package dto

import "time"

type UserDTO struct {
	ID            int64        `json:"id"`
	FirstName     string       `json:"first_name"`
	Username      string       `json:"username"`
	IsAdmin       bool         `json:"is_admin"`
	IsBlacklisted bool         `json:"is_blacklisted"`
	IsContribute  bool         `json:"isContribute"`
	Channels      []ChannelDTO `json:"channels,omitempty"`
}

type UserLookupDTO struct {
	ID        int64  `json:"id"`
	FirstName string `json:"first_name"`
	Username  string `json:"username"`
}

type ChannelDTO struct {
	ID                     int64              `json:"id"`
	Title                  string             `json:"title"`
	NewPackCaption         string             `json:"newPackCaption"`
	NewPackMessageButtons  bool               `json:"newPackMessageButtons"`
	NewPackStickerButtons  bool               `json:"newPackStickerButtons"`
	NewPackMessagePosition string             `json:"newPackMessagePosition"`
	NewPackReplyToSticker  bool               `json:"newPackReplyToSticker"`
	InviteURL              string             `json:"inviteUrl"`
	OwnerID                int64              `json:"ownerId"`
	Reactions              string             `json:"reactions"`
	ReactionPosition       int                `json:"reactionPosition"`
	DynamicLinks           bool               `json:"dynamicLinks"`
	DLBotButtons           bool               `json:"dlBotButtons"`
	DLBotCaptions          bool               `json:"dlBotCaptions"`
	DLBotReactions         bool               `json:"dlBotReactions"`
	NativeReactionsEnabled bool               `json:"nativeReactionsEnabled"`
	NativeReactions        string             `json:"nativeReactions"`
	NativeReactionMode     string             `json:"nativeReactionMode"`
	DefaultCaption         *DefaultCaptionDTO `json:"defaultCaption,omitempty"`
	Buttons                []ButtonDTO        `json:"buttons,omitempty"`
	CustomCaptions         []CustomCaptionDTO `json:"customCaptions,omitempty"`
	CreatedAt              time.Time          `json:"created_at"`
	UpdatedAt              time.Time          `json:"updated_at"`
}

type DefaultCaptionDTO struct {
	CaptionID         string         `json:"captionId"`
	Caption           string         `json:"caption"`
	MessagePermission *PermissionDTO `json:"messagePermission,omitempty"`
	ButtonsPermission *PermissionDTO `json:"buttonsPermission,omitempty"`
	CreatedAt         time.Time      `json:"created_at"`
}

type PermissionDTO struct {
	LinkPreview bool `json:"linkPreview"`
	Message     bool `json:"message"`
	Audio       bool `json:"audio"`
	Video       bool `json:"video"`
	Photo       bool `json:"photo"`
	Document    bool `json:"document"`
	Sticker     bool `json:"sticker"`
	GIF         bool `json:"gif"`
	Reactions   bool `json:"reactions,omitempty"`
}

type ButtonDTO struct {
	ButtonID  string `json:"buttonId"`
	Name      string `json:"nameButton"`
	URL       string `json:"buttonUrl"`
	Style     string `json:"style,omitempty"`
	PositionX int    `json:"positionX"`
	PositionY int    `json:"positionY"`
}

type CustomCaptionDTO struct {
	CaptionID   string      `json:"captionId"`
	Code        string      `json:"code"`
	Caption     string      `json:"caption"`
	LinkPreview bool        `json:"linkPreview"`
	Buttons     []ButtonDTO `json:"buttons,omitempty"`
	CreatedAt   time.Time   `json:"created_at"`
}

type ScheduledPostDTO struct {
	ID            string     `json:"id"`
	OwnerID       int64      `json:"ownerId"`
	ChannelID     int64      `json:"channelId"`
	ChannelTitle  string     `json:"channelTitle"`
	ScheduleType  string     `json:"scheduleType"`
	ScheduleTime  string     `json:"scheduleTime"`
	ScheduledAt   *time.Time `json:"scheduledAt,omitempty"`
	ScheduleDays  string     `json:"scheduleDays,omitempty"`
	NextRunAt     time.Time  `json:"nextRunAt"`
	RepeatUntil   *time.Time `json:"repeatUntil,omitempty"`
	QueueGroupID  string     `json:"queueGroupId,omitempty"`
	QueuePosition int        `json:"queuePosition"`
	LoopQueue     bool       `json:"loopQueue"`
	Status        string     `json:"status"`
	SentAt        *time.Time `json:"sentAt,omitempty"`
	SentCount     int        `json:"sentCount"`
	LastError     string     `json:"lastError,omitempty"`
	CreatedAt     time.Time  `json:"createdAt"`
}
