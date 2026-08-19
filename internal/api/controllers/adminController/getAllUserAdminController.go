package admincontroller

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

type UsersAdminController struct {
	container *container.AppContainer
}

func NewUsersAdminController(app *container.AppContainer) *UsersAdminController {
	return &UsersAdminController{
		container: app,
	}
}

type NoticeRequest struct {
	Message   string  `json:"message"`
	Target    string  `json:"target"`
	TargetID  int64   `json:"targetId"`
	TargetIDs []int64 `json:"targetIds"`
	ImageUrl  string  `json:"imageUrl"`
	Buttons   []struct {
		Text  string `json:"text"`
		Type  string `json:"type"`
		Value string `json:"value"`
	} `json:"buttons"`
}

func (c *UsersAdminController) GetAllUsersAdminController(ctx *gin.Context) {
	users, err := c.container.UserService.GetAllUsersWithChannels(ctx)

	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(users))
}

func (c *UsersAdminController) UpdateUserAdminController(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID de usuário inválido"))
		return
	}

	newValue, err := c.container.UserService.UpdateUserAdmin(ctx, userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"isAdmin": newValue}, "Status de admin atualizado"))
}

func (c *UsersAdminController) UpdateUserBlacklistController(ctx *gin.Context) {
	userIDStr := ctx.Param("userId")
	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID de usuário inválido"))
		return
	}

	newValue, err := c.container.UserService.UpdateUserBlacklist(ctx, userID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"isBlacklisted": newValue}, "Status de blacklist atualizado"))
}

func (c *UsersAdminController) GetAdminOverview(ctx *gin.Context) {
	users, err := c.container.UserService.GetAllUsersWithChannels(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	channels, err := c.container.ChannelService.GetAllChannels(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"users":    users,
		"channels": channels,
	}))
}

func (c *UsersAdminController) SendNoticeAdminController(ctx *gin.Context) {
	var notice NoticeRequest

	if err := ctx.ShouldBindJSON(&notice); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Broadcast iniciado"))

	go c.dispatchNotice(notice)
}

func (c *UsersAdminController) dispatchNotice(notice NoticeRequest) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("ADMIN", "Panic recuperado em dispatchNotice: %v", r)
		}
	}()
	ctx := context.Background()

	var buttons []container.BroadcastButton
	for _, btn := range notice.Buttons {
		buttons = append(buttons, container.BroadcastButton{
			Text:  btn.Text,
			Type:  btn.Type,
			Value: btn.Value,
		})
	}

	baseText := utils.MarkdownToTelegramHTML(notice.Message)
	supportText := "# 📨 <b>MENSAGEM DO SUPORTE</b>\n\n" + baseText
	enqueueTargets := func(ids []int64, text string) {
		sent := make(map[int64]bool, len(ids))
		for _, id := range ids {
			if id == 0 || sent[id] {
				continue
			}
			c.container.BroadcastQueue <- container.BroadcastJob{
				ChatID:   id,
				Text:     text,
				ImageUrl: notice.ImageUrl,
				Buttons:  buttons,
			}
			sent[id] = true
		}
	}

	switch notice.Target {

	case "single":
		if notice.TargetID == 0 {
			logger.Error("API", "TargetID ausente para envio individual")
			return
		}

		enqueueTargets([]int64{notice.TargetID}, supportText)

	case "user_ids":
		if len(notice.TargetIDs) == 0 {
			logger.Error("API", "TargetIDs ausentes para envio a usuários específicos")
			return
		}

		enqueueTargets(notice.TargetIDs, supportText)

	case "channel_ids":
		if len(notice.TargetIDs) == 0 {
			logger.Error("API", "TargetIDs ausentes para envio a canais específicos")
			return
		}

		enqueueTargets(notice.TargetIDs, baseText)

	case "users":
		users, err := c.container.UserService.GetAllUsersWithChannels(ctx)
		if err != nil {
			logger.Error("API", "Erro ao buscar usuários para broadcast: %v", err)
			return
		}

		for _, user := range users {
			c.container.BroadcastQueue <- container.BroadcastJob{
				ChatID:   user.UserId,
				Text:     baseText,
				ImageUrl: notice.ImageUrl,
				Buttons:  buttons,
			}
		}

	case "channels":
		channels, err := c.container.ChannelService.GetAllChannels(ctx)
		if err != nil {
			logger.Error("API", "Erro ao buscar usuários para broadcast: %v", err)
			return
		}

		for _, channel := range channels {
			c.container.BroadcastQueue <- container.BroadcastJob{
				ChatID:   channel.ID,
				Text:     baseText,
				ImageUrl: notice.ImageUrl,
				Buttons:  buttons,
			}
		}

	case "all":
		users, _ := c.container.UserService.GetAllUsersWithChannels(ctx)
		channels, _ := c.container.ChannelService.GetAllChannels(ctx)

		sentMap := make(map[int64]bool)

		for _, user := range users {
			if !sentMap[user.UserId] {
				c.container.BroadcastQueue <- container.BroadcastJob{
					ChatID:   user.UserId,
					Text:     baseText,
					ImageUrl: notice.ImageUrl,
					Buttons:  buttons,
				}
				sentMap[user.UserId] = true
			}
		}

		for _, channel := range channels {
			if !sentMap[channel.ID] {
				c.container.BroadcastQueue <- container.BroadcastJob{
					ChatID:   channel.ID,
					Text:     baseText,
					ImageUrl: notice.ImageUrl,
					Buttons:  buttons,
				}
				sentMap[channel.ID] = true
			}
		}
	}
}
