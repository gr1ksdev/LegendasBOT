package types

import (
	"github.com/leirbagxis/FreddyBot/internal/database/models"
)

type ButtonCreateRequest struct {
	NameButton string `json:"nameButton" binding:"required"`
	ButtonURL  string `json:"buttonUrl,omitempty"`
	Style      string `json:"style,omitempty"`
}

type ButtonCreateResponse struct {
	Success bool           `json:"success"`
	Message string         `json:"message"`
	Data    *models.Button `json:"data,omitempty"`
}

type ButtonResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}

type ButtonPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type ButtonLayoutItem struct {
	ID string `json:"buttonId" binding:"required"`
}

type UpdateLayoutRequest struct {
	Layout [][]ButtonLayoutItem `json:"layout" binding:"required"`
}

type UpdateLayoutResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}
