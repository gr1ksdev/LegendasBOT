package admincontroller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
)

// AdminSubscriptionController gerencia endpoints admin de assinaturas.
type AdminSubscriptionController struct {
	svc *services.SubscriptionService
}

// NewAdminSubscriptionController cria um novo controller.
func NewAdminSubscriptionController(svc *services.SubscriptionService) *AdminSubscriptionController {
	return &AdminSubscriptionController{svc: svc}
}

// ListSubscriptions retorna todas as assinaturas.
// GET /api/admin/subscriptions
func (ctrl *AdminSubscriptionController) ListSubscriptions(ctx *gin.Context) {
	subs, err := ctrl.svc.AdminListSubscriptions(ctx.Request.Context())
	if err != nil {
		ctx.Error(err)
		return
	}
	ctx.JSON(http.StatusOK, types.NewSuccessResponse(subs))
}

type cancelRequest struct {
	UserIDs []int64 `json:"userIds"`
	Instant bool    `json:"instant"`
}

// Cancel cancela/expiura assinaturas de um ou mais usuarios.
// POST /api/admin/subscriptions/cancel
func (ctrl *AdminSubscriptionController) Cancel(ctx *gin.Context) {
	var req cancelRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(err)
		return
	}

	if len(req.UserIDs) == 0 {
		ctx.JSON(http.StatusBadRequest, types.NewErrorResponse("Nenhum usuario informado"))
		return
	}

	var errorsList []string
	for _, userID := range req.UserIDs {
		if err := ctrl.svc.AdminCancelSubscription(ctx.Request.Context(), userID, req.Instant); err != nil {
			errorsList = append(errorsList, err.Error())
		}
	}

	status := http.StatusOK
	message := "Assinaturas canceladas com sucesso"
	if len(errorsList) > 0 {
		status = http.StatusMultiStatus
		message = "Algumas assinaturas nao foram encontradas"
	}

	ctx.JSON(status, types.NewSuccessResponse(gin.H{
		"message": message,
		"errors":  errorsList,
	}))
}

type refundRequest struct {
	UserID   int64  `json:"userId"`
	ChargeID string `json:"telegramPaymentChargeId"`
}

// Refund reembolsa o pagamento Stars de um usuario e expira a assinatura.
// POST /api/admin/subscriptions/refund
func (ctrl *AdminSubscriptionController) Refund(ctx *gin.Context) {
	var req refundRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, types.NewErrorResponse("Dados invalidos: "+err.Error()))
		return
	}

	if req.UserID == 0 || req.ChargeID == "" {
		ctx.JSON(http.StatusBadRequest, types.NewErrorResponse("userId e telegramPaymentChargeId sao obrigatorios"))
		return
	}

	adminID := auth.GetUserID(ctx)
	if adminID == 0 {
		adminID = req.UserID // fallback: se nao conseguiu extrair admin, usa o proprio userID
	}

	if err := ctrl.svc.AdminRefundPayment(ctx.Request.Context(), req.UserID, req.ChargeID, adminID); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Pagamento reembolsado e assinatura expirada com sucesso.",
	}))
}
