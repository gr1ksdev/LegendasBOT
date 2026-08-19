package types

type CreateCustomCaptionRequest struct {
	Code        string `json:"code" binding:"required"`
	Caption     string `json:"caption" binding:"required"`
	LinkPreview bool   `json:"linkPreview"`
}

type CreateCustomCaptionResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data"`
}

type CustomCaptionLayoutItem struct {
	ID string `json:"buttonId" binding:"required"`
}

type UpdateCustomCaptionLayoutRequest struct {
	Layout [][]CustomCaptionLayoutItem `json:"layout" binding:"required"`
}
