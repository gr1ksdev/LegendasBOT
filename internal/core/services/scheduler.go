package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/internal/utils"
	apperrors "github.com/leirbagxis/FreddyBot/pkg/errors"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

type ScheduleOptions struct {
	ScheduleType  string
	ScheduleTime  string
	ScheduledAt   *time.Time
	ScheduleDays  []int
	RepeatUntil   *time.Time
	QueueGroupID  string
	QueuePosition int
	LoopQueue     bool
	PinMessage    bool
	AutoDeleteMin int
	IntervalMin   int
	WindowStart   string
	WindowEnd     string
}

type SchedulerService struct {
	repo              *repositories.ScheduledPostRepository
	channelRepo       *repositories.ChannelRepository
	cacheService      *cache.Service
	bot               *telego.Bot
	autoDeleteService *AutoDeleteService
}

func NewSchedulerService(
	repo *repositories.ScheduledPostRepository,
	channelRepo *repositories.ChannelRepository,
	cacheService *cache.Service,
	bot *telego.Bot,
) *SchedulerService {
	return &SchedulerService{
		repo:         repo,
		channelRepo:  channelRepo,
		cacheService: cacheService,
		bot:          bot,
	}
}

func (s *SchedulerService) SetAutoDeleteService(svc *AutoDeleteService) {
	s.autoDeleteService = svc
}

func (s *SchedulerService) Start(ctx context.Context) {
	logger.Info("SCHEDULER", "Scheduler iniciado")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.processDuePosts()

	for {
		select {
		case <-ctx.Done():
			logger.Info("SCHEDULER", "Scheduler encerrado")
			return
		case <-ticker.C:
			s.processDuePosts()
		}
	}
}

func (s *SchedulerService) processDuePosts() {
	ctx := context.Background()
	now := time.Now()
	if err := s.repo.RecoverStaleClaims(ctx, now.Add(-5*time.Minute)); err != nil {
		logger.Error("SCHEDULER", "Erro ao recuperar posts interrompidos: %v", err)
	}
	posts, err := s.repo.ClaimDuePosts(ctx, now)
	if err != nil {
		logger.Error("SCHEDULER", "Erro ao reivindicar posts pendentes: %v", err)
		return
	}

	maxPerCycle := 20
	if len(posts) > maxPerCycle {
		posts = posts[:maxPerCycle]
	}

	var wg sync.WaitGroup
	for _, post := range posts {
		wg.Add(1)
		p := post
		go func() {
			defer wg.Done()
			s.sendScheduledPost(ctx, &p)
		}()
	}
	wg.Wait()
}

