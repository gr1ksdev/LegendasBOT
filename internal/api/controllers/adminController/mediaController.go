package admincontroller

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

type MediaController struct {
	container *container.AppContainer
}

func NewMediaController(c *container.AppContainer) *MediaController {
	return &MediaController{container: c}
}

func (c *MediaController) GetMediaPreview(ctx *gin.Context) {
	fileID := ctx.Param("fileId")
	if fileID == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "fileId is required"})
		return
	}

	bot := c.container.TelegoBot
	file, err := bot.GetFile(context.Background(), &telego.GetFileParams{FileID: fileID})
	if err != nil {
		logger.Error("API", "Erro ao obter file do Telegram: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file info from Telegram"})
		return
	}

	telegramURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", config.TelegramBotToken, file.FilePath)

	// Fazer o download da imagem do Telegram e servir os bytes
	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Get(telegramURL)
	if err != nil {
		logger.Error("API", "Erro ao baixar arquivo do Telegram: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to download file from Telegram"})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "Telegram returned non-200 status"})
		return
	}

	const maxMediaPreviewBytes int64 = 20 << 20
	if resp.ContentLength > maxMediaPreviewBytes {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Media preview is too large"})
		return
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, maxMediaPreviewBytes+1))
	if err != nil {
		logger.Error("API", "Erro ao ler arquivo do Telegram: %v", err)
		ctx.JSON(http.StatusBadGateway, gin.H{"error": "Failed to read media from Telegram"})
		return
	}
	if int64(len(data)) > maxMediaPreviewBytes {
		ctx.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "Media preview is too large"})
		return
	}

	// Copiar headers relevantes (especialmente Content-Type)
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	ctx.Data(http.StatusOK, contentType, data)
}
