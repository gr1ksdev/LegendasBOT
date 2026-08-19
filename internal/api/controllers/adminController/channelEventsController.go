package admincontroller

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
)

type ChannelEventsController struct {
	container *container.AppContainer
}

func NewChannelEventsController(app *container.AppContainer) *ChannelEventsController {
	return &ChannelEventsController{container: app}
}

func (c *ChannelEventsController) List(ctx *gin.Context) {
	filters := services.ChannelEventListFilters{
		ChannelID: parseInt64Query(ctx, "channelId"),
		OwnerID:   parseInt64Query(ctx, "ownerId"),
		ActorID:   parseInt64Query(ctx, "actorId"),
		Source:    ctx.Query("source"),
		EventType: ctx.Query("eventType"),
		Status:    ctx.Query("status"),
		SessionID: ctx.Query("sessionId"),
		Query:     ctx.Query("q"),
		Limit:     parseIntQuery(ctx, "limit", 50),
		Offset:    parseIntQuery(ctx, "offset", 0),
	}

	if dateFrom := parseTimeQuery(ctx, "dateFrom"); dateFrom != nil {
		filters.DateFrom = dateFrom
	}
	if dateTo := parseTimeQuery(ctx, "dateTo"); dateTo != nil {
		filters.DateTo = dateTo
	}

	result, err := c.container.ChannelEventService.ListAdmin(ctx, filters)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(result))
}

func parseInt64Query(ctx *gin.Context, key string) int64 {
	value := ctx.Query(key)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseInt(value, 10, 64)
	return parsed
}

func parseIntQuery(ctx *gin.Context, key string, fallback int) int {
	value := ctx.Query(key)
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}

func parseTimeQuery(ctx *gin.Context, key string) *time.Time {
	value := ctx.Query(key)
	if value == "" {
		return nil
	}
	if parsed, err := time.Parse(time.RFC3339, value); err == nil {
		return &parsed
	}
	if parsed, err := time.Parse("2006-01-02", value); err == nil {
		return &parsed
	}
	return nil
}
