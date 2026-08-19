package models

import "time"

type AutoDeletePost struct {
	ID        uint       `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID int64      `gorm:"index:idx_autodel_due" json:"channelId"`
	MessageID int        `json:"messageId"`
	DeleteAt  time.Time  `gorm:"index:idx_autodel_due" json:"deleteAt"`
	Status    string     `gorm:"index:idx_autodel_due;default:'pending'" json:"status"` // "pending" | "deleted" | "failed"
	LastError string     `json:"lastError,omitempty"`
	DeletedAt *time.Time `json:"deletedAt,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}
