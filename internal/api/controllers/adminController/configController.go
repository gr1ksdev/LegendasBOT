package admincontroller

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
)

type ConfigController struct {
	container *container.AppContainer
}

func NewConfigController(container *container.AppContainer) *ConfigController {
	return &ConfigController{container: container}
}

func (ctrl *ConfigController) GetConfig(ctx *gin.Context) {
	config, err := ctrl.container.ServerService.GetConfig(ctx)
	if err != nil {
		logger.Error("API", "Erro ao buscar configurações no ServerService: %v", err)
		ctx.Error(errors.New(http.StatusInternalServerError, "Erro ao buscar configurações"))
		return
	}

	logger.Bot("Configuração recuperada com sucesso: %+v", config)
	ctx.JSON(http.StatusOK, types.NewSuccessResponse(config))
}

func (ctrl *ConfigController) UpdateConfig(ctx *gin.Context) {
	var body struct {
		Maintenance             bool   `json:"maintence"`
		ForceJoin               bool   `json:"forceJoin"`
		GlobalDefaultCaption    string `json:"globalDefaultCaption"`
		GlobalNewPackCaption    string `json:"globalNewPackCaption"`
		FixedPostBuilderEnabled bool   `json:"fixedPostBuilderEnabled"`
		FixedPostBuilderKey     string `json:"fixedPostBuilderKey"`
		FixedPostBuilderPayload string `json:"fixedPostBuilderPayload"`
	}

	if err := ctx.ShouldBindJSON(&body); err != nil {
		logger.Error("API", "Erro ao fazer bind do JSON no UpdateConfig: %v", err)
		ctx.Error(errors.BadRequest("Dados inválidos"))
		return
	}

	logger.Bot("Recebendo atualização de configuração: %+v", body)

	fixedKey := strings.TrimSpace(body.FixedPostBuilderKey)
	if fixedKey == "" {
		fixedKey = "legendasbot"
	}
	if strings.ContainsAny(fixedKey, " \t\n\r/") {
		ctx.Error(errors.BadRequest("Key fixa do PostBuilder inválida"))
		return
	}

	fixedPayload := strings.TrimSpace(body.FixedPostBuilderPayload)
	var fixedState cache.PostBuilderState
	if fixedPayload != "" {
		if err := json.Unmarshal([]byte(fixedPayload), &fixedState); err != nil {
			ctx.Error(errors.BadRequest("Payload do PostBuilder fixo inválida"))
			return
		}
	}
	if body.FixedPostBuilderEnabled && fixedPayload == "" {
		ctx.Error(errors.BadRequest("Payload do PostBuilder fixo é obrigatória quando ativo"))
		return
	}

	previousConfig, _ := ctrl.container.ServerService.GetConfig(ctx)

	config, err := ctrl.container.ServerService.UpdateConfig(ctx, body.Maintenance, body.ForceJoin, body.GlobalDefaultCaption, body.GlobalNewPackCaption, body.FixedPostBuilderEnabled, fixedKey, fixedPayload)
	if err != nil {
		logger.Error("API", "Erro ao atualizar configuração no ServerService: %v", err)
		ctx.Error(errors.New(http.StatusInternalServerError, "Erro ao atualizar configurações"))
		return
	}

	if previousConfig != nil && previousConfig.FixedPostBuilderKey != "" && previousConfig.FixedPostBuilderKey != config.FixedPostBuilderKey {
		_ = ctrl.container.CacheService.DeletePostBuilderSession(ctx, previousConfig.FixedPostBuilderKey)
	}

	if config.FixedPostBuilderEnabled {
		if err := ctrl.container.CacheService.SetPostBuilderSession(ctx, config.FixedPostBuilderKey, fixedState, 0); err != nil {
			logger.Error("API", "Erro ao sincronizar PostBuilder fixo no Redis: %v", err)
			ctx.Error(errors.New(http.StatusInternalServerError, "Configuração salva, mas falhou ao sincronizar PostBuilder fixo"))
			return
		}
	} else {
		if err := ctrl.container.CacheService.DeletePostBuilderSession(ctx, config.FixedPostBuilderKey); err != nil {
			logger.Error("API", "Erro ao remover PostBuilder fixo do Redis: %v", err)
		}
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(config, "Configurações atualizadas com sucesso"))
}
