package controllers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/leirbagxis/FreddyBot/internal/api/types"
	"github.com/leirbagxis/FreddyBot/internal/container"
	"github.com/leirbagxis/FreddyBot/internal/core/services"
	"github.com/leirbagxis/FreddyBot/pkg/errors"
)

type SchedulerController struct {
	container *container.AppContainer
}

func NewSchedulerController(c *container.AppContainer) *SchedulerController {
	return &SchedulerController{container: c}
}

func (ctrl *SchedulerController) GetMySchedules(ctx *gin.Context) {
	userID := ctx.GetInt64("userID")
	if userID == 0 {
		ctx.Error(errors.New(http.StatusUnauthorized, "Não autorizado"))
		return
	}

	schedules, err := ctrl.container.SchedulerService.GetUserSchedules(ctx, userID)
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(schedules))
}

func (ctrl *SchedulerController) GetScheduleByID(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetInt64("userID")

	schedule, err := ctrl.container.SchedulerService.GetScheduleByID(ctx, id)
	if err != nil {
		ctx.Error(errors.New(http.StatusNotFound, "Agendamento não encontrado"))
		return
	}

	if schedule.OwnerID != userID {
		ctx.Error(errors.New(http.StatusForbidden, "Não autorizado"))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(schedule))
}

func (ctrl *SchedulerController) UpdateStatus(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetInt64("userID")

	var req types.UpdateScheduleStatusRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	var err error
	switch req.Status {
	case "paused":
		err = ctrl.container.SchedulerService.PauseScheduledPost(ctx, id, userID)
	case "pending":
		err = ctrl.container.SchedulerService.ResumeScheduledPost(ctx, id, userID)
	case "cancelled":
		err = ctrl.container.SchedulerService.CancelScheduledPost(ctx, id, userID)
	}

	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Status atualizado"))
}

func (ctrl *SchedulerController) DeleteSchedule(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetInt64("userID")

	err := ctrl.container.SchedulerService.DeleteScheduledPost(ctx, id, userID)
	if err != nil {
		ctx.Error(errors.Internal(err))
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Agendamento removido"))
}

func (ctrl *SchedulerController) EditSchedule(ctx *gin.Context) {
	id := ctx.Param("id")
	userID := ctx.GetInt64("userID")

	var req types.EditScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	var nextRunAt *time.Time
	if req.NextRunAt != "" {
		t, err := time.Parse(time.RFC3339, req.NextRunAt)
		if err != nil {
			ctx.Error(errors.BadRequest("Formato de data inválido (use ISO 8601)"))
			return
		}
		nextRunAt = &t
	}

	// Update next run time if provided
	if req.NextRunAt != "" || req.ScheduleTime != "" {
		err := ctrl.container.SchedulerService.UpdateScheduleTime(ctx, id, userID, nextRunAt, req.ScheduleTime)
		if err != nil {
			ctx.Error(err)
			return
		}
	}

	// Update interval configuration if provided
	if req.IntervalMin != nil {
		windowStart := ""
		if req.WindowStart != nil {
			windowStart = *req.WindowStart
		}
		windowEnd := ""
		if req.WindowEnd != nil {
			windowEnd = *req.WindowEnd
		}
		err := ctrl.container.SchedulerService.UpdateScheduleInterval(ctx, id, userID, *req.IntervalMin, windowStart, windowEnd)
		if err != nil {
			ctx.Error(err)
			return
		}
	}

	// Update pin message flag if provided
	if req.PinMessage != nil {
		err := ctrl.container.SchedulerService.UpdateSchedulePinMessage(ctx, id, userID, *req.PinMessage)
		if err != nil {
			ctx.Error(errors.Internal(err))
			return
		}
	}

	// Update auto delete if provided
	if req.AutoDeleteMin != nil {
		err := ctrl.container.SchedulerService.UpdateScheduleAutoDelete(ctx, id, userID, *req.AutoDeleteMin)
		if err != nil {
			ctx.Error(errors.Internal(err))
			return
		}
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse[any](nil, "Agendamento atualizado"))
}

func (ctrl *SchedulerController) CreateSchedule(ctx *gin.Context) {
	userID := ctx.GetInt64("userID")
	if userID == 0 {
		ctx.Error(errors.New(http.StatusUnauthorized, "Não autorizado"))
		return
	}

	var req types.CreateScheduleRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.Error(errors.BadRequest("Dados inválidos: " + err.Error()))
		return
	}

	session, err := ctrl.container.CacheService.GetPostBuilderState(ctx, userID)
	if err != nil || session == nil {
		ctx.Error(errors.BadRequest("Sessão do PostBuilder expirada ou inválida"))
		return
	}

	postDataBytes, _ := json.Marshal(session)
	postData := string(postDataBytes)

	opts := services.ScheduleOptions{
		ScheduleType:  req.ScheduleType,
		ScheduleTime:  req.ScheduleTime,
		ScheduleDays:  req.ScheduleDays,
		IntervalMin:   req.IntervalMin,
		WindowStart:   req.WindowStart,
		WindowEnd:     req.WindowEnd,
		LoopQueue:     req.LoopQueue,
		PinMessage:    req.PinMessage,
		AutoDeleteMin: req.AutoDeleteMin,
	}

	if opts.AutoDeleteMin <= 0 && session.AutoDeleteMin > 0 {
		opts.AutoDeleteMin = session.AutoDeleteMin
	}

	if req.ScheduledAt != "" {
		t, err := time.Parse(time.RFC3339, req.ScheduledAt)
		if err != nil {
			ctx.Error(errors.BadRequest("Formato de data inválido (use ISO 8601)"))
			return
		}
		opts.ScheduledAt = &t
	}

	if req.RepeatUntil != "" {
		t, err := time.Parse(time.RFC3339, req.RepeatUntil)
		if err != nil {
			ctx.Error(errors.BadRequest("Formato de RepeatUntil inválido"))
			return
		}
		opts.RepeatUntil = &t
	}

	schedule, err := ctrl.container.SchedulerService.CreateScheduledPost(
		ctx, userID, req.ChannelID, postData, opts,
	)
	if err != nil {
		ctx.Error(err)
		return
	}

	ctx.JSON(http.StatusOK, types.NewSuccessResponse(schedule, "Agendamento criado com sucesso"))
}
