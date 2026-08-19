package controllers

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"path/filepath"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/mymmrac/telego"
)

const maxChannelPhotoBytes = 10 << 20

type CaptionController struct {
	container *container.AppContainer
}

func NewCaptionController(container *container.AppContainer) *CaptionController {
	return &CaptionController{
		container: container,
	}
}

func (c *CaptionController) UpdateDefaultCaptionController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var captionData types.CaptionDefaultUpdateRequest
	if err := ctx.ShouldBindJSON(&captionData); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	rowsAffected, err := c.container.CaptionService.UpdateDefaultCaption(ctx, channelId, captionData)
	if err != nil {
		ctx.Error(err)
		return
	}

	// ── Baixar emojis personalizados em background ──
	userID, _ := ctx.Get("userID")
	if uid, ok := userID.(int64); ok && c.container.EmojiService.TextContainsEmoji(captionData.Caption) {
		go func(uID int64, caption string) {
			if err := c.container.EmojiService.EnsureEmojisForText(context.Background(), uID, caption); err != nil {
				// Apenas loga, não quebra o fluxo
			}
		}(uid, captionData.Caption)
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Legenda padrão atualizada com sucesso"))
}

func (c *CaptionController) UpdateNewPackCaptionController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var captionData types.NewPackCaptionUpdateRequest
	if err := ctx.ShouldBindJSON(&captionData); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	rowsAffected, err := c.container.CaptionService.UpdateNewPackCaption(ctx, channelId, captionData)
	if err != nil {
		ctx.Error(err)
		return
	}

	// ── Baixar emojis personalizados em background ──
	captionText := captionData.Text()
	userID, _ := ctx.Get("userID")
	if uid, ok := userID.(int64); ok && c.container.EmojiService.TextContainsEmoji(captionText) {
		go func(uID int64, caption string) {
			_ = c.container.EmojiService.EnsureEmojisForText(context.Background(), uID, caption)
		}(uid, captionText)
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Legenda de novos packs atualizada com sucesso"))
}

func (c *CaptionController) UpdateReactionsController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var reactionsData types.ReactionsUpdateRequest
	if err := ctx.ShouldBindJSON(&reactionsData); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	rowsAffected, err := c.container.CaptionService.UpdateReactions(ctx, channelId, reactionsData)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Reações atualizadas com sucesso"))
}

func (c *CaptionController) UpdateReactionPositionController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var posData types.ReactionPositionUpdateRequest
	if err := ctx.ShouldBindJSON(&posData); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	rowsAffected, err := c.container.CaptionService.UpdateReactionPosition(ctx, channelId, posData)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{"rows_affected": rowsAffected}, "Posição das reações atualizada com sucesso"))
}

func (c *CaptionController) UpdateNativeReactionsController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var req types.NativeReactionsUpdateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	if err := c.container.CaptionService.UpdateNativeReactions(ctx, channelId, req.NativeReactions); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Reações nativas atualizadas"))
}

func (c *CaptionController) UpdateNativeReactionModeController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var req types.NativeReactionModeRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	if err := c.container.CaptionService.UpdateNativeReactionMode(ctx, channelId, req.Mode); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Modo de reações nativas atualizado"))
}

func (c *CaptionController) UpdateNativeReactionsEnabledController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	var req types.NativeReactionsEnabledRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	if err := c.container.CaptionService.UpdateNativeReactionsEnabled(ctx, channelId, req.Enabled); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Reações nativas "+map[bool]string{true: "ativadas", false: "desativadas"}[req.Enabled]))
}

func downloadTelegramFile(ctx context.Context, client *http.Client, downloadURL string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("telegram returned HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxBytes+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("telegram file exceeds %d bytes", maxBytes)
	}
	return data, nil
}

// GetChannelPhotoController busca a foto do canal no servidor sem expor a URL de download do Telegram.
func (c *CaptionController) GetChannelPhotoController(ctx *gin.Context) {
	channelIdStr := ctx.Param("channelId")
	channelId, err := strconv.ParseInt(channelIdStr, 10, 64)
	if err != nil {
		ctx.Error(errors.BadRequest("ID do canal inválido"))
		return
	}

	bot := c.container.TelegoBot

	chat, err := bot.GetChat(context.Background(), &telego.GetChatParams{
		ChatID: telego.ChatID{ID: channelId},
	})
	if err != nil || chat == nil {
		ctx.Error(errors.New(http.StatusNotFound, "Canal não encontrado ou sem foto"))
		return
	}

	if chat.Photo == nil || chat.Photo.BigFileID == "" {
		ctx.Error(errors.New(http.StatusNotFound, "Este canal não possui foto"))
		return
	}

	file, err := bot.GetFile(context.Background(), &telego.GetFileParams{
		FileID: chat.Photo.BigFileID,
	})
	if err != nil || file == nil || file.FilePath == "" {
		ctx.Error(errors.New(http.StatusNotFound, "Não foi possível obter a foto"))
		return
	}

	data, err := downloadTelegramFile(ctx.Request.Context(), &http.Client{Timeout: 15 * time.Second}, bot.FileDownloadURL(file.FilePath), maxChannelPhotoBytes)
	if err != nil {
		ctx.Error(errors.New(http.StatusBadGateway, "Não foi possível baixar a foto"))
		return
	}

	contentType := mime.TypeByExtension(filepath.Ext(file.FilePath))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	ctx.Header("Cache-Control", "private, max-age=3600")
	ctx.Data(http.StatusOK, contentType, data)
}