func (s *SchedulerService) sendScheduledPost(ctx context.Context, post *models.ScheduledPost) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("SCHEDULER", "Panic recuperado em sendScheduledPost (post %s): %v", post.ID, r)
		}
	}()
	logger.Info("SCHEDULER", "Enviando post agendado %s no canal %d", post.ID, post.ChannelID)

	var state cache.PostBuilderState
	if err := json.Unmarshal([]byte(post.PostData), &state); err != nil {
		logger.Error("SCHEDULER", "Erro ao desserializar PostData: %v", err)
		s.repo.UpdateError(ctx, post.ID, err.Error(), post.RetryCount+1)
		return
	}

	msgID, err := s.buildAndSend(ctx, post.ChannelID, &state)
	if err != nil {
		logger.Error("SCHEDULER", "Erro ao enviar post %s: %v", post.ID, err)
		s.repo.UpdateError(ctx, post.ID, err.Error(), post.RetryCount+1)
		return
	}

	// Fixar mensagem se configurado
	if post.PinMessage && msgID > 0 {
		_ = s.bot.PinChatMessage(ctx, &telego.PinChatMessageParams{
			ChatID:    telego.ChatID{ID: post.ChannelID},
			MessageID: msgID,
		})
	}

	// Auto-destruição se configurada
	autoDelMin := post.AutoDeleteMin
	if autoDelMin <= 0 {
		autoDelMin = state.AutoDeleteMin
	}
	if autoDelMin > 0 && msgID > 0 && s.autoDeleteService != nil {
		_ = s.autoDeleteService.ScheduleAutoDelete(ctx, post.ChannelID, msgID, autoDelMin)
	}

	sentAt := time.Now()
	s.repo.MarkSent(ctx, post.ID, sentAt)

	switch post.ScheduleType {
	case "once":
		logger.Info("SCHEDULER", "Post one-shot %s enviado com sucesso", post.ID)
	case "daily":
		nextRun := s.calculateNextDaily(post)
		if nextRun != nil && (post.RepeatUntil == nil || nextRun.Before(*post.RepeatUntil)) {
			s.repo.UpdateStatus(ctx, post.ID, "pending")
			s.repo.UpdateNextRunAt(ctx, post.ID, *nextRun)
			logger.Info("SCHEDULER", "Post diário %s: próximo envio %s", post.ID, nextRun.Format("02/01 15:04"))
		} else {
			logger.Info("SCHEDULER", "Post diário %s finalizado (repeat_until atingido)", post.ID)
		}
	case "weekly":
		nextRun := s.calculateNextWeekly(post)
		if nextRun != nil && (post.RepeatUntil == nil || nextRun.Before(*post.RepeatUntil)) {
			s.repo.UpdateStatus(ctx, post.ID, "pending")
			s.repo.UpdateNextRunAt(ctx, post.ID, *nextRun)
			logger.Info("SCHEDULER", "Post semanal %s: próximo envio %s", post.ID, nextRun.Format("02/01 15:04"))
		} else {
			logger.Info("SCHEDULER", "Post semanal %s finalizado", post.ID)
		}
	case "interval":
		nextRun := s.calculateNextInterval(post, sentAt)
		if nextRun != nil && (post.RepeatUntil == nil || nextRun.Before(*post.RepeatUntil)) {
			s.repo.UpdateStatus(ctx, post.ID, "pending")
			s.repo.UpdateNextRunAt(ctx, post.ID, *nextRun)
			logger.Info("SCHEDULER", "Post intervalo %s: próximo envio %s (cada %dmin)", post.ID, nextRun.Format("02/01 15:04"), post.IntervalMin)
		} else {
			logger.Info("SCHEDULER", "Post intervalo %s finalizado (repeat_until atingido)", post.ID)
		}
	case "queue":
		s.advanceQueue(ctx, post)
	}
}

