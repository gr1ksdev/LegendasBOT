package services

import (
	"context"
	"fmt"
	"strconv"
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

func setupAutoDeleteTest(t *testing.T) (*miniredis.Miniredis, *AutoDeleteService, *repositories.AutoDeleteRepository, cache.SchedulerQueue, *gorm.DB) {
	t.Helper()

	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("iniciar miniredis: %v", err)
	}

	redisCli := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	queue := cache.NewRedisSchedulerQueueWithClient(redisCli, fmt.Sprintf("test:autodelete:%d", time.Now().UnixNano()))

	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:autodel-test-%d?mode=memory&cache=shared", time.Now().UnixNano())), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}

	if err := db.AutoMigrate(&models.AutoDeletePost{}); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	repo := repositories.NewAutoDeleteRepository(db)
	svc := NewAutoDeleteService(repo, nil, queue)

	return mr, svc, repo, queue, db
}

func TestAutoDeleteRebuildQueue(t *testing.T) {
	mr, svc, repo, queue, _ := setupAutoDeleteTest(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	item1 := &models.AutoDeletePost{
		ChannelID: 100,
		MessageID: 1001,
		DeleteAt:  now.Add(10 * time.Minute),
		Status:    "pending",
	}
	item2 := &models.AutoDeletePost{
		ChannelID: 100,
		MessageID: 1002,
		DeleteAt:  now.Add(5 * time.Minute),
		Status:    "pending",
	}
	item3 := &models.AutoDeletePost{
		ChannelID: 100,
		MessageID: 1003,
		DeleteAt:  now.Add(1 * time.Minute),
		Status:    "deleted", // Não deve entrar na fila
	}
	item4 := &models.AutoDeletePost{
		ChannelID: 100,
		MessageID: 1004,
		DeleteAt:  now.Add(2 * time.Minute),
		Status:    "failed", // Não deve entrar na fila
	}

	_ = repo.Create(ctx, item1)
	_ = repo.Create(ctx, item2)
	_ = repo.Create(ctx, item3)
	_ = repo.Create(ctx, item4)

	// Executa rebuild
	if err := svc.RebuildQueue(ctx); err != nil {
		t.Fatalf("RebuildQueue: %v", err)
	}

	// Verifica tamanho da fila no Redis (deve ser 2)
	size, err := queue.Size(ctx)
	if err != nil {
		t.Fatalf("Size: %v", err)
	}
	if size != 2 {
		t.Fatalf("esperado size=2, obtido %d", size)
	}

	// Próximo deve ser item2 (5 min)
	next, err := queue.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if next == nil || next.ScheduleID != strconv.FormatUint(uint64(item2.ID), 10) {
		t.Fatalf("esperado próximo item %d, obtido %+v", item2.ID, next)
	}
}

func TestAutoDeleteScheduleAndWake(t *testing.T) {
	mr, svc, repo, queue, _ := setupAutoDeleteTest(t)
	defer mr.Close()

	ctx := context.Background()

	err := svc.ScheduleAutoDelete(ctx, 12345, 999, 10)
	if err != nil {
		t.Fatalf("ScheduleAutoDelete: %v", err)
	}

	// Deve haver 1 item no banco
	posts, err := repo.GetAllPending(ctx)
	if err != nil || len(posts) != 1 {
		t.Fatalf("GetAllPending: %v, count=%d", err, len(posts))
	}

	// Deve haver 1 item na fila Redis
	size, err := queue.Size(ctx)
	if err != nil || size != 1 {
		t.Fatalf("Size: %v, size=%d", err, size)
	}

	next, err := queue.Next(ctx)
	if err != nil || next == nil {
		t.Fatalf("Next: %v, next=%+v", err, next)
	}
	if next.ScheduleID != strconv.FormatUint(uint64(posts[0].ID), 10) {
		t.Fatalf("esperado ID %d na fila, obtido %s", posts[0].ID, next.ScheduleID)
	}

	// O canal wakeCh deve conter um sinal
	select {
	case <-svc.wakeCh:
		// OK
	default:
		t.Fatal("esperado sinal no wakeCh após agendamento")
	}
}

func TestAutoDeleteCASMarkDeletedAndFailed(t *testing.T) {
	mr, _, repo, _, _ := setupAutoDeleteTest(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().UTC()

	item := &models.AutoDeletePost{
		ChannelID: 100,
		MessageID: 1001,
		DeleteAt:  now,
		Status:    "pending",
	}
	_ = repo.Create(ctx, item)

	// Primeiro MarkDeleted deve funcionar
	err := repo.MarkDeleted(ctx, item.ID, now)
	if err != nil {
		t.Fatalf("MarkDeleted: %v", err)
	}

	updated, err := repo.GetByID(ctx, item.ID)
	if err != nil || updated.Status != "deleted" {
		t.Fatalf("esperado status 'deleted', obtido %s", updated.Status)
	}

	// Segundo MarkDeleted ou MarkFailed no mesmo item não deve quebrar
	err = repo.MarkFailed(ctx, item.ID, "algum erro")
	if err != nil {
		t.Fatalf("MarkFailed: %v", err)
	}

	// Status deve permanecer 'deleted' devido à condição WHERE status='pending'
	updated, _ = repo.GetByID(ctx, item.ID)
	if updated.Status != "deleted" {
		t.Fatalf("esperado status mantido 'deleted', obtido %s", updated.Status)
	}
}

func TestAutoDeleteQueueIsolation(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	redisCli := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	schedQueue := cache.NewRedisSchedulerQueueWithClient(redisCli, "scheduler:queue:v1")
	autoDelQueue := cache.NewRedisSchedulerQueueWithClient(redisCli, "auto_delete:queue:v1")

	ctx := context.Background()
	now := time.Now().UTC()

	_ = schedQueue.Add(ctx, "sch-1", now.Add(10*time.Minute))
	_ = autoDelQueue.Add(ctx, "autodel-1", now.Add(5*time.Minute))

	schedSize, _ := schedQueue.Size(ctx)
	autoDelSize, _ := autoDelQueue.Size(ctx)

	if schedSize != 1 || autoDelSize != 1 {
		t.Fatalf("esperado isolamento com 1 item cada, obtido sched=%d, autodel=%d", schedSize, autoDelSize)
	}

	schedNext, _ := schedQueue.Next(ctx)
	autoDelNext, _ := autoDelQueue.Next(ctx)

	if schedNext.ScheduleID != "sch-1" || autoDelNext.ScheduleID != "autodel-1" {
		t.Fatalf("esperado items corretos em cada fila, obtido sched=%s, autodel=%s", schedNext.ScheduleID, autoDelNext.ScheduleID)
	}
}
