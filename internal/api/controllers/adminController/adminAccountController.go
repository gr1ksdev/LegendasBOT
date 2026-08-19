package admincontroller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

// AdminAccountController gerencia os endpoints de contas MTProto de admin.
type AdminAccountController struct {
	container *container.AppContainer
}

// NewAdminAccountController cria um novo controller de contas admin.
func NewAdminAccountController(container *container.AppContainer) *AdminAccountController {
	return &AdminAccountController{container: container}
}

// --- Request Types ---

// AdminConnectRequest representa o request de inicio de autenticacao admin.
type AdminConnectRequest struct {
	Label       string `json:"label" binding:"required"`
	PhoneNumber string `json:"phoneNumber" binding:"required"`
}

// AdminVerifyRequest representa o request de verificacao de codigo admin.
type AdminVerifyRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	Code      string `json:"code" binding:"required"`
}

// AdminPasswordRequest representa o request de senha 2FA admin.
type AdminPasswordRequest struct {
	SessionID string `json:"sessionId" binding:"required"`
	Password  string `json:"password" binding:"required"`
}

// AdminToggleRequest representa o request de toggle enable/disable.
type AdminToggleRequest struct {
	Enabled bool `json:"enabled"`
}

// --- Handlers ---

// ListAccounts retorna todas as contas MTProto de admin.
// GET /api/admin/accounts
func (ctrl *AdminAccountController) ListAccounts(ctx *gin.Context) {
	accounts, err := ctrl.container.AdminAccountService.ListAccounts(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
	}

	if accounts == nil {
		accounts = []models.AdminMTProtoAccount{}
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(accounts))
}

// ConnectAccount inicia o fluxo de autenticacao para uma nova conta admin.
// POST /api/admin/accounts/connect
func (ctrl *AdminAccountController) ConnectAccount(ctx *gin.Context) {
	var req AdminConnectRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Label e número de telefone são obrigatórios"))
		return
	}

	if len(req.PhoneNumber) < 8 || len(req.PhoneNumber) > 20 {
		ctx.Error(errors.BadRequest("Número de telefone inválido"))
		return
	}

	if req.Label == "" {
		ctx.Error(errors.BadRequest("Label é obrigatório"))
		return
	}

	status, err := ctrl.container.AdminAccountService.StartAuth(ctx.Request.Context(), req.Label, req.PhoneNumber)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Erro ao enviar código: "+err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// VerifyCode verifica o codigo de autenticacao.
// POST /api/admin/accounts/verify
func (ctrl *AdminAccountController) VerifyCode(ctx *gin.Context) {
	var req AdminVerifyRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Code == "" || req.SessionID == "" {
		ctx.Error(errors.BadRequest("Código e sessionId são obrigatórios"))
		return
	}

	status, err := ctrl.container.AdminAccountService.VerifyCode(ctx.Request.Context(), req.SessionID, req.Code)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Erro ao verificar código: "+err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// SendPassword envia a senha 2FA.
// POST /api/admin/accounts/password
func (ctrl *AdminAccountController) SendPassword(ctx *gin.Context) {
	var req AdminPasswordRequest
	if err := ctx.ShouldBindJSON(&req); err != nil || req.Password == "" || req.SessionID == "" {
		ctx.Error(errors.BadRequest("Senha e sessionId são obrigatórios"))
		return
	}

	status, err := ctrl.container.AdminAccountService.VerifyPassword(ctx.Request.Context(), req.SessionID, req.Password)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Erro ao verificar senha: "+err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// DeleteAccount remove uma conta admin.
// DELETE /api/admin/accounts/:id
func (ctrl *AdminAccountController) DeleteAccount(ctx *gin.Context) {
	accountID := ctx.Param("id")
	if accountID == "" {
		ctx.Error(errors.BadRequest("ID da conta é obrigatório"))
		return
	}

	if err := ctrl.container.AdminAccountService.DeleteAccount(ctx.Request.Context(), accountID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Conta removida com sucesso",
	}))
}

// ToggleAccount ativa/desativa uma conta admin.
// POST /api/admin/accounts/:id/toggle
func (ctrl *AdminAccountController) ToggleAccount(ctx *gin.Context) {
	accountID := ctx.Param("id")
	if accountID == "" {
		ctx.Error(errors.BadRequest("ID da conta é obrigatório"))
		return
	}

	var req AdminToggleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("enabled é obrigatório"))
		return
	}

	account, err := ctrl.container.AdminAccountService.ToggleAccount(ctx.Request.Context(), accountID, req.Enabled)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(account))
}
