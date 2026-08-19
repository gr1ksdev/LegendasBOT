package models

import (
	"time"
)

// ConnectedAccount representa uma conta Telegram conectada por um usuario
// do bot, utilizando o protocolo MTProto.
type ConnectedAccount struct {
	ID               string     `gorm:"type:text;primaryKey" json:"id"`
	UserID           int64      `gorm:"uniqueIndex;not null" json:"userId"`
	TelegramUserID   int64      `gorm:"not null" json:"telegramUserId"`
	Username         string     `json:"username"`
	FirstName        string     `json:"firstName"`
	EncryptedSession string     `gorm:"type:text;not null" json:"-"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
	LastUsedAt       *time.Time `json:"lastUsedAt"`
	Enabled          bool       `gorm:"default:true" json:"enabled"`
}

// TableName returns the table name for ConnectedAccount.
func (ConnectedAccount) TableName() string {
	return "connected_accounts"
}

// ConnectedAccountChannel representa a autorizacao de uma conta conectada
// para atuar em um canal especifico.
type ConnectedAccountChannel struct {
	ID                 string            `gorm:"type:text;primaryKey" json:"id"`
	ConnectedAccountID string            `gorm:"uniqueIndex:idx_acc_channel;not null" json:"connectedAccountId"`
	ChannelID          int64             `gorm:"uniqueIndex:idx_acc_channel;not null" json:"channelId"`
	AccessHash         int64             `gorm:"not null;default:0" json:"accessHash"`
	Enabled            bool              `gorm:"default:true" json:"enabled"`
	ConnectedAccount   *ConnectedAccount `gorm:"foreignKey:ConnectedAccountID;constraint:OnDelete:CASCADE;" json:"-"`
}

// TableName returns the table name for ConnectedAccountChannel.
func (ConnectedAccountChannel) TableName() string {
	return "connected_account_channels"
}
