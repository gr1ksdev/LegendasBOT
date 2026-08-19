package services

import (
	"context"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

type AutoDeleteService struct {
	repo *repositories.AutoDeleteRepository
	bot  *telego.Bot
}

func NewAutoDeleteService(repo *repositories.AutoDeleteRepository, bot *telego.Bot) *AutoDeleteService {
	return &AutoDeleteService{
		repo: repo,
		bot:  bot,
	}
}

func (s *AutoDeleteService) ScheduleAutoDelete(ctx context.Context, channelID int64, messageID int, autoDeleteMin int) error {
	if autoDeleteMin <= 0 || channelID == 0 || messageID == 0 {
		return nil
	}

	deleteAt := time.Now().Add(time.Duration(autoDeleteMin) * time.Minute)
	item := &models.AutoDeletePost{
		ChannelID: channelID,
		MessageID: messageID,
		DeleteAt:  deleteAt,
		Status:    "pending",
	}

	if err := s.repo.Create(ctx, item); err != nil {
		logger.Error("AUTODELETE", "Falha ao registrar auto-destruição da mensagem %d no canal %d: %v", messageID, channelID, err)
		return err
	}

	logger.Bot("⏱️ Auto-destruição agendada para mensagem %d no canal %d em %v", messageID, channelID, deleteAt.Format("15:04:05"))
	return nil
}

func (s *AutoDeleteService) Start(ctx context.Context) {
	logger.Info("AUTODELETE", "Serviço de auto-destruição de posts iniciado")
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	s.processDueDeletions(ctx)

	for {
		select {
		case <-ctx.Done():
			logger.Info("AUTODELETE", "Serviço de auto-destruição encerrado")
			return
		case <-ticker.C:
			s.processDueDeletions(ctx)
		}
	}
}

func (s *AutoDeleteService) processDueDeletions(ctx context.Context) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("AUTODELETE", "Panic recuperado na rotina de auto-exclusão: %v", r)
		}
	}()

	now := time.Now()
	posts, err := s.repo.GetDuePosts(ctx, now)
	if err != nil {
		logger.Error("AUTODELETE", "Erro ao buscar posts para exclusão: %v", err)
		return
	}

	if len(posts) == 0 {
		return
	}

	var wg sync.WaitGroup
	for _, p := range posts {
		wg.Add(1)
		item := p
		go func() {
			defer wg.Done()
			s.deleteSinglePost(ctx, &item)
		}()
	}
	wg.Wait()
}

func (s *AutoDeleteService) deleteSinglePost(ctx context.Context, item *models.AutoDeletePost) {
	defer func() {
		if r := recover(); r != nil {
			logger.Error("AUTODELETE", "Panic ao excluir mensagem %d no canal %d: %v", item.MessageID, item.ChannelID, r)
		}
	}()

	now := time.Now()
	err := s.bot.DeleteMessage(ctx, &telego.DeleteMessageParams{
		ChatID:    telego.ChatID{ID: item.ChannelID},
		MessageID: item.MessageID,
	})

	if err != nil {
		logger.Error("AUTODELETE", "❌ Falha ao excluir mensagem %d no canal %d: %v", item.MessageID, item.ChannelID, err)
		_ = s.repo.MarkFailed(ctx, item.ID, err.Error())
		return
	}

	logger.Bot("🗑️ Mensagem %d excluída automaticamente no canal %d", item.MessageID, item.ChannelID)
	_ = s.repo.MarkDeleted(ctx, item.ID, now)
}