func (s *SchedulerService) buildAndSend(ctx context.Context, chatID int64, state *cache.PostBuilderState) (int, error) {
	var sb strings.Builder
	if state.Title != "" {
		sb.WriteString(state.Title + "\n\n")
	}
	if state.Body != "" {
		sb.WriteString(state.Body + "\n\n")
	}
	if state.Footer != "" {
		sb.WriteString(state.Footer)
	}
	caption := sb.String()

	if strings.TrimSpace(caption) == "" && state.MediaType == "" {
		return 0, fmt.Errorf("post vazio (sem mídia e sem texto)")
	}

	var kb telego.ReplyMarkup
	if len(state.Buttons) > 0 || state.Reactions != "" {
		ikb := &telego.InlineKeyboardMarkup{}
		for _, btn := range state.Buttons {
			ikb.InlineKeyboard = append(ikb.InlineKeyboard, []telego.InlineKeyboardButton{
				{Text: btn.Text, URL: btn.URL},
			})
		}
		if state.Reactions != "" {
			reactions := strings.Split(state.Reactions, ",")
			var reactionRow []telego.InlineKeyboardButton
			for _, r := range reactions {
				val := strings.TrimSpace(r)
				if val != "" {
					reactionRow = append(reactionRow, telego.InlineKeyboardButton{
						CallbackData: "vote:" + val,
						Text:         val,
					})
				}
			}
			if len(reactionRow) > 0 {
				ikb.InlineKeyboard = append(ikb.InlineKeyboard, reactionRow)
			}
		}
		kb = ikb
	}

	switch state.MediaType {
	case "photo":
		params := &telego.SendPhotoParams{
			ChatID:    telego.ChatID{ID: chatID},
			Photo:     telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := s.bot.SendPhoto(ctx, params)
		if err != nil {
			return 0, err
		}
		return msg.MessageID, nil
	case "video":
		params := &telego.SendVideoParams{
			ChatID:    telego.ChatID{ID: chatID},
			Video:     telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := s.bot.SendVideo(ctx, params)
		if err != nil {
			return 0, err
		}
		return msg.MessageID, nil
	case "animation":
		params := &telego.SendAnimationParams{
			ChatID:    telego.ChatID{ID: chatID},
			Animation: telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := s.bot.SendAnimation(ctx, params)
		if err != nil {
			return 0, err
		}
		return msg.MessageID, nil
	case "audio":
		params := &telego.SendAudioParams{
			ChatID:    telego.ChatID{ID: chatID},
			Audio:     telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := s.bot.SendAudio(ctx, params)
		if err != nil {
			return 0, err
		}
		return msg.MessageID, nil
	case "document":
		params := &telego.SendDocumentParams{
			ChatID:    telego.ChatID{ID: chatID},
			Document:  telego.InputFile{FileID: state.MediaFileID},
			Caption:   caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := s.bot.SendDocument(ctx, params)
		if err != nil {
			return 0, err
		}
		return msg.MessageID, nil
	default:
		params := &telego.SendMessageParams{
			ChatID:    telego.ChatID{ID: chatID},
			Text:      caption,
			ParseMode: telego.ModeHTML,
		}
		if kb != nil {
			params.ReplyMarkup = kb
		}
		msg, err := s.bot.SendMessage(ctx, params)
		if err != nil {
			return 0, err
		}
		return msg.MessageID, nil
	}
}

func (s *SchedulerService) calculateNextDaily(post *models.ScheduledPost) *time.Time {
	brazilTZ := utils.BrazilTZ()
	now := time.Now().In(brazilTZ)
	hour, min := parseHHMM(post.ScheduleTime)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, hour, min, 0, 0, brazilTZ).UTC()
	return &next
}

func (s *SchedulerService) calculateNextWeekly(post *models.ScheduledPost) *time.Time {
	brazilTZ := utils.BrazilTZ()
	now := time.Now().In(brazilTZ)
	hour, min := parseHHMM(post.ScheduleTime)

	var days []int
	if post.ScheduleDays != "" {
		json.Unmarshal([]byte(post.ScheduleDays), &days)
	}
	if len(days) == 0 {
		days = []int{0, 1, 2, 3, 4, 5, 6}
	}

	for i := 1; i <= 7; i++ {
		nextDay := now.AddDate(0, 0, i)
		for _, d := range days {
			if int(nextDay.Weekday()) == d {
				result := time.Date(nextDay.Year(), nextDay.Month(), nextDay.Day(), hour, min, 0, 0, brazilTZ).UTC()
				return &result
			}
		}
	}
	return nil
}

func (s *SchedulerService) calculateNextInterval(post *models.ScheduledPost, sentAt time.Time) *time.Time {
	intervalMin := post.IntervalMin
	if intervalMin < 5 {
		intervalMin = 5
	}
	next := sentAt.Add(time.Duration(intervalMin) * time.Minute)

	if post.WindowStart != "" && post.WindowEnd != "" {
		brazilTZ := utils.BrazilTZ()
		nextLocal := next.In(brazilTZ)

		startH, startM := parseHHMM(post.WindowStart)
		endH, endM := parseHHMM(post.WindowEnd)

		windowStart := time.Date(nextLocal.Year(), nextLocal.Month(), nextLocal.Day(), startH, startM, 0, 0, brazilTZ)
		windowEnd := time.Date(nextLocal.Year(), nextLocal.Month(), nextLocal.Day(), endH, endM, 0, 0, brazilTZ)

		if nextLocal.Before(windowStart) {
			next = windowStart
		} else if nextLocal.After(windowEnd) {
			next = windowStart.AddDate(0, 0, 1)
		}
	}

	result := next.UTC()
	return &result
}

func (s *SchedulerService) advanceQueue(ctx context.Context, sentPost *models.ScheduledPost) {
	if sentPost.QueueGroupID == "" {
		return
	}

	posts, err := s.repo.GetQueueGroup(ctx, sentPost.QueueGroupID)
	if err != nil {
		logger.Error("SCHEDULER", "Erro ao buscar fila %s: %v", sentPost.QueueGroupID, err)
		return
	}

	var nextPost *models.ScheduledPost
	for i := range posts {
		if posts[i].QueuePosition > sentPost.QueuePosition && posts[i].Status == "pending" {
			nextPost = &posts[i]
			break
		}
	}

	if nextPost != nil {
		brazilTZ := utils.BrazilTZ()
		now := time.Now().In(brazilTZ)
		hour, min := parseHHMM(sentPost.ScheduleTime)
		nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, hour, min, 0, 0, brazilTZ).UTC()
		s.repo.UpdateNextRunAt(ctx, nextPost.ID, nextRun)
		logger.Info("SCHEDULER", "Fila %s: próximo post %s agendado para %s", sentPost.QueueGroupID, nextPost.ID, nextRun.Format("02/01 15:04"))
	} else if sentPost.LoopQueue {
		s.resetQueue(ctx, sentPost.QueueGroupID, posts, sentPost.ScheduleTime)
	} else {
		logger.Info("SCHEDULER", "Fila %s concluída", sentPost.QueueGroupID)
	}
}

func (s *SchedulerService) resetQueue(ctx context.Context, queueGroupID string, posts []models.ScheduledPost, scheduleTime string) {
	brazilTZ := utils.BrazilTZ()
	now := time.Now().In(brazilTZ)
	hour, min := parseHHMM(scheduleTime)
	nextRun := time.Date(now.Year(), now.Month(), now.Day()+1, hour, min, 0, 0, brazilTZ).UTC()

	for i := range posts {
		if posts[i].Status == "sent" {
			s.repo.UpdateStatus(ctx, posts[i].ID, "pending")
		}
	}

	if len(posts) > 0 {
		s.repo.UpdateNextRunAt(ctx, posts[0].ID, nextRun)
	}
	logger.Info("SCHEDULER", "Fila %s reiniciada, próximo: %s", queueGroupID, nextRun.Format("02/01 15:04"))
}

func parseHHMM(s string) (int, int) {
	hour, min, err := validateScheduleTime(s)
	if err != nil {
		return 12, 0
	}
	return hour, min
}

func validateScheduleTime(s string) (int, int, error) {
	parts := strings.Split(strings.TrimSpace(s), ":")
	if len(parts) != 2 || len(parts[0]) != 2 || len(parts[1]) != 2 {
		return 0, 0, fmt.Errorf("horário inválido, use HH:MM")
	}
	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return 0, 0, fmt.Errorf("hora inválida, use HH:MM")
	}
	min, err := strconv.Atoi(parts[1])
	if err != nil || min < 0 || min > 59 {
		return 0, 0, fmt.Errorf("minuto inválido, use HH:MM")
	}
	return hour, min, nil
}

// CRUD methods

func (s *SchedulerService) CreateScheduledPost(ctx context.Context, ownerID, channelID int64, postData string, opts ScheduleOptions) (*models.ScheduledPost, error) {
	channel, err := s.channelRepo.GetChannelByIDLight(ctx, channelID)
	if err != nil || channel.OwnerID != ownerID {
		return nil, apperrors.ErrForbidden
	}
	if opts.ScheduleType == "daily" || opts.ScheduleType == "weekly" {
		h, m, err := validateScheduleTime(opts.ScheduleTime)
		if err != nil {
			return nil, apperrors.BadRequest(err.Error())
		}
		opts.ScheduleTime = fmt.Sprintf("%02d:%02d", h, m)
	}
	if opts.ScheduleType == "interval" {
		if opts.IntervalMin < 5 {
			return nil, apperrors.BadRequest("intervalo mínimo é 5 minutos")
		}
		if opts.IntervalMin > 1440 {
			return nil, apperrors.BadRequest("para intervalos de 24h ou mais, use o tipo 'daily'")
		}
		if (opts.WindowStart != "" && opts.WindowEnd == "") || (opts.WindowStart == "" && opts.WindowEnd != "") {
			return nil, apperrors.BadRequest("informe início e fim da janela de horário, ou deixe ambos vazios")
		}
		if opts.WindowStart != "" {
			sh, sm, err := validateScheduleTime(opts.WindowStart)
			if err != nil {
				return nil, apperrors.BadRequest("horário de início da janela inválido")
			}
			eh, em, err := validateScheduleTime(opts.WindowEnd)
			if err != nil {
				return nil, apperrors.BadRequest("horário de fim da janela inválido")
			}
			opts.WindowStart = fmt.Sprintf("%02d:%02d", sh, sm)
			opts.WindowEnd = fmt.Sprintf("%02d:%02d", eh, em)
		}
	}

	limit, _ := s.repo.CountByOwner(ctx, ownerID)
	if limit >= 50 {
		return nil, fmt.Errorf("limite de 50 agendamentos ativos atingido")
	}

	id := fmt.Sprintf("sch_%d_%d", time.Now().UnixNano(), ownerID)

	nextRunAt := time.Now().UTC()
	if opts.ScheduleType == "once" && opts.ScheduledAt != nil {
		nextRunAt = *opts.ScheduledAt
	} else if opts.ScheduleType == "daily" || opts.ScheduleType == "weekly" {
		brazilTZ := utils.BrazilTZ()
		now := time.Now().In(brazilTZ)
		hour, min := parseHHMM(opts.ScheduleTime)
		nextRun := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, brazilTZ).UTC()
		if nextRun.Before(time.Now().UTC()) {
			nextRun = time.Date(now.Year(), now.Month(), now.Day()+1, hour, min, 0, 0, brazilTZ).UTC()
		}
		nextRunAt = nextRun
	} else if opts.ScheduleType == "interval" {
		if opts.WindowStart != "" {
			brazilTZ := utils.BrazilTZ()
			now := time.Now().In(brazilTZ)
			startH, startM := parseHHMM(opts.WindowStart)
			windowStart := time.Date(now.Year(), now.Month(), now.Day(), startH, startM, 0, 0, brazilTZ)
			if now.Before(windowStart) {
				nextRunAt = windowStart.UTC()
			} else {
				nextRunAt = time.Now().Add(time.Duration(opts.IntervalMin) * time.Minute).UTC()
			}
		} else {
			nextRunAt = time.Now().Add(time.Duration(opts.IntervalMin) * time.Minute).UTC()
		}
	}

	var scheduleDaysStr string
	if len(opts.ScheduleDays) > 0 {
		b, _ := json.Marshal(opts.ScheduleDays)
		scheduleDaysStr = string(b)
	}

	post := &models.ScheduledPost{
		ID:            id,
		OwnerID:       ownerID,
		ChannelID:     channelID,
		ChannelTitle:  channel.Title,
		PostData:      postData,
		ScheduleType:  opts.ScheduleType,
		ScheduleTime:  opts.ScheduleTime,
		ScheduledAt:   opts.ScheduledAt,
		ScheduleDays:  scheduleDaysStr,
		NextRunAt:     nextRunAt,
		RepeatUntil:   opts.RepeatUntil,
		IntervalMin:   opts.IntervalMin,
		WindowStart:   opts.WindowStart,
		WindowEnd:     opts.WindowEnd,
		QueueGroupID:  opts.QueueGroupID,
		QueuePosition: opts.QueuePosition,
		LoopQueue:     opts.LoopQueue,
		PinMessage:    opts.PinMessage,
		AutoDeleteMin: opts.AutoDeleteMin,
		Status:        "pending",
		SentCount:     0,
	}

	if err := s.repo.Create(ctx, post); err != nil {
		return nil, err
	}
	return post, nil
}

func (s *SchedulerService) CancelScheduledPost(ctx context.Context, id string, ownerID int64) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	return s.repo.UpdateStatus(ctx, id, "cancelled")
}

