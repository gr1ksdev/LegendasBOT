package models

import "time"

type ServerConfig struct {
	ID                      uint      `gorm:"primaryKey" json:"id"`
	Maintence               bool      `gorm:"column:maintence;default:false" json:"maintence"`
	ForceJoin               bool      `gorm:"default:false" json:"forceJoin"`
	GlobalDefaultCaption    string    `json:"globalDefaultCaption"`
	GlobalNewPackCaption    string    `json:"globalNewPackCaption"`
	FixedPostBuilderEnabled bool      `gorm:"default:true" json:"fixedPostBuilderEnabled"`
	FixedPostBuilderKey     string    `gorm:"default:legendasbot" json:"fixedPostBuilderKey"`
	FixedPostBuilderPayload string    `gorm:"type:text" json:"fixedPostBuilderPayload"`
	LogRetentionDays        int       `gorm:"default:30" json:"logRetentionDays"`
	LogsEnabled             bool      `gorm:"default:true" json:"logsEnabled"`
	CreatedAt               time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type User struct {
	UserId        int64     `gorm:"primaryKey" json:"id"` // ID do Telegram
	FirstName     string    `json:"first_name"`
	Username      string    `gorm:"index" json:"username"`
	IsAdmin       bool      `gorm:"default:false" json:"is_admin"`
	IsBlacklisted bool      `gorm:"default:false" json:"is_blacklisted"`
	IsContribute  bool      `gorm:"default:false" json:"isContribute"`
	Features      string    `gorm:"type:text;default:'{}'" json:"features"` // JSON de UserFeatures
	Channels      []Channel `gorm:"foreignKey:OwnerID" json:"channels"`
	CreatedAt     time.Time `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt     time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Channel struct {
	ID                     int64           `gorm:"primaryKey" json:"id"` // ID do Telegram
	Title                  string          `json:"title"`
	NewPackCaption         string          `json:"newPackCaption"`
	NewPackMessageButtons  *bool           `gorm:"default:true" json:"newPackMessageButtons"`
	NewPackStickerButtons  *bool           `gorm:"default:true" json:"newPackStickerButtons"`
	NewPackMessagePosition *string         `gorm:"default:above" json:"newPackMessagePosition"`
	NewPackReplyToSticker  *bool           `gorm:"default:false" json:"newPackReplyToSticker"`
	InviteURL              string          `json:"inviteUrl"`
	OwnerID                int64           `gorm:"index" json:"ownerId"`
	Owner                  *User           `gorm:"foreignKey:OwnerID" json:"owner,omitempty"`
	DefaultCaption         *DefaultCaption `gorm:"foreignKey:OwnerChannelID;constraint:OnDelete:CASCADE;" json:"defaultCaption,omitempty"`
	Buttons                []Button        `gorm:"foreignKey:OwnerChannelID;constraint:OnDelete:CASCADE;" json:"buttons"`
	Separator              *Separator      `gorm:"foreignKey:OwnerChannelID;constraint:OnDelete:CASCADE;" json:"separator,omitempty"`
	CustomCaptions         []CustomCaption `gorm:"foreignKey:OwnerChannelID;constraint:OnDelete:CASCADE;" json:"customCaptions"`
	TokenVersion           int64           `gorm:"not null;default:1"`
	Reactions              string          `json:"reactions"`
	ReactionPosition       int             `gorm:"default:0" json:"reactionPosition"`
	DynamicLinks           bool            `gorm:"default:false" json:"dynamicLinks"`
	DLBotButtons           bool            `gorm:"default:true" json:"dlBotButtons"`
	DLBotCaptions          bool            `gorm:"default:true" json:"dlBotCaptions"`
	DLBotReactions         bool            `gorm:"default:true" json:"dlBotReactions"`
	NativeReactionsEnabled bool            `gorm:"default:false" json:"nativeReactionsEnabled"`
	NativeReactions        string          `json:"nativeReactions"`
	NativeReactionMode     string          `gorm:"default:random" json:"nativeReactionMode"`
	CreatedAt              time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt              time.Time       `gorm:"autoUpdateTime;index" json:"updated_at"`
}

type ChannelEvent struct {
	ID                string    `gorm:"type:text;primaryKey" json:"id"`
	ChannelID         int64     `gorm:"index;index:idx_channel_event_channel_created" json:"channelId"`
	ChannelTitle      string    `gorm:"index" json:"channelTitle"`
	OwnerID           int64     `gorm:"index;index:idx_channel_event_owner_created" json:"ownerId"`
	ActorID           int64     `gorm:"index" json:"actorId"`
	Source            string    `gorm:"index" json:"source"`
	EventType         string    `gorm:"index" json:"eventType"`
	Status            string    `gorm:"index" json:"status"`
	MessageType       string    `json:"messageType"`
	TelegramMessageID int       `json:"telegramMessageId"`
	SessionID         string    `gorm:"index" json:"sessionId"`
	ErrorMessage      string    `gorm:"type:text" json:"errorMessage"`
	Metadata          string    `gorm:"type:text" json:"metadata"`
	CreatedAt         time.Time `gorm:"autoCreateTime;index;index:idx_channel_event_channel_created,sort:desc;index:idx_channel_event_owner_created,sort:desc" json:"created_at"`
}

type DefaultCaption struct {
	CaptionID         string             `gorm:"type:text;primaryKey" json:"captionId"`
	Caption           string             `json:"caption"`
	Entities          string             `gorm:"type:text" json:"entities"`        // JSON das MessageEntities (MTProto)
	UseEntities       bool               `gorm:"default:false" json:"useEntities"` // Usar entities ao invés de DetectParseMode
	MessagePermission *MessagePermission `gorm:"foreignKey:OwnerCaptionID;constraint:OnDelete:CASCADE;" json:"messagePermission,omitempty"`
	ButtonsPermission *ButtonsPermission `gorm:"foreignKey:OwnerCaptionID;constraint:OnDelete:CASCADE;" json:"buttonsPermission,omitempty"`
	OwnerChannelID    int64              `gorm:"unique;index" json:"ownerChannelId"`
	CreatedAt         time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time          `gorm:"autoUpdateTime" json:"updated_at"`
}

type MessagePermission struct {
	MessagePermissionID string    `gorm:"type:text;primaryKey" json:"messagePermissionId"`
	LinkPreview         bool      `gorm:"default:true" json:"linkPreview"`
	Message             bool      `gorm:"default:true" json:"message"`
	Audio               bool      `gorm:"default:true" json:"audio"`
	Video               bool      `gorm:"default:true" json:"video"`
	Photo               bool      `gorm:"default:true" json:"photo"`
	Document            bool      `gorm:"default:true" json:"document"`
	Sticker             bool      `gorm:"default:true" json:"sticker"`
	GIF                 bool      `gorm:"default:true" json:"gif"`
	Reactions           bool      `gorm:"default:true" json:"reactions"`
	OwnerCaptionID      string    `gorm:"unique;index" json:"ownerCaptionId"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type ButtonsPermission struct {
	ButtonsPermissionID string    `gorm:"type:text;primaryKey" json:"buttonsPermissionId"`
	Message             bool      `gorm:"default:true" json:"message"`
	Audio               bool      `gorm:"default:true" json:"audio"`
	Video               bool      `gorm:"default:true" json:"video"`
	Photo               bool      `gorm:"default:true" json:"photo"`
	Document            bool      `gorm:"default:true" json:"document"`
	Sticker             bool      `gorm:"default:true" json:"sticker"`
	GIF                 bool      `gorm:"default:true" json:"gif"`
	OwnerCaptionID      string    `gorm:"unique;index" json:"ownerCaptionId"`
	CreatedAt           time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt           time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Button struct {
	ButtonID       string    `gorm:"type:text;primaryKey" json:"buttonId"`
	NameButton     string    `json:"nameButton"`
	ButtonURL      string    `json:"buttonUrl"`
	Style          string    `gorm:"type:text" json:"style,omitempty"`
	PositionX      int       `gorm:"default:0;index:idx_button_pos" json:"positionX"`
	PositionY      int       `gorm:"default:0;index:idx_button_pos" json:"positionY"`
	OwnerChannelID int64     `gorm:"index" json:"ownerChannelId"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type Separator struct {
	ID                string    `gorm:"type:text;primaryKey" json:"id"`
	Type              string    `gorm:"type:text;default:sticker" json:"type"` // "sticker" ou "custom_emoji"
	SeparatorID       string    `json:"separatorId"`                           // FileID do sticker (tipo "sticker")
	SeparatorURL      string    `json:"separatorUrl"`                          // URL do sticker (tipo "sticker")
	EmojiText         string    `json:"emojiText"`                             // caractere do emoji (tipo "custom_emoji")
	EmojiID           string    `json:"emojiId"`                               // ID do primeiro emoji customizado (tipo "custom_emoji")
	EmojiEntitiesJSON string    `json:"emojiEntitiesJson"`                     // JSON completo das entities custom_emoji (para multiplos emojis diferentes)
	OwnerChannelID    int64     `gorm:"unique;index" json:"ownerChannelId"`
	CreatedAt         time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt         time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

type CustomCaption struct {
	CaptionID      string                `gorm:"type:text;primaryKey" json:"captionId"`
	Code           string                `gorm:"index:idx_hashtag_lookup" json:"code"`
	Caption        string                `json:"caption"`
	LinkPreview    bool                  `json:"linkPreview"`
	Buttons        []CustomCaptionButton `gorm:"foreignKey:OwnerCaptionID;constraint:OnDelete:CASCADE;" json:"buttons"`
	OwnerChannelID int64                 `gorm:"index;index:idx_hashtag_lookup" json:"ownerChannelId"`
	CreatedAt      time.Time             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time             `gorm:"autoUpdateTime" json:"updated_at"`
}

type CustomCaptionButton struct {
	ButtonID       string    `gorm:"type:text;primaryKey" json:"buttonId"`
	NameButton     string    `json:"nameButton"`
	ButtonURL      string    `json:"buttonUrl"`
	Style          string    `gorm:"type:text" json:"style,omitempty"`
	PositionX      int       `gorm:"default:0" json:"positionX"`
	PositionY      int       `gorm:"default:0" json:"positionY"`
	OwnerCaptionID string    `gorm:"index" json:"ownerCaptionId"`
	CreatedAt      time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt      time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// ── Custom Emoji (cache global do arquivo) ──
type CustomEmoji struct {
	EmojiID   string    `gorm:"primaryKey;type:text" json:"emojiId"`
	FileData  []byte    `gorm:"type:bytea" json:"-"`                     // BLOB do arquivo (bytea no PostgreSQL)
	FileType  string    `gorm:"type:text;default:.webp" json:"fileType"` // ".webp", ".webm" ou ".tgs"
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

// ── User → Emoji Access (controle de quem pode visualizar) ──
type UserEmojiAccess struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    int64     `gorm:"index:idx_user_emoji,unique" json:"userId"`
	EmojiID   string    `gorm:"index:idx_user_emoji,unique;type:text" json:"emojiId"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
}

type Vote struct {
	ID              uint   `gorm:"primaryKey" json:"id"`
	ChatID          int64  `gorm:"index:idx_vote_user,unique;index:idx_vote_count" json:"chat_id"`
	MessageID       int    `gorm:"index:idx_vote_user,unique;index:idx_vote_count" json:"message_id"`
	InlineMessageID string `gorm:"index:idx_vote_user,unique;index:idx_vote_count" json:"inline_message_id"`
	UserID          int64  `gorm:"index:idx_vote_user,unique" json:"user_id"`
	Emoji           string `gorm:"index:idx_vote_count" json:"emoji"`
	CreatedAt       time.Time
}
