package controllers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

const maxPermissionRequestBytes = 64 << 10

func bindLimitedJSON(c *gin.Context, target any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxPermissionRequestBytes)
	return c.ShouldBindJSON(target)
}

type PermissionController struct {
	container *container.AppContainer
}

func NewPermissionController(container *container.AppContainer) *PermissionController {
	return &PermissionController{
		container: container,
	}
}

func (ctrl *PermissionController) UpdateMessagePermissionController(c *gin.Context) {
	channelIDStr := c.Param("channelId")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		c.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body types.UpdateMessagePermissionRequest
	if err := bindLimitedJSON(c, &body); err != nil {
		c.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	result, err := ctrl.container.PermissionsService.UpdateMessagePermission(c, channelID, body)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(result))

}

func (ctrl *PermissionController) UpdateButtonsPermissionController(c *gin.Context) {
	channelIDStr := c.Param("channelId")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		c.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body types.UpdateButtonsPermissionRequest
	if err := bindLimitedJSON(c, &body); err != nil {
		c.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	result, err := ctrl.container.PermissionsService.UpdateButtonsPermission(c, channelID, body)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(result))

}

func (ctrl *PermissionController) UpdateReactionsActiveController(c *gin.Context) {
	channelIDStr := c.Param("channelId")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		c.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body struct {
		Active bool `json:"active"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	result, err := ctrl.container.PermissionsService.UpdateReactionsActive(c, channelID, body.Active)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(result))
}

func (ctrl *PermissionController) UpdateDynamicLinksController(c *gin.Context) {
	channelIDStr := c.Param("channelId")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		c.Error(errors.BadRequest("channelId inválido"))
		return
	}

	var body struct {
		Active         bool `json:"dynamicLinks"`
		DLBotButtons   bool `json:"dlBotButtons"`
		DLBotCaptions  bool `json:"dlBotCaptions"`
		DLBotReactions bool `json:"dlBotReactions"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	settings := map[string]any{
		"dynamic_links":    body.Active,
		"dl_bot_buttons":   body.DLBotButtons,
		"dl_bot_captions":  body.DLBotCaptions,
		"dl_bot_reactions": body.DLBotReactions,
	}

	err = ctrl.container.ChannelService.UpdateDynamicLinks(c, channelID, settings)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(body))
}