func (s *SchedulerService) PauseScheduledPost(ctx context.Context, id string, ownerID int64) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	return s.repo.UpdateStatus(ctx, id, "paused")
}

func (s *SchedulerService) ResumeScheduledPost(ctx context.Context, id string, ownerID int64) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	return s.repo.UpdateStatus(ctx, id, "pending")
}

func (s *SchedulerService) GetUserSchedules(ctx context.Context, ownerID int64) ([]models.ScheduledPost, error) {
	return s.repo.GetByOwner(ctx, ownerID)
}

func (s *SchedulerService) DeleteScheduledPost(ctx context.Context, id string, ownerID int64) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	return s.repo.Delete(ctx, id)
}

func (s *SchedulerService) UpdateScheduleTime(ctx context.Context, id string, ownerID int64, nextRunAt *time.Time, scheduleTime string) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	if scheduleTime != "" {
		if _, _, err := validateScheduleTime(scheduleTime); err != nil {
			return apperrors.BadRequest(err.Error())
		}
		if nextRunAt == nil {
			switch post.ScheduleType {
			case "daily":
				hour, min := parseHHMM(scheduleTime)
				now := time.Now().In(utils.BrazilTZ())
				next := time.Date(now.Year(), now.Month(), now.Day(), hour, min, 0, 0, utils.BrazilTZ()).UTC()
				if !next.After(time.Now().UTC()) {
					next = next.AddDate(0, 0, 1)
				}
				nextRunAt = &next
			case "weekly":
				copyPost := *post
				copyPost.ScheduleTime = scheduleTime
				if next := s.calculateNextWeekly(&copyPost); next != nil {
					nextRunAt = next
				}
			}
		}
	}
	return s.repo.UpdateScheduleTime(ctx, id, nextRunAt, scheduleTime)
}

