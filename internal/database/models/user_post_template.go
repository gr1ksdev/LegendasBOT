package models

import "time"

type UserPostTemplate struct {
	ID           string    `gorm:"type:text;primaryKey" json:"id"`
	OwnerID      int64     `gorm:"index" json:"ownerId"`
	Name         string    `gorm:"type:text" json:"name"`
	TemplateData string    `gorm:"type:text" json:"templateData"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updatedAt"`
}
