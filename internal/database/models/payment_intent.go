package models

import "time"

const (
	PaymentIntentPending    = "pending"
	PaymentIntentProcessing = "processing"
	PaymentIntentPaid       = "paid"
	PaymentIntentExpired    = "expired"

	PaymentIntentSubscription = "subscription"
	PaymentIntentExtraChannel = "extra_channel"
)

// PaymentIntent registra imutavelmente o que o usuario pretende comprar antes
// de entregar um link de invoice ao Telegram.
type PaymentIntent struct {
	ID            string     `gorm:"type:text;primaryKey" json:"id"`
	Payload       string     `gorm:"type:text;uniqueIndex;not null" json:"payload"`
	UserID        int64      `gorm:"index;not null" json:"userId"`
	Type          string     `gorm:"type:text;not null" json:"type"`
	ExtraChannels int        `gorm:"default:0" json:"extraChannels"`
	AmountStars   int        `gorm:"not null" json:"amountStars"`
	Status        string     `gorm:"type:text;index;not null" json:"status"`
	ChargeID      string     `gorm:"type:text;uniqueIndex" json:"chargeId,omitempty"`
	ExpiresAt     time.Time  `gorm:"index;not null" json:"expiresAt"`
	PaidAt        *time.Time `json:"paidAt,omitempty"`
	CreatedAt     time.Time  `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt     time.Time  `gorm:"autoUpdateTime" json:"updatedAt"`
}

func (PaymentIntent) TableName() string { return "payment_intents" }
