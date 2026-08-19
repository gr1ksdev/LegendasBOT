package cache

type ChannelPayload struct {
	ChannelID  int64  `json:"channel_id"`
	Title      string `json:"title,omitempty"`
	OwnerID    int64  `json:"owner_id"`
	NewOwnerID int64  `json:"new_owner_id,omitempty"`
}

type Session struct {
	Key     string         `json:"key"`
	Payload ChannelPayload `json:"payload"`
}

type PostBuilderButton struct {
	Text          string `json:"text"`
	URL           string `json:"url"`
	Style         string `json:"style,omitempty"`
	CustomEmojiID string `json:"custom_emoji_id,omitempty"`
}

type PostBuilderState struct {
	MediaType         string              `json:"media_type"`
	MediaFileID       string              `json:"media_file_id"`
	MenuMessageID     int                 `json:"menu_message_id"`
	PromptMessageID   int                 `json:"prompt_message_id"`
	Title             string              `json:"title"`
	Body              string              `json:"body"`
	Footer            string              `json:"footer"`
	Reactions         string              `json:"reactions"`
	Buttons           []PostBuilderButton `json:"buttons"`
	Step              string              `json:"step"`
	TemplateSessionID string              `json:"template_session_id,omitempty"`
	AutoDeleteMin     int                 `json:"auto_delete_min,omitempty"`
}

type ScheduleState struct {
	SessionID    string `json:"session_id"`
	ChannelID    string `json:"channel_id"`
	ScheduleType string `json:"schedule_type"`
}
