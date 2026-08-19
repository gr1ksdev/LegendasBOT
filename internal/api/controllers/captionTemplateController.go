package controllers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type CaptionTemplateController struct {
	container *container.AppContainer
}

func NewCaptionTemplateController(container *container.AppContainer) *CaptionTemplateController {
	return &CaptionTemplateController{
		container: container,
	}
}

func (ctrl *CaptionTemplateController) Save(ctx *gin.Context) {
	channelID, userID, err := parseChannelAndUser(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := ctx.ShouldBindJSON(&body); err != nil {
		ctx.Error(errors.BadRequest("nome é obrigatório"))
		return
	}

	channel, err := ctrl.container.ChannelService.GetChannelByID(ctx, channelID)
	if err != nil {
		ctx.Error(err)
		return
	}

	var customCaptions []types.CustomCaptionSnapshot
	for _, cc := range channel.CustomCaptions {
		var buttons []types.CaptionButtonSnapshot
		for _, b := range cc.Buttons {
			buttons = append(buttons, types.CaptionButtonSnapshot{
				NameButton: b.NameButton,
				ButtonURL:  b.ButtonURL,
				Style:      b.Style,
			})
		}
		customCaptions = append(customCaptions, types.CustomCaptionSnapshot{
			Code:        cc.Code,
			Caption:     cc.Caption,
			LinkPreview: cc.LinkPreview,
			Buttons:     buttons,
		})
	}

	defaultCaption := ""
	if channel.DefaultCaption != nil {
		defaultCaption = channel.DefaultCaption.Caption
	}
	templateData := types.CaptionTemplateData{
		DefaultCaption: defaultCaption,
		CustomCaptions: customCaptions,
	}

	dataJSON, err := json.Marshal(templateData)
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	tpl, err := ctrl.container.PostTemplateService.SaveTemplate(ctx, userID, body.Name, string(dataJSON))
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusCreated, types.NewSuccessResponse(gin.H{
		"id":           tpl.ID,
		"name":         tpl.Name,
		"templateData": tpl.TemplateData,
		"createdAt":    tpl.CreatedAt,
	}, "Template salvo"))
}

func (ctrl *CaptionTemplateController) List(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	templates, err := ctrl.container.PostTemplateService.ListTemplates(ctx, userID.(int64))
	if err != nil {
		ctx.Error(err)
		return
	}

	type templateItem struct {
		ID           string `json:"id"`
		Name         string `json:"name"`
		TemplateData string `json:"templateData"`
		CreatedAt    string `json:"createdAt"`
		UpdatedAt    string `json:"updatedAt"`
	}

	var items []templateItem
	for _, t := range templates {
		items = append(items, templateItem{
			ID:           t.ID,
			Name:         t.Name,
			TemplateData: t.TemplateData,
			CreatedAt:    t.CreatedAt.Format("2006-01-02T15:04:05Z"),
			UpdatedAt:    t.UpdatedAt.Format("2006-01-02T15:04:05Z"),
		})
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(items, ""))
}

func (ctrl *CaptionTemplateController) Get(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	templateID := ctx.Param("templateId")

	tpl, err := ctrl.container.PostTemplateService.GetTemplateByID(ctx, templateID)
	if err != nil {
		ctx.Error(err)
		return
	}

	if tpl.OwnerID != userID.(int64) {
		ctx.Error(errors.ErrForbidden)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(gin.H{
		"id":           tpl.ID,
		"name":         tpl.Name,
		"templateData": tpl.TemplateData,
		"createdAt":    tpl.CreatedAt,
		"updatedAt":    tpl.UpdatedAt,
	}, ""))
}

func (ctrl *CaptionTemplateController) Apply(ctx *gin.Context) {
	channelID, userID, err := parseChannelAndUser(ctx)
	if err != nil {
		ctx.Error(err)
		return
	}

	templateID := ctx.Param("templateId")

	tpl, err := ctrl.container.PostTemplateService.GetTemplateByID(ctx, templateID)
	if err != nil {
		ctx.Error(err)
		return
	}

	if tpl.OwnerID != userID {
		ctx.Error(errors.ErrForbidden)
		return
	}

	var data types.CaptionTemplateData
	if err := json.Unmarshal([]byte(tpl.TemplateData), &data); err != nil {
		ctx.Error(errors.BadRequest("dados do template inválidos"))
		return
	}

	_, err = ctrl.container.CaptionService.UpdateDefaultCaption(ctx, channelID, types.CaptionDefaultUpdateRequest{
		Caption: data.DefaultCaption,
	})
	if err != nil {
		ctx.Error(err)
		return
	}

	if len(data.CustomCaptions) > 0 {
		if err := ctrl.container.CustomCaptionService.ReplaceCustomCaptionsForChannel(ctx, channelID, data.CustomCaptions); err != nil {
			ctx.Error(err)
			return
		}
	}

	ctrl.container.CacheService.InvalidateChannel(ctx, channelID)
	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Template aplicado com sucesso"))
}

func (ctrl *CaptionTemplateController) Delete(ctx *gin.Context) {
	userID, ok := ctx.Get("userID")
	if !ok {
		ctx.Error(errors.ErrUnauthorized)
		return
	}

	templateID := ctx.Param("templateId")

	if err := ctrl.container.PostTemplateService.DeleteTemplate(ctx, templateID, userID.(int64)); err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Template excluído"))
}

func parseChannelAndUser(ctx *gin.Context) (int64, int64, error) {
	channelIDStr := ctx.Param("channelId")
	channelID, err := strconv.ParseInt(channelIDStr, 10, 64)
	if err != nil {
		return 0, 0, errors.BadRequest("ID do canal inválido")
	}

	userID, ok := ctx.Get("userID")
	if !ok {
		return 0, 0, errors.ErrUnauthorized
	}

	return channelID, userID.(int64), nil
}
