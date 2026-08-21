package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/glebarez/sqlite"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/leirbagxis/FreddyBot/internal/database/repositories"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func setupSchedulerTest(t *testing.T) (*miniredis.Miniredis, *SchedulerService, *repositories.ScheduledPostRepository, *repositories.ChannelRepository, *gorm.DB) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("iniciar miniredis: %v", err)
	}

	redisCli := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	queue := cache.NewRedisSchedulerQueueWithClient(redisCli, fmt.Sprintf("test:scheduler:%d", time.Now().UnixNano()))

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:sched-test-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.Channel{}, &models.ScheduledPost{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	postRepo := repositories.NewScheduledPostRepository(db)
	chanRepo := repositories.NewChannelRepository(db)

	svc := NewSchedulerService(postRepo, chanRepo, nil, nil, queue)
	return mr, svc, postRepo, chanRepo, db
}

func TestSchedulerRebuildQueue(t *testing.T) {
	mr, svc, postRepo, _, _ := setupSchedulerTest(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	// Cria posts no DB
	p1 := &models.ScheduledPost{
		ID:           "post-pending-1",
		OwnerID:      1,
		ChannelID:    100,
		PostData:     `{"title":"test1"}`,
		ScheduleType: "once",
		NextRunAt:    now.Add(10 * time.Minute),
		Status:       "pending",
	}
	p2 := &models.ScheduledPost{
		ID:           "post-pending-2",
		OwnerID:      1,
		ChannelID:    100,
		PostData:     `{"title":"test2"}`,
		ScheduleType: "once",
		NextRunAt:    now.Add(5 * time.Minute),
		Status:       "pending",
	}
	p3 := &models.ScheduledPost{
		ID:           "post-paused",
		OwnerID:      1,
		ChannelID:    100,
		PostData:     `{"title":"paused"}`,
		ScheduleType: "once",
		NextRunAt:    now.Add(1 * time.Minute),
		Status:       "paused", // Não deve entrar na fila
	}
	p4 := &models.ScheduledPost{
		ID:           "post-sent",
		OwnerID:      1,
		ChannelID:    100,
		PostData:     `{"title":"sent"}`,
		ScheduleType: "once",
		NextRunAt:    now.Add(-1 * time.Minute),
		Status:       "sent", // Não deve entrar na fila
	}

	_ = postRepo.Create(ctx, p1)
	_ = postRepo.Create(ctx, p2)
	_ = postRepo.Create(ctx, p3)
	_ = postRepo.Create(ctx, p4)

	// Rebuild
	if err := svc.RebuildQueue(ctx); err != nil {
		t.Fatalf("RebuildQueue: %v", err)
	}

	size, err := svc.queue.Size(ctx)
	if err != nil || size != 2 {
		t.Fatalf("Queue size = %d (err = %v), esperado 2", size, err)
	}

	next, err := svc.queue.Next(ctx)
	if err != nil || next == nil || next.ScheduleID != "post-pending-2" {
		t.Fatalf("Next = %+v, esperado post-pending-2", next)
	}
}

func TestSchedulerCRUDUpdatesQueue(t *testing.T) {
	mr, svc, _, chanRepo, _ := setupSchedulerTest(t)
	defer mr.Close()

	ctx := context.Background()
	_ = chanRepo.CreateChannel(ctx, &models.Channel{ID: 100, OwnerID: 1, Title: "Test Channel"})

	// 1. CreateScheduledPost
	post, err := svc.CreateScheduledPost(ctx, 1, 100, `{"title":"test"}`, ScheduleOptions{
		ScheduleType: "daily",
		ScheduleTime: "14:00",
	})
	if err != nil {
		t.Fatalf("CreateScheduledPost: %v", err)
	}

	size, _ := svc.queue.Size(ctx)
	if size != 1 {
		t.Fatalf("Size após Create = %d, esperado 1", size)
	}

	next, _ := svc.queue.Next(ctx)
	if next == nil || next.ScheduleID != post.ID {
		t.Fatalf("Next após Create = %+v, esperado %s", next, post.ID)
	}

	// 2. PauseScheduledPost
	if err := svc.PauseScheduledPost(ctx, post.ID, 1); err != nil {
		t.Fatalf("PauseScheduledPost: %v", err)
	}
	size, _ = svc.queue.Size(ctx)
	if size != 0 {
		t.Fatalf("Size após Pause = %d, esperado 0", size)
	}

	// 3. ResumeScheduledPost
	if err := svc.ResumeScheduledPost(ctx, post.ID, 1); err != nil {
		t.Fatalf("ResumeScheduledPost: %v", err)
	}
	size, _ = svc.queue.Size(ctx)
	if size != 1 {
		t.Fatalf("Size após Resume = %d, esperado 1", size)
	}

	// 4. UpdateScheduleTime
	newNextRun := time.Now().Add(2 * time.Hour).Truncate(time.Second).UTC()
	if err := svc.UpdateScheduleTime(ctx, post.ID, 1, &newNextRun, "16:00"); err != nil {
		t.Fatalf("UpdateScheduleTime: %v", err)
	}
	next, _ = svc.queue.Next(ctx)
	if next == nil || !next.ScheduledAt.Equal(newNextRun) {
		t.Fatalf("Next ScheduledAt = %v, esperado %v", next.ScheduledAt, newNextRun)
	}

	// 5. CancelScheduledPost
	if err := svc.CancelScheduledPost(ctx, post.ID, 1); err != nil {
		t.Fatalf("CancelScheduledPost: %v", err)
	}
	size, _ = svc.queue.Size(ctx)
	if size != 0 {
		t.Fatalf("Size após Cancel = %d, esperado 0", size)
	}

	// 6. DeleteScheduledPost
	_ = svc.ResumeScheduledPost(ctx, post.ID, 1)
	if err := svc.DeleteScheduledPost(ctx, post.ID, 1); err != nil {
		t.Fatalf("DeleteScheduledPost: %v", err)
	}
	size, _ = svc.queue.Size(ctx)
	if size != 0 {
		t.Fatalf("Size após Delete = %d, esperado 0", size)
	}
}

func TestSchedulerEventDrivenExecution(t *testing.T) {
	mr, svc, postRepo, chanRepo, _ := setupSchedulerTest(t)
	defer mr.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_ = chanRepo.CreateChannel(ctx, &models.Channel{ID: 100, OwnerID: 1, Title: "Test Channel"})

	// Inicia o scheduler em background
	done := make(chan struct{})
	go func() {
		svc.Start(ctx)
		close(done)
	}()

	// Cria post vencido (due)
	pastTime := time.Now().Add(-5 * time.Second).Truncate(time.Second).UTC()
	post := &models.ScheduledPost{
		ID:           "test-due-exec",
		OwnerID:      1,
		ChannelID:    100,
		PostData:     `{"title":"exec"}`,
		ScheduleType: "once",
		NextRunAt:    pastTime,
		Status:       "pending",
	}
	_ = postRepo.Create(ctx, post)
	_ = svc.queue.Add(ctx, post.ID, post.NextRunAt)

	// Envia Wake para acordar o scheduler
	svc.Wake()

	// Aguarda processamento
	time.Sleep(100 * time.Millisecond)

	// Verifica se o claim foi executado e o post foi processado
	p, err := postRepo.GetByID(ctx, post.ID)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	// Como bot é nil no teste, o buildAndSend tenta enviar e falha ou o post é marcado como sent/failed
	if p.Status == "pending" {
		t.Fatalf("Post status ainda é pending após execução, esperado sent ou failed")
	}

	// Encerra loop
	cancel()
	select {
	case <-done:
		// Sucesso
	case <-time.After(2 * time.Second):
		t.Fatalf("Scheduler não encerrou graciosamente")
	}
}
