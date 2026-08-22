package admincontroller

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

type cachedMedia struct {
	contentType string
	data        []byte
	cachedAt    time.Time
}

type MediaController struct {
	container *container.AppContainer
	memCache  sync.Map
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

	etag := fmt.Sprintf("\"%s\"", fileID)
	ctx.Header("Cache-Control", "public, max-age=31536000, immutable")
	ctx.Header("ETag", etag)

	if match := ctx.GetHeader("If-None-Match"); match == etag {
		ctx.Status(http.StatusNotModified)
		return
	}

	// 1. MemCache check
	if val, ok := c.memCache.Load(fileID); ok {
		cached := val.(cachedMedia)
		if time.Since(cached.cachedAt) < 24*time.Hour {
			ctx.Data(http.StatusOK, cached.contentType, cached.data)
			return
		}
		c.memCache.Delete(fileID)
	}

	// 2. Redis check
	redisClient := cache.GetRedisClient()
	redisKey := "media_preview:" + fileID
	if redisClient != nil {
		if raw, err := redisClient.Get(ctx.Request.Context(), redisKey).Bytes(); err == nil && len(raw) > 0 {
			if splitIdx := bytes.IndexByte(raw, '\n'); splitIdx > 0 {
				ct := string(raw[:splitIdx])
				d := raw[splitIdx+1:]
				c.memCache.Store(fileID, cachedMedia{
					contentType: ct,
					data:        d,
					cachedAt:    time.Now(),
				})
				ctx.Data(http.StatusOK, ct, d)
				return
			}
		}
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

	// Guardar em memória e Redis
	c.memCache.Store(fileID, cachedMedia{
		contentType: contentType,
		data:        data,
		cachedAt:    time.Now(),
	})
	if redisClient != nil {
		payload := append([]byte(contentType+"\n"), data...)
		_ = redisClient.Set(ctx.Request.Context(), redisKey, payload, 7*24*time.Hour).Err()
	}

	ctx.Data(http.StatusOK, contentType, data)
}
