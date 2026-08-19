package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type PostTemplateController struct {
	container *container.AppContainer
}

func NewPostTemplateController(c *container.AppContainer) *PostTemplateController {
	return &PostTemplateController{container: c}
}

type SavePostTemplateRequest struct {
	Name string `json:"name" binding:"required"`
}

func (ctrl *PostTemplateController) ListTemplates(ctx *gin.Context) {
	userID := ctx.GetInt64("userID")
	if userID == 0 {
		ctx.Error(errors.New(http.StatusUnauthorized, "Não autorizado"))
		return
	}

	templates, err := ctrl.container.PostTemplateService.ListTemplates(ctx, userID)
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(templates))
}

func (ctrl *PostTemplateController) SaveCurrentTemplate(ctx *gin.Context) {
	userID := ctx.GetInt64("userID")
	if userID == 0 {
		ctx.Error(errors.New(http.StatusUnauthorized, "Não autorizado"))
		return
	}

	var req SavePostTemplateRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Nome do rascunho é obrigatório"))
		return
	}

	session, err := ctrl.container.CacheService.GetPostBuilderState(ctx, userID)
	if err != nil || session == nil {
		ctx.Error(errors.BadRequest("Sessão do PostBuilder não encontrada no cache"))
		return
	}

	sessionBytes, _ := json.Marshal(session)
	tpl, err := ctrl.container.PostTemplateService.SaveTemplate(ctx, userID, req.Name, string(sessionBytes))
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusCreated, types.NewSuccessResponse(tpl, "Rascunho salvo com sucesso"))
}

func (ctrl *PostTemplateController) DeleteTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetInt64("userID")

	if err := ctrl.container.PostTemplateService.DeleteTemplate(ctx, id, userID); err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Rascunho excluído com sucesso"))
}

func (ctrl *PostTemplateController) LoadTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetInt64("userID")

	tpl, err := ctrl.container.PostTemplateService.GetTemplateByID(ctx, id)
	if err != nil || tpl == nil || tpl.OwnerID != userID {
		ctx.Error(errors.New(http.StatusNotFound, "Rascunho não encontrado"))
		return
	}

	var state cache.PostBuilderState
	if err := json.Unmarshal([]byte(tpl.TemplateData), &state); err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	state.MenuMessageID = 0
	state.PromptMessageID = 0
	state.Step = ""

	if err := ctrl.container.CacheService.SetPostBuilderState(ctx, userID, state); err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(state, "Rascunho carregado com sucesso"))
}
