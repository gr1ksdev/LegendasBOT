package controllers

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
)

// SubscriptionController gerencia os endpoints de assinatura premium.
type SubscriptionController struct {
	svc *services.SubscriptionService
}

// NewSubscriptionController cria um novo controller de assinaturas.
func NewSubscriptionController(svc *services.SubscriptionService) *SubscriptionController {
	return &SubscriptionController{svc: svc}
}

// GetSubscription retorna o status da assinatura do usuario autenticado.
// GET /api/subscription
func (ctrl *SubscriptionController) GetSubscription(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)

	status, err := ctrl.svc.GetStatus(ctx.Request.Context(), userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(status))
}

// CreateInvoice cria uma invoice do Telegram Stars para assinatura.
// POST /api/subscription/create?test=true (modo teste, ativa direto sem pagamento)
// POST /api/subscription/create?channels=3 (numero de canais a incluir)
func (ctrl *SubscriptionController) CreateInvoice(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	testMode := ctx.Query("test") == "true"

	channelCount := 1
	if c := ctx.Query("channels"); c != "" {
		if n, err := parseInt64(c); err == nil && n > 0 {
			channelCount = int(n)
		}
	}

	result, err := ctrl.svc.CreateInvoice(ctx.Request.Context(), userID, testMode, channelCount)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"invoiceUrl": result.InvoiceURL,
		"payload":    result.Payload,
		"totalStars": result.TotalStars,
	}))
}

// parseInt64 e um helper simples para converter string em int64.
func parseInt64(s string) (int64, error) {
	var n int64
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int64(c-'0')
	}
	return n, nil
}

// CreateExtraChannelInvoice cria uma invoice para adicionar um canal extra.
// POST /api/subscription/channels/add-invoice?test=true
func (ctrl *SubscriptionController) CreateExtraChannelInvoice(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)
	testMode := ctx.Query("test") == "true"

	result, err := ctrl.svc.CreateExtraChannelInvoice(ctx.Request.Context(), userID, testMode)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"invoiceUrl": result.InvoiceURL,
		"payload":    result.Payload,
		"totalStars": result.TotalStars,
	}))
}

// Cancel cancela a assinatura do usuario no fim do periodo.
// POST /api/subscription/cancel
func (ctrl *SubscriptionController) Cancel(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)

	if err := ctrl.svc.Cancel(ctx.Request.Context(), userID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Assinatura será cancelada ao final do período atual.",
	}))
}

// AddExtraChannel adiciona um canal extra a assinatura.
// POST /api/subscription/channels/add
func (ctrl *SubscriptionController) AddExtraChannel(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)

	if err := ctrl.svc.AddExtraChannel(ctx.Request.Context(), userID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Canal extra adicionado com sucesso.",
	}))
}

// RemoveExtraChannel remove um canal extra da assinatura.
// POST /api/subscription/channels/remove
func (ctrl *SubscriptionController) RemoveExtraChannel(ctx *gin.Context) {
	userID := auth.GetUserID(ctx)

	if err := ctrl.svc.RemoveExtraChannel(ctx.Request.Context(), userID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Canal extra removido com sucesso.",
	}))
}
