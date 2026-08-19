package types

type UpdateMessagePermissionRequest struct {
	LinkPreview *bool `json:"linkPreview" binding:"required"`
	Message     *bool `json:"message" binding:"required"`
	Audio       *bool `json:"audio" binding:"required"`
	Video       *bool `json:"video" binding:"required"`
	Photo       *bool `json:"photo" binding:"required"`
	Document    *bool `json:"document" binding:"required"`
	Sticker     *bool `json:"sticker" binding:"required"`
	GIF         *bool `json:"gif" binding:"required"`
	Reactions   *bool `json:"reactions" binding:"required"`
}

type UpdateButtonsPermissionRequest struct {
	Message  *bool `json:"message" binding:"required"`
	Audio    *bool `json:"audio" binding:"required"`
	Video    *bool `json:"video" binding:"required"`
	Photo    *bool `json:"photo" binding:"required"`
	Document *bool `json:"document" binding:"required"`
	Sticker  *bool `json:"sticker" binding:"required"`
	GIF      *bool `json:"gif" binding:"required"`
}

type UpdatePermissionsResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
