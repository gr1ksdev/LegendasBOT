package controllers

import (
	"context"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/dto"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/leirbagxis/FreddyBot/pkg/parser"
	"github.com/mymmrac/telego"
)

type UserController struct {
	container *container.AppContainer
}

func NewUserController(container *container.AppContainer) *UserController {
	return &UserController{
		container: container,
	}
}

func (c *UserController) GetUserChannelsController(ctx *gin.Context) {
	userID, exists := ctx.Get("userID")
	if !exists {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	channels, err := c.container.ChannelService.GetUserChannels(ctx, userID.(int64))
	if err != nil {
		ctx.Error(err)
		return
	}

	var dtos []dto.ChannelDTO
	for _, ch := range channels {
		dtos = append(dtos, dto.ToChannelDTO(&ch))
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(dtos))
}

func (c *UserController) GetUserInfo(ctx *gin.Context) {
	userParams := ctx.Param("userParams")
	if len(userParams) < 5 {
		ctx.Error(errors.BadRequest("ID ou Username inválido!"))
		return
	}

	userID, _ := strconv.ParseInt(userParams, 10, 64)
	if userID == 0 {
		user, err := c.container.UserService.GetUserByUsername(ctx, userParams)
		if err != nil {
			ctx.Error(err)
			return
		}

		ctx.JSON(http.StatusOK, types.NewSuccessResponse(dto.ToUserLookupDTO(user)))
		return

	}

	user, err := c.container.UserService.GetUserByID(ctx, userID)
	if err != nil {
		ctx.Error(errors.ErrNotFound)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(dto.ToUserLookupDTO(user)))
}

func (c *UserController) TransferChannelController(ctx *gin.Context) {
	var body *types.TransferChannelRequest
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("payload inválido: " + err.Error()))
		return
	}

	actorID := auth.GetUserID(ctx)
	roleValue, _ := ctx.Get("role")
	role, _ := roleValue.(auth.Role)
	if actorID == 0 {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	if body.NewOwnerID == actorID {
		ctx.Error(errors.BadRequest("O novo dono precisa ser diferente de voce."))
		return
	}

	newOwner, err := c.container.TelegoBot.GetChat(ctx, &telego.GetChatParams{ChatID: telego.ChatID{ID: body.NewOwnerID}})
	if err != nil {
		logger.Error("API", "Erro ao obter chat do novo dono: %v", err)
		ctx.Error(errors.BadRequest("O novo dono precisa iniciar o bot pelo menos uma vez."))
		return
	}

	// Verifica se o novo dono é um bot
	botInfo, err := c.container.TelegoBot.GetMe(ctx)
	if err != nil || botInfo == nil {
		logger.Error("API", "Erro ao obter dados do bot via GetMe: %v", err)
		ctx.Error(errors.New(http.StatusInternalServerError, "Erro ao obter dados do bot"))
		return
	}
	if body.NewOwnerID == botInfo.ID {
		ctx.Error(errors.BadRequest("O novo dono não pode ser eu."))
		return
	}

	admins, err := c.container.TelegoBot.GetChatAdministrators(ctx, &telego.GetChatAdministratorsParams{
		ChatID: telego.ChatID{ID: body.ChannelID},
	})
	if err != nil {
		logger.Error("API", "Erro ao buscar administradores do canal: %v", err)
		ctx.Error(errors.New(http.StatusInternalServerError, "Erro ao consultar administradores do canal."))
		return
	}

	isAdmin := false
	for _, admin := range admins {
		status := admin.MemberStatus()
		if status == telego.MemberStatusAdministrator {
			if a, ok := admin.(*telego.ChatMemberAdministrator); ok && a.User.ID == body.NewOwnerID {
				isAdmin = true
				break
			}
		}
		if status == telego.MemberStatusCreator {
			if a, ok := admin.(*telego.ChatMemberOwner); ok && a.User.ID == body.NewOwnerID {
				isAdmin = true
				break
			}
		}
	}

	if !isAdmin {
		ctx.Error(errors.BadRequest("O novo dono precisa ser administrador do canal."))
		return
	}

	channel, err := c.container.ChannelService.TransferChannel(
		ctx,
		actorID,
		body.ChannelID,
		body.NewOwnerID,
		role == auth.RoleAdmin || role == auth.RoleOwner,
	)
	if err != nil {
		ctx.Error(err)
		return
	}

	channelID := fmt.Sprintf("%d", body.ChannelID)
	newOwnerIDStr := fmt.Sprintf("%d", body.NewOwnerID)

	data := map[string]string{
		"channelId":    channelID,
		"channelName":  channel.Title,
		"newOwnerName": newOwner.LastName,
		"newOwnerId":   newOwnerIDStr,
		"miniAppUrl":   auth.GenerateMiniAppUrl(newOwnerIDStr, channelID),
	}

	textOld, buttonOld := parser.GetMessageTelego("success-old-paccess-message", data)
	paramsOld := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: channel.OwnerID},
		Text:      textOld,
		ParseMode: telego.ModeHTML,
	}
	if buttonOld != nil {
		paramsOld.ReplyMarkup = buttonOld
	}
	_, _ = c.container.TelegoBot.SendMessage(context.Background(), paramsOld)

	textNew, buttonNew := parser.GetMessageTelego("success-new-paccess-message", data)
	paramsNew := &telego.SendMessageParams{
		ChatID:    telego.ChatID{ID: body.NewOwnerID},
		Text:      textNew,
		ParseMode: telego.ModeHTML,
	}
	if buttonNew != nil {
		paramsNew.ReplyMarkup = buttonNew
	}
	_, _ = c.container.TelegoBot.SendMessage(context.Background(), paramsNew)

	_, err = c.container.CacheService.DeleteAllUserSessionsBySuffix(ctx, channel.OwnerID)
	if err != nil {
		logger.Error("API", "Erro ao excluir all sessions: %v", err)
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Dono migrado com sucesso!"))
}
