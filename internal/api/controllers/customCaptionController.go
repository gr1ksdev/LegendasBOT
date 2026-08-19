package controllers

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/dto"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type CustomCaptionController struct {
	container *container.AppContainer
}

func NewCustomCaptionController(container *container.AppContainer) *CustomCaptionController {
	return &CustomCaptionController{
		container: container,
	}
}

func (ctrl *CustomCaptionController) CreateCustomCaptionController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelID, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body types.CreateCustomCaptionRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("payload inválido: " + err.Error()))
		return
	}

	result, err := ctrl.container.CustomCaptionService.CreateCustomCaption(ctx, channelID, body)
	if err != nil {
		ctx.Error(err)
		return
	}

	// ── Baixar emojis personalizados em background ──
	userID, _ := ctx.Get("userID")
	if uid, ok := userID.(int64); ok && ctrl.container.EmojiService.TextContainsEmoji(body.Caption) {
		go func(uID int64, caption string) {
			_ = ctrl.container.EmojiService.EnsureEmojisForText(context.Background(), uID, caption)
		}(uid, body.Caption)
	}

	ctx.JSON(http.StatusCreated, types.NewSuccessResponse(dto.ToCustomCaptionDTO(result), "Custom caption criada com sucesso"))
}

func (ctrl *CustomCaptionController) UpdateCustomCaptionController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	captionID := ctx.Param("captionId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var body types.CreateCustomCaptionRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("payload inválido: " + err.Error()))
		return
	}

	rowsAffected, err := ctrl.container.CustomCaptionService.UpdateCustomCaption(ctx, channelId, captionID, body)
	if err != nil {
		ctx.Error(err)
		return
	}

	// ── Baixar emojis personalizados em background ──
	userID, _ := ctx.Get("userID")
	if uid, ok := userID.(int64); ok && ctrl.container.EmojiService.TextContainsEmoji(body.Caption) {
		go func(uID int64, caption string) {
			_ = ctrl.container.EmojiService.EnsureEmojisForText(context.Background(), uID, caption)
		}(uid, body.Caption)
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Custom caption atualizada com sucesso"))
}

func (ctrl *CustomCaptionController) DeleteCustomCaptionController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	captionID := ctx.Param("captionId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	err = ctrl.container.CustomCaptionService.DeleteCustomCaption(ctx, channelId, captionID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Custom caption deletada com sucesso"))
}

// --- BUTTONS LOGIC CONSOLIDATED IN ButtonService ---

func (ctrl *CustomCaptionController) CreateCustomCaptionButtonController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	captionID := ctx.Param("captionId")
	channelID, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body types.ButtonCreateRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("payload inválido: " + err.Error()))
		return
	}

	result, err := ctrl.container.ButtonService.CreateCustomCaptionButton(ctx, channelID, captionID, body)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, types.NewSuccessResponse(dto.ToCustomCaptionButtonDTO(result), "Botão da Custom Caption criado com sucesso"))
}

func (ctrl *CustomCaptionController) UpdateCustomCaptionButtonController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	captionID := ctx.Param("captionId")
	buttonID := ctx.Param("buttonId")
	channelID, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body types.ButtonCreateRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("payload inválido: " + err.Error()))
		return
	}

	rowsAffected, err := ctrl.container.ButtonService.UpdateCustomCaptionButton(ctx, channelID, captionID, buttonID, body)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Botão customizado atualizado com sucesso"))
}

func (ctrl *CustomCaptionController) UpdateCustomCaptionLayoutController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	captionID := ctx.Param("captionId")
	channelID, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body types.UpdateCustomCaptionLayoutRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("payload inválido: " + err.Error()))
		return
	}

	total, err := ctrl.container.ButtonService.UpdateCustomCaptionLayout(ctx, channelID, captionID, body)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"total": total}, "Layout dos botões da custom caption atualizado com sucesso"))
}

func (ctrl *CustomCaptionController) DeleteCustomCaptionButtonController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	captionID := ctx.Param("captionId")
	buttonID := ctx.Param("buttonId")
	channelID, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("channelId inválido"))
		return
	}

	_, err = ctrl.container.ButtonService.DeleteCustomCaptionButton(ctx, channelID, captionID, buttonID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Botão deletado com sucesso"))
}
