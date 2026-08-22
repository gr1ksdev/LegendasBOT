package models

import "time"

// SchemaMigration registra as migrações manuais/DDL executadas no banco de dados para evitar reexecuções.
type SchemaMigration struct {
	ID        string    `gorm:"primaryKey;type:varchar(128)" json:"id"`
	AppliedAt time.Time `gorm:"not null" json:"applied_at"`
}
