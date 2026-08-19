package controllers

import (
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/auth"
	"github.com/leirbagxis/FreddyBot/internal/api/dto"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type ChannelController struct {
	container *container.AppContainer
}

func NewChannelController(container *container.AppContainer) *ChannelController {
	return &ChannelController{
		container: container,
	}
}

func (c *ChannelController) GetAllChannelsController(ctx *gin.Context) {
	channels, err := c.container.ChannelService.GetAllChannels(ctx)
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

func (c *ChannelController) GetChannelByIDController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("channelId inválido"))
		return
	}

	// Obter dados do contexto (injetados pelo middleware)
	ctxUserID, _ := ctx.Get("userID")
	ctxRole, _ := ctx.Get("role")

	userID := ctxUserID.(int64)
	role := ctxRole.(auth.Role)

	channel, err := c.container.ChannelService.GetChannelByID(ctx, channelId)
	if err != nil {
		ctx.Error(err)
		return
	}

	// --- VERIFICAÇÃO DE PERMISSÃO ---
	// Se não for Admin/Owner, o UserID do token deve ser igual ao OwnerID do canal
	if role != auth.RoleAdmin && role != auth.RoleOwner {
		if channel.OwnerID != userID {
			ctx.Error(errors.ErrForbidden)
			return
		}
	}

	userDTO := dto.ToUserDTO(channel.Owner)
	if len(userDTO.Channels) == 0 && channel.OwnerID != 0 {
		if userChannels, err := c.container.ChannelService.GetUserChannels(ctx, channel.OwnerID); err == nil {
			for _, ch := range userChannels {
				userDTO.Channels = append(userDTO.Channels, dto.ToChannelDTO(&ch))
			}
		}
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"user":    userDTO,
		"channel": dto.ToChannelDTO(channel),
	}))
}

func (c *ChannelController) DisconectChannel(ctx *gin.Context) {
	channelIDStr, exists := ctx.Get("channelID")
	channelID, ok := channelIDStr.(int64)
	if !exists || !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	channel, err := c.container.ChannelService.GetChannelByID(ctx, channelID)
	if err != nil {
		ctx.Error(err)
		return
	}

	err = c.container.ChannelService.DisconnectChannel(ctx, channel.OwnerID, channelID)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.Status(http.StatusNoContent)
}

// GetSeparatorByChannel retorna o separator atual do canal.
// GET /api/channel/:channelId/separator
func (c *ChannelController) GetSeparatorByChannel(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	separator, err := c.container.SeparatorService.GetSeparatorByOwnerChannelID(ctx, channelId)
	if err != nil {
		ctx.Error(err)
		return
	}
	if separator == nil {
		ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"id":                separator.ID,
		"type":              separator.Type,
		"emojiText":         separator.EmojiText,
		"emojiId":           separator.EmojiID,
		"emojiEntitiesJSON": separator.EmojiEntitiesJSON,
		"ownerChannelId":    separator.OwnerChannelID,
	}))
}

type updateSeparatorRequest struct {
	Type              string `json:"type"`
	EmojiText         string `json:"emojiText"`
	EmojiID           string `json:"emojiId"`
	EmojiEntitiesJSON string `json:"emojiEntitiesJSON"`
}

// UpdateSeparator cria ou atualiza o separator do canal.
// PUT /api/channel/:channelId/separator
func (c *ChannelController) UpdateSeparator(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var req updateSeparatorRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos"))
		return
	}

	if req.Type == "" {
		req.Type = "custom_emoji"
	}

	// Validar Type — apenas valores permitidos
	validTypes := map[string]bool{"sticker": true, "custom_emoji": true}
	if !validTypes[req.Type] {
		ctx.Error(errors.BadRequest("Tipo de separador inválido. Use 'sticker' ou 'custom_emoji'"))
		return
	}

	// Validar EmojiEntitiesJSON — deve ser um JSON array válido com campos obrigatorios
	if req.EmojiEntitiesJSON != "" {
		var entities []struct {
			Type    string `json:"type"`
			Offset  int    `json:"offset"`
			Length  int    `json:"length"`
			EmojiID string `json:"emoji_id"`
		}
		if err := json.Unmarshal([]byte(req.EmojiEntitiesJSON), &entities); err != nil {
			ctx.Error(errors.BadRequest("emojiEntitiesJSON inválido: não é um JSON array válido"))
			return
		}
		for i, e := range entities {
			if e.Type != "custom_emoji" || e.EmojiID == "" {
				ctx.Error(errors.BadRequest(fmt.Sprintf("entidade %d inválida: type deve ser 'custom_emoji' e emoji_id é obrigatório", i)))
				return
			}
		}
	}

	separator := &models.Separator{
		Type:              req.Type,
		EmojiText:         req.EmojiText,
		EmojiID:           req.EmojiID,
		EmojiEntitiesJSON: req.EmojiEntitiesJSON,
		OwnerChannelID:    channelId,
	}

	if err := c.container.SeparatorService.SaveSeparator(ctx, separator); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Separador atualizado com sucesso",
	}))
}

// DeleteSeparator remove o separator do canal.
// DELETE /api/channel/:channelId/separator
func (c *ChannelController) DeleteSeparator(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	if err := c.container.SeparatorService.DeleteSeparatorByOwnerChannelId(ctx, channelId); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"message": "Separador removido com sucesso",
	}))
}

func (c *ChannelController) GetSeparator(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	separatorId := ctx.Param("separatorId")
	if separatorId == "" {
		ctx.Error(errors.BadRequest("Separator ID é obrigatório"))
		return
	}

	stickerData, err := c.container.SeparatorService.GetSeparatorByTwoID(ctx, channelId, separatorId)
	if err != nil {
		ctx.Error(err)
		return
	}
	if stickerData == nil {
		ctx.Error(errors.ErrNotFound)
		return
	}

	stickerFile := stickerData.SeparatorURL
	if stickerFile == "" {
		ctx.Error(errors.New(http.StatusInternalServerError, "Sticker sem arquivo"))
		return
	}

	ext := strings.ToLower(filepath.Ext(stickerFile))

	if ext == ".tgs" {
		ctx.Error(errors.New(http.StatusNotImplemented, "Formato TGS ainda não suportado"))
		return
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.TelegramBotToken, stickerFile)
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(telegramURL)
	if err != nil {
		ctx.Error(errors.New(http.StatusInternalServerError, "Erro ao buscar conteúdo do sticker"))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ctx.Error(errors.New(http.StatusInternalServerError, "Erro ao buscar conteúdo do sticker"))
		return
	}

	contentType := mime.TypeByExtension(ext)
	if contentType == "" {
		if ext == ".webm" {
			contentType = "video/webm"
		} else {
			contentType = "application/octet-stream"
		}
	}

	const maxStickerBytes int64 = 10 << 20
	if resp.ContentLength > maxStickerBytes {
		ctx.Error(errors.New(http.StatusRequestEntityTooLarge, "Sticker muito grande"))
		return
	}
	content, err := io.ReadAll(io.LimitReader(resp.Body, maxStickerBytes+1))
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}
	if int64(len(content)) > maxStickerBytes {
		ctx.Error(errors.New(http.StatusRequestEntityTooLarge, "Sticker muito grande"))
		return
	}

	// Adiciona headers explícitos
	ctx.Header("Content-Type", contentType)
	ctx.Header("Content-Disposition", "inline; filename=sticker"+ext)
	ctx.Data(http.StatusOK, contentType, content)
}
