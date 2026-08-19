package types

// AccountStatusResponse representa o status da conta conectada.
type AccountStatusResponse struct {
	Status      string  `json:"status"` // "connected", "disconnected"
	TelegramID  *int64  `json:"telegramId,omitempty"`
	Username    *string `json:"username,omitempty"`
	FirstName   *string `json:"firstName,omitempty"`
	AvatarURL   *string `json:"avatarUrl,omitempty"`
	ConnectedAt *string `json:"connectedAt,omitempty"`
	LastUsedAt  *string `json:"lastUsedAt,omitempty"`
}

// ConnectRequest representa o request de inicio de autenticacao.
type ConnectRequest struct {
	PhoneNumber string `json:"phoneNumber" binding:"required"`
}

// VerifyRequest representa o request de verificacao de codigo.
type VerifyRequest struct {
	Code string `json:"code" binding:"required"`
}

// PasswordRequest representa o request de senha 2FA.
type PasswordRequest struct {
	Password string `json:"password" binding:"required"`
}
