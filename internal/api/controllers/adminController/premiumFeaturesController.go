package admincontroller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

// PremiumFeaturesController gerencia os endpoints de features premium para admin.
type PremiumFeaturesController struct {
	container *container.AppContainer
}

// NewPremiumFeaturesController cria um novo controller.
func NewPremiumFeaturesController(container *container.AppContainer) *PremiumFeaturesController {
	return &PremiumFeaturesController{container: container}
}

// ListFeatures retorna todas as features premium cadastradas.
func (ctrl *PremiumFeaturesController) ListFeatures(c *gin.Context) {
	features, err := ctrl.container.PremiumFeatureService.ListFeatures(c.Request.Context())
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(features))
}

// UpdateFeatureRequest representa o request de atualizacao de uma feature.
type UpdateFeatureRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Enabled     *bool  `json:"enabled"`
	Price       *int   `json:"price"`
}

// UpdateFeature atualiza os dados de uma feature premium.
func (ctrl *PremiumFeaturesController) UpdateFeature(c *gin.Context) {
	key := c.Param("key")

	var req UpdateFeatureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.BadRequest("Dados inválidos"))
		return
	}

	feature, err := ctrl.container.PremiumFeatureService.GetFeature(c.Request.Context(), key)
	if err != nil {
		_ = c.Error(err)
		return
	}
	if feature == nil {
		_ = c.Error(errors.ErrNotFound)
		return
	}

	if req.Name != "" {
		feature.Name = req.Name
	}
	if req.Description != "" {
		feature.Description = req.Description
	}
	if req.Enabled != nil {
		feature.Enabled = *req.Enabled
	}
	if req.Price != nil {
		if *req.Price < 0 {
			c.Error(errors.BadRequest("Preço não pode ser negativo"))
			return
		}
		feature.Price = *req.Price
	}

	if err := ctrl.container.PremiumFeatureService.UpdateFeature(c.Request.Context(), feature); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Feature atualizada com sucesso"))
}

// ToggleFeature ativa ou desativa uma feature globalmente.
func (ctrl *PremiumFeaturesController) ToggleFeature(c *gin.Context) {
	key := c.Param("key")

	var req struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(errors.BadRequest("enabled é obrigatório"))
		return
	}

	if err := ctrl.container.PremiumFeatureService.ToggleFeature(c.Request.Context(), key, req.Enabled); err != nil {
		_ = c.Error(err)
		return
	}

	status := "desativada"
	if req.Enabled {
		status = "ativada"
	}

	c.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"key":     key,
		"enabled": req.Enabled,
	}, "Feature "+status+" com sucesso"))
}
