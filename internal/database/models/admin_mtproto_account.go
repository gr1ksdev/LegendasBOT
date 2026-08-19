package models

import (
	"time"
)

// AdminMTProtoAccount representa uma conta Telegram gerenciada pelo admin
// para ser utilizada no fluxo de edicao de postagem via MTProto.
type AdminMTProtoAccount struct {
	ID               string     `gorm:"type:text;primaryKey" json:"id"`
	Label            string     `gorm:"type:text;not null" json:"label"`
	PhoneNumber      string     `gorm:"type:text;not null" json:"phoneNumber"`
	TelegramUserID   int64      `gorm:"not null" json:"telegramUserId"`
	Username         string     `json:"username"`
	FirstName        string     `json:"firstName"`
	EncryptedSession string     `gorm:"type:text;not null" json:"-"`
	Enabled          bool       `gorm:"default:true" json:"enabled"`
	Status           string     `gorm:"type:text;default:disconnected" json:"status"` // "connected", "disconnected", "error"
	LastUsedAt       *time.Time `json:"lastUsedAt"`
	CreatedAt        time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt        time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName returns the table name for AdminMTProtoAccount.
func (AdminMTProtoAccount) TableName() string {
	return "admin_mtproto_accounts"
}
