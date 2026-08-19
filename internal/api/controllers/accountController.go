package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

// AccountController gerencia os endpoints de contas conectadas.
type AccountController struct {
	container *container.AppContainer
}

// NewAccountController cria um novo controller de contas.
func NewAccountController(container *container.AppContainer) *AccountController {
	return &AccountController{container: container}
}

// GetAccountStatus retorna o status da conta conectada do usuario.
func (c *AccountController) GetAccountStatus(ctx *gin.Context) {
	uid := auth.GetUserID(ctx)
	if uid == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	account, err := c.container.ConnectedAccountService.GetAccount(ctx, uid)
	if err != nil {
		ctx.Error(err)
		return
	}

	if account == nil {
		ctx.JSON(http.StatusOK, types.NewSuccessResponse(types.AccountStatusResponse{
			Status: "disconnected",
		}))
		return
	}

	connectedAt := account.CreatedAt.Format("2006-01-02T15:04:05Z07:00")
	var lastUsedAt *string
	if account.LastUsedAt != nil {
		formatted := account.LastUsedAt.Format("2006-01-02T15:04:05Z07:00")
		lastUsedAt = &formatted
	}

	var avatarURL *string
	if account.Username != "" {
		u := "https://t.me/i/userpic/320/" + account.Username + ".jpg"
		avatarURL = &u
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(types.AccountStatusResponse{
		Status:      "connected",
		TelegramID:  &account.TelegramUserID,
		Username:    &account.Username,
		FirstName:   &account.FirstName,
		AvatarURL:   avatarURL,
		ConnectedAt: &connectedAt,
		LastUsedAt:  lastUsedAt,
	}))
}

// ConnectAccount inicia o fluxo de autenticacao.
func (c *AccountController) ConnectAccount(ctx *gin.Context) {
	uid := auth.GetUserID(ctx)
	if uid == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	var req types.ConnectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Número de telefone é obrigatório"))
		return
	}

	// Validar formato do telefone
	if len(req.PhoneNumber) < 8 || len(req.PhoneNumber) > 20 {
		ctx.Error(errors.BadRequest("Número de telefone inválido"))
		return
	}

	// Iniciar autenticacao via MTProto
	status, err := c.container.MTProtoAuthService.SendCode(ctx, uid, req.PhoneNumber)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Erro ao enviar código: "+err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// VerifyCode verifica o codigo de autenticacao.
func (c *AccountController) VerifyCode(ctx *gin.Context) {
	uid := auth.GetUserID(ctx)
	if uid == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	var req types.VerifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Code == "" {
		ctx.Error(errors.BadRequest("Código é obrigatório"))
		return
	}

	status, err := c.container.MTProtoAuthService.VerifyCode(ctx, uid, req.Code)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Erro ao verificar código: "+err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// SendPassword envia a senha 2FA.
func (c *AccountController) SendPassword(ctx *gin.Context) {
	uid := auth.GetUserID(ctx)
	if uid == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	var req types.PasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Password == "" {
		ctx.Error(errors.BadRequest("Senha é obrigatória"))
		return
	}

	status, err := c.container.MTProtoAuthService.VerifyPassword(ctx, uid, req.Password)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Erro ao verificar senha: "+err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// DisconnectAccount remove a conta conectada do usuario.
func (c *AccountController) DisconnectAccount(ctx *gin.Context) {
	uid := auth.GetUserID(ctx)
	if uid == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	// Limpar cache do factory
	if c.container.ExecutorFactory != nil {
		c.container.ExecutorFactory.InvalidateCache(uid)
	}

	if err := c.container.ConnectedAccountService.Disconnect(ctx, uid); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Conta desconectada com sucesso",
	}))
}

// GetAuthStatus returns the current auth step status for the user.
func (c *AccountController) GetAuthStatus(ctx *gin.Context) {
	uid := auth.GetUserID(ctx)
	if uid == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	status, err := c.container.MTProtoAuthService.GetStatus(ctx, uid)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}
