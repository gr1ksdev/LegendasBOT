package models

import "time"

type UserCaptionTemplate struct {
	ID        string                      `gorm:"type:text;primaryKey" json:"id"`
	UserID    int64                       `gorm:"index:idx_user_template_code,unique" json:"userId"`
	Code      string                      `gorm:"index:idx_user_template_code,unique" json:"code"`
	Caption   string                      `gorm:"type:text" json:"caption"`
	Reactions string                      `gorm:"type:text" json:"reactions"`
	Buttons   []UserCaptionTemplateButton `gorm:"foreignKey:OwnerTemplateID;constraint:OnDelete:CASCADE;" json:"buttons"`
	CreatedAt time.Time                   `gorm:"autoCreateTime" json:"createdAt"`
	UpdatedAt time.Time                   `gorm:"autoUpdateTime" json:"updatedAt"`
}

type UserCaptionTemplateButton struct {
	ButtonID        string `gorm:"type:text;primaryKey" json:"buttonId"`
	NameButton      string `gorm:"type:text;not null" json:"nameButton"`
	ButtonURL       string `gorm:"type:text" json:"buttonUrl"`
	Style           string `gorm:"type:text" json:"style,omitempty"`
	PositionX       int    `gorm:"default:0" json:"positionX"`
	PositionY       int    `gorm:"default:0" json:"positionY"`
	OwnerTemplateID string `gorm:"index" json:"ownerTemplateId"`
}
