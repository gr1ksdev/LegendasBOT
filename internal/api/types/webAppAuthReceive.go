package types

type WebAPPAuthRequest struct {
	ChannelID int64 `json:"channelID"`
	User      struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		PhotoURL  string `json:"photo_url"`
		AuthDate  string `json:"auth_date"`
		Hash      string `json:"hash"`
	} `json:"user"`
}

type MeChannelsAuthRequest struct {
	User struct {
		ID        int64  `json:"id"`
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Username  string `json:"username"`
		PhotoURL  string `json:"photo_url"`
		AuthDate  string `json:"auth_date"`
		Hash      string `json:"hash"`
	} `json:"user"`
}

type ValidateResult struct {
	IsValid bool
	Data    map[string]string
}
