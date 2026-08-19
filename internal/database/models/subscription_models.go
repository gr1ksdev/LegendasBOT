package models

import (
	"time"
)

// ── Subscription Status ──

const (
	SubscriptionActive   = "active"
	SubscriptionCanceled = "canceled"
	SubscriptionExpired  = "expired"
)

// ── Subscription ──

// Subscription representa uma assinatura premium de um usuario.
type Subscription struct {
	ID                   string    `gorm:"type:text;primaryKey" json:"id"`
	UserID               int64     `gorm:"index;not null" json:"userId"`
	Status               string    `gorm:"type:text;default:active" json:"status"`
	CurrentPeriodStart   time.Time `json:"currentPeriodStart"`
	CurrentPeriodEnd     time.Time `json:"currentPeriodEnd"`
	ExtraChannels        int       `gorm:"default:0" json:"extraChannels"`
	CancelAtPeriodEnd    bool      `gorm:"default:false" json:"cancelAtPeriodEnd"`
	TelegramPaymentID    string    `gorm:"type:text" json:"telegramPaymentId"`               // charge_id da assinatura principal
	ExtraChannelPayments string    `gorm:"type:text;default:''" json:"extraChannelPayments"` // charge IDs dos canais extras (separados por virgula)
	CreatedAt            time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt            time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName returns the table name for Subscription.
func (Subscription) TableName() string {
	return "subscriptions"
}

// ── Refund ──

// Refund representa o reembolso de uma compra Stars.
type Refund struct {
	ID                      string    `gorm:"type:text;primaryKey" json:"id"`
	SubscriptionID          string    `gorm:"index;not null" json:"subscriptionId"`
	UserID                  int64     `gorm:"index;not null" json:"userId"`
	TelegramPaymentChargeID string    `gorm:"type:text;not null" json:"telegramPaymentChargeId"`
	AmountStars             int       `gorm:"not null" json:"amountStars"`
	Status                  string    `gorm:"type:text;default:processed" json:"status"`
	RefundedAt              time.Time `json:"refundedAt"`
	RefundedBy              int64     `json:"refundedBy"`
	CreatedAt               time.Time `gorm:"autoCreateTime" json:"createdAt"`
}

// TableName returns the table name for Refund.
func (Refund) TableName() string {
	return "refunds"
}

// ── User Features ──

// UserFeatures representa os recursos disponiveis para um usuario
// com base em sua assinatura ativa. Armazenado como JSON na coluna `features` do User.
// Em vez de usar "if premium {}", o codigo deve verificar campos especificos
// como ManagedPremiumAccount, CustomEmojis, etc.
type UserFeatures struct {
	ManagedPremiumAccount bool `json:"managedPremiumAccount,omitempty"`
	CustomEmojis          bool `json:"customEmojis,omitempty"`
	ExtraChannels         int  `json:"extraChannels,omitempty"`
}
