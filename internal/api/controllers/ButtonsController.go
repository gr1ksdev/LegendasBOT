package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/dto"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type ButtonsController struct {
	container *container.AppContainer
}

func NewButtonsController(container *container.AppContainer) *ButtonsController {
	return &ButtonsController{
		container: container,
	}
}

func (c *ButtonsController) CreateDefaultButtonController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var buttonData types.ButtonCreateRequest
	if err := ctx.ShouldBindJSON(&buttonData); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	result, err := c.container.ButtonService.CreateButton(ctx, channelId, buttonData)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(dto.ToButtonDTO(result), "Botão criado com sucesso"))
}

func (c *ButtonsController) DeleteDefaultButtonController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	buttonID := ctx.Param("buttonId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	_, err = c.container.ButtonService.DeleteButton(ctx, channelId, buttonID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Botão deletado com sucesso"))
}

func (c *ButtonsController) UpdateDefaultButtonController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	buttonID := ctx.Param("buttonId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var buttonData types.ButtonCreateRequest
	if err := ctx.ShouldBindJSON(&buttonData); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	rowsAffected, err := c.container.ButtonService.UpdateButton(ctx, channelId, buttonID, buttonData)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Botão padrao atualizado com sucesso"))
}

func (c *ButtonsController) UpdateLayoutDefaultButtons(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var body types.UpdateLayoutRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	total, err := c.container.ButtonService.UpdateButtonsLayout(ctx, channelId, body)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"total": total}, "Layout dos botões atualizado com sucesso"))
}
