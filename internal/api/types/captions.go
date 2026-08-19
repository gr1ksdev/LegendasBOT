package types

type CaptionDefaultUpdateRequest struct {
	Caption string `json:"caption" binding:"required"`
}

type NewPackCaptionUpdateRequest struct {
	Caption                string  `json:"caption"`
	NewPackCaption         string  `json:"newPackCaption"`
	NewPackMessageButtons  *bool   `json:"newPackMessageButtons"`
	NewPackStickerButtons  *bool   `json:"newPackStickerButtons"`
	NewPackMessagePosition *string `json:"newPackMessagePosition"`
	NewPackReplyToSticker  *bool   `json:"newPackReplyToSticker"`
}

func (r NewPackCaptionUpdateRequest) Text() string {
	if r.NewPackCaption != "" {
		return r.NewPackCaption
	}
	return r.Caption
}

type ReactionsUpdateRequest struct {
	Reactions string `json:"reactions"`
}

type ReactionPositionUpdateRequest struct {
	ReactionPosition int `json:"reactionPosition"`
}

type NativeReactionsUpdateRequest struct {
	NativeReactions string `json:"nativeReactions"`
}

type NativeReactionModeRequest struct {
	Mode string `json:"mode" binding:"required"`
}

type NativeReactionsEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type CaptionUpdateResponse struct {
	Success bool                   `json:"success"`
	Message string                 `json:"message"`
	Data    map[string]interface{} `json:"data,omitempty"`
}
