package types

type TransferChannelRequest struct {
	NewOwnerID int64 `json:"newOwnerId" binding:"required"`
	ChannelID  int64 `json:"channelId" binding:"required"`
}
