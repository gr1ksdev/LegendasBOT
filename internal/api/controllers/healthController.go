package controllers

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
)

type HealthController struct {
	container *container.AppContainer
}

func NewHealthController(container *container.AppContainer) *HealthController {
	return &HealthController{container: container}
}

// Healthz indica liveness do processo (sem dependências externas).
func (h *HealthController) Healthz(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// Readyz indica prontidão para receber tráfego (verifica banco de dados e redis).
func (h *HealthController) Readyz(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	dbStatus := "up"
	redisStatus := "up"
	isReady := true

	if h.container != nil && h.container.DB != nil {
		sqlDB, err := h.container.DB.DB()
		if err != nil || sqlDB.PingContext(ctx) != nil {
			dbStatus = "down"
			isReady = false
		}
	} else {
		dbStatus = "down"
		isReady = false
	}

	if err := cache.HealthCheck(ctx); err != nil {
		redisStatus = "down"
		isReady = false
	}

	response := gin.H{
		"status":   "ok",
		"database": dbStatus,
		"redis":    redisStatus,
	}

	if !isReady {
		response["status"] = "unavailable"
		c.JSON(http.StatusServiceUnavailable, response)
		return
	}

	c.JSON(http.StatusOK, response)
}