func (s *SchedulerService) UpdateScheduleInterval(ctx context.Context, id string, ownerID int64, intervalMin int, windowStart, windowEnd string) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	if intervalMin < 5 {
		return apperrors.BadRequest("intervalo mínimo é 5 minutos")
	}
	if intervalMin > 1440 {
		return apperrors.BadRequest("para intervalos de 24h ou mais, use o tipo 'daily'")
	}

	normStart := ""
	normEnd := ""
	if windowStart != "" || windowEnd != "" {
		if windowStart == "" || windowEnd == "" {
			return apperrors.BadRequest("informe início e fim da janela de horário, ou deixe ambos vazios")
		}
		sh, sm, err := validateScheduleTime(windowStart)
		if err != nil {
			return apperrors.BadRequest("horário de início da janela inválido")
		}
		eh, em, err := validateScheduleTime(windowEnd)
		if err != nil {
			return apperrors.BadRequest("horário de fim da janela inválido")
		}
		normStart = fmt.Sprintf("%02d:%02d", sh, sm)
		normEnd = fmt.Sprintf("%02d:%02d", eh, em)
	}

	return s.repo.UpdateScheduleInterval(ctx, id, intervalMin, normStart, normEnd)
}

func (s *SchedulerService) UpdateSchedulePinMessage(ctx context.Context, id string, ownerID int64, pinMessage bool) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	return s.repo.UpdatePinMessage(ctx, id, pinMessage)
}

func (s *SchedulerService) UpdateScheduleAutoDelete(ctx context.Context, id string, ownerID int64, autoDeleteMin int) error {
	post, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return err
	}
	if post.OwnerID != ownerID {
		return fmt.Errorf("não autorizado")
	}
	return s.repo.UpdateAutoDelete(ctx, id, autoDeleteMin)
}

func (s *SchedulerService) GetScheduleByID(ctx context.Context, id string) (*models.ScheduledPost, error) {
	return s.repo.GetByID(ctx, id)
}
