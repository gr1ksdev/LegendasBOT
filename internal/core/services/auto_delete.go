package services

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/mymmrac/telego"
)

type AutoDeleteService struct {
	repo   *repositories.AutoDeleteRepository
	bot    *telego.Bot
	queue  cache.SchedulerQueue
	wakeCh chan struct{}
}

func NewAutoDeleteService(repo *repositories.AutoDeleteRepository, bot *telego.Bot, queue cache.SchedulerQueue) *AutoDeleteService {
	return &AutoDeleteService{
		repo:   repo,
		bot:    bot,
		queue:  queue,
		wakeCh: make(chan struct{}, 1),
	}
}

// Wake envia um sinal non-blocking para o loop do auto-delete recalcular seu timer imediatamente.
func (s *AutoDeleteService) Wake() {
	select {
	case s.wakeCh <- struct{}{}:
	default:
	}
}

// RebuildQueue recarrega todos os auto-deletes pendentes do PostgreSQL e popula a fila Sorted Set do Redis.
func (s *AutoDeleteService) RebuildQueue(ctx context.Context) error {
	posts, err := s.repo.GetAllPending(ctx)
	if err != nil {
		return fmt.Errorf("fetch pending auto-deletes from db: %w", err)
	}

	if s.queue == nil {
		return nil
	}

	if err := s.queue.Clear(ctx); err != nil {
		logger.Warn("AUTODELETE", "Aviso ao limpar fila Redis: %v", err)
	}

	for _, p := range posts {
		idStr := strconv.FormatUint(uint64(p.ID), 10)
		if err := s.queue.Add(ctx, idStr, p.DeleteAt); err != nil {
			logger.Error("AUTODELETE", "Erro ao adicionar post %d à fila Redis: %v", p.ID, err)
		}
	}

	logger.Info("AUTODELETE", "Queue rebuilt: %d pending auto-deletes", len(posts))
	return nil
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

	if s.queue != nil {
		idStr := strconv.FormatUint(uint64(item.ID), 10)
		if err := s.queue.Add(ctx, idStr, deleteAt); err != nil {
			logger.Warn("AUTODELETE", "Aviso: falha ao adicionar na fila Redis (será reconstruída no próximo rebuild): %v", err)
		}
		s.Wake()
	}

	logger.Bot("⏱️ Auto-destruição agendada para mensagem %d no canal %d em %v", messageID, channelID, deleteAt.Format("15:04:05"))
	return nil
}

func (s *AutoDeleteService) Start(ctx context.Context) {
	logger.Info("AUTODELETE", "Serviço de auto-destruição de posts iniciado (event-driven com Redis ZSET)")

	for {
		select {
		case <-ctx.Done():
			logger.Info("AUTODELETE", "Serviço de auto-destruição encerrado")
			return
		default:
		}

		if s.queue == nil {
			logger.Error("AUTODELETE", "Fila do autodelete não inicializada")
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Minute):
				continue
			}
		}

		// 1. Consultar o próximo item na fila Redis
		next, err := s.queue.Next(ctx)
		if err != nil {
			logger.Error("AUTODELETE", "Erro ao consultar próximo auto-delete no Redis: %v", err)
			// Degradação graciosa: aguarda 3 minutos ou wake/shutdown antes de tentar reconectar/rebuild
			select {
			case <-ctx.Done():
				return
			case <-s.wakeCh:
				continue
			case <-time.After(3 * time.Minute):
				_ = s.RebuildQueue(ctx)
				continue
			}
		}

		now := time.Now().UTC()

		// 2. Se a fila estiver vazia, aguardar evento (wake) ou shutdown
		if next == nil {
			select {
			case <-ctx.Done():
				logger.Info("AUTODELETE", "Serviço de auto-destruição encerrado")
				return
			case <-s.wakeCh:
				continue
			}
		}

		// 3. Se o próximo item for no futuro, aguardar até a hora ou wake/shutdown
		delay := next.ScheduledAt.Sub(now)
		if delay > 0 {
			timer := time.NewTimer(delay)
			select {
			case <-ctx.Done():
				timer.Stop()
				logger.Info("AUTODELETE", "Serviço de auto-destruição encerrado")
				return
			case <-s.wakeCh:
				timer.Stop()
				continue
			case <-timer.C:
				// Hora chegou!
			}
		}

		// 4. Buscar todos os auto-deletes vencidos até o momento
		dueItems, err := s.queue.Due(ctx, time.Now().UTC(), 50)
		if err != nil {
			logger.Error("AUTODELETE", "Erro ao buscar itens vencidos no Redis: %v", err)
			continue
		}

		if len(dueItems) == 0 {
			continue
		}

		var wg sync.WaitGroup
		for _, item := range dueItems {
			itemIDStr := item.ScheduleID
			if s.queue != nil {
				_ = s.queue.Remove(ctx, itemIDStr)
			}

			id64, err := strconv.ParseUint(itemIDStr, 10, 64)
			if err != nil {
				logger.Error("AUTODELETE", "ID inválido no item da fila %s: %v", itemIDStr, err)
				continue
			}

			post, getErr := s.repo.GetByID(ctx, uint(id64))
			if getErr != nil || post == nil {
				logger.Error("AUTODELETE", "Erro ao buscar post %d do banco: %v", id64, getErr)
				continue
			}

			if post.Status != "pending" {
				continue
			}

			wg.Add(1)
			p := post
			go func() {
				defer wg.Done()
				s.deleteSinglePost(ctx, p)
			}()
		}
		wg.Wait()
	}
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
