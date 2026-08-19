package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type UserCaptionTemplateController struct {
	container *container.AppContainer
}

func NewUserCaptionTemplateController(container *container.AppContainer) *UserCaptionTemplateController {
	return &UserCaptionTemplateController{
		container: container,
	}
}

func (ctrl *UserCaptionTemplateController) List(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	templates, err := ctrl.container.UserCaptionTemplateService.List(ctx, userID.(int64))
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(templates, ""))
}

func (ctrl *UserCaptionTemplateController) Create(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	var body struct {
		Code      string `json:"code" binding:"required"`
		Caption   string `json:"caption"`
		Reactions string `json:"reactions"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("nome curto é obrigatório"))
		return
	}

	tpl, err := ctrl.container.UserCaptionTemplateService.Create(ctx, userID.(int64), body.Code, body.Caption, body.Reactions)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, types.NewSuccessResponse(tpl, "Template criado"))
}

func (ctrl *UserCaptionTemplateController) Get(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	tpl, err := ctrl.container.UserCaptionTemplateService.GetByID(ctx, ctx.Param("id"))
	if err != nil {
		ctx.Error(err)
		return
	}

	if tpl.UserID != userID.(int64) {
		ctx.Error(errors.ErrForbidden)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(tpl, ""))
}

func (ctrl *UserCaptionTemplateController) Update(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	var body struct {
		Code      string `json:"code" binding:"required"`
		Caption   string `json:"caption"`
		Reactions string `json:"reactions"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("nome curto é obrigatório"))
		return
	}

	if err := ctrl.container.UserCaptionTemplateService.Update(ctx, userID.(int64), ctx.Param("id"), body.Code, body.Caption, body.Reactions); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Template atualizado"))
}

func (ctrl *UserCaptionTemplateController) CreateButton(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	tpl, err := ctrl.container.UserCaptionTemplateService.GetByID(ctx, ctx.Param("id"))
	if err != nil {
		ctx.Error(err)
		return
	}
	if tpl.UserID != userID.(int64) {
		ctx.Error(errors.ErrForbidden)
		return
	}

	var body struct {
		NameButton string `json:"nameButton" binding:"required"`
		ButtonURL  string `json:"buttonUrl"`
		Style      string `json:"style"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("nome do botão é obrigatório"))
		return
	}

	maxY := 0
	for _, b := range tpl.Buttons {
		if b.PositionY >= maxY {
			maxY = b.PositionY + 1
		}
	}

	button := &models.UserCaptionTemplateButton{
		ButtonID:        uuid.NewString(),
		NameButton:      body.NameButton,
		ButtonURL:       body.ButtonURL,
		Style:           body.Style,
		PositionX:       0,
		PositionY:       maxY,
		OwnerTemplateID: ctx.Param("id"),
	}

	if err := ctrl.container.UserCaptionTemplateService.CreateButton(ctx, button); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, types.NewSuccessResponse(button, "Botão adicionado"))
}

func (ctrl *UserCaptionTemplateController) UpdateButton(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	tpl, err := ctrl.container.UserCaptionTemplateService.GetByID(ctx, ctx.Param("id"))
	if err != nil || tpl == nil {
		ctx.Error(errors.ErrNotFound)
		return
	}

	if tpl.UserID != userID.(int64) {
		ctx.Error(errors.ErrForbidden)
		return
	}

	var body struct {
		NameButton string `json:"nameButton" binding:"required"`
		ButtonURL  string `json:"buttonUrl"`
		Style      string `json:"style"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("nome do botão é obrigatório"))
		return
	}

	if err := ctrl.container.UserCaptionTemplateService.UpdateButton(ctx, ctx.Param("id"), ctx.Param("buttonId"), body.NameButton, body.ButtonURL, body.Style); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Botão atualizado"))
}

func (ctrl *UserCaptionTemplateController) DeleteButton(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	tpl, err := ctrl.container.UserCaptionTemplateService.GetByID(ctx, ctx.Param("id"))
	if err != nil {
		ctx.Error(err)
		return
	}
	if tpl.UserID != userID.(int64) {
		ctx.Error(errors.ErrForbidden)
		return
	}

	if err := ctrl.container.UserCaptionTemplateService.DeleteButton(ctx, ctx.Param("id"), ctx.Param("buttonId")); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Botão removido"))
}

func (ctrl *UserCaptionTemplateController) UpdateLayout(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	tpl, err := ctrl.container.UserCaptionTemplateService.GetByID(ctx, ctx.Param("id"))
	if err != nil {
		ctx.Error(err)
		return
	}
	if tpl.UserID != userID.(int64) {
		ctx.Error(errors.ErrForbidden)
		return
	}

	var body types.UpdateLayoutRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("layout inválido"))
		return
	}

	if err := ctrl.container.UserCaptionTemplateService.UpdateLayout(ctx, userID.(int64), ctx.Param("id"), body.Layout); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Layout atualizado"))
}

func (ctrl *UserCaptionTemplateController) Delete(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	if err := ctrl.container.UserCaptionTemplateService.Delete(ctx, ctx.Param("id"), userID.(int64)); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Template excluído"))
}
