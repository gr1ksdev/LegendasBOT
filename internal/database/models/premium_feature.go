package models

import "time"

// PremiumFeature representa uma feature/configuracao do sistema de assinatura
// premium. Admins podem ativar/desativar features globalmente e definir precos.
//
// A chave primaria e o identificador unico da feature (ex: "managed_premium_account").
type PremiumFeature struct {
	Key         string    `gorm:"primaryKey;type:text" json:"key"`
	Name        string    `gorm:"type:text;not null" json:"name"`
	Description string    `gorm:"type:text" json:"description"`
	Enabled     bool      `gorm:"default:true" json:"enabled"`
	Price       int       `gorm:"default:0" json:"price"` // precos em Stars (0 = gratuito/incluso)
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}

// TableName retorna o nome da tabela no banco.
func (PremiumFeature) TableName() string {
	return "premium_features"
}
