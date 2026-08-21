package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/leirbagxis/FreddyBot/internal/cache"
	"github.com/redis/go-redis/v9"
)

func setupTestQueue(t *testing.T) (*miniredis.Miniredis, cache.SchedulerQueue) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("falha ao iniciar miniredis: %v", err)
	}

	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	queue := cache.NewRedisSchedulerQueueWithClient(client, "test:scheduler:queue")
	return mr, queue
}

func TestQueueAddAndNext(t *testing.T) {
	mr, queue := setupTestQueue(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	// 1. Fila vazia
	next, err := queue.Next(ctx)
	if err != nil {
		t.Fatalf("Next em fila vazia retornou erro: %v", err)
	}
	if next != nil {
		t.Fatalf("Next em fila vazia deveria retornar nil, retornou: %+v", next)
	}

	// 2. Adiciona 3 schedules fora de ordem
	t1 := now.Add(10 * time.Minute)
	t2 := now.Add(5 * time.Minute)
	t3 := now.Add(15 * time.Minute)

	if err := queue.Add(ctx, "sch_10m", t1); err != nil {
		t.Fatalf("Add sch_10m: %v", err)
	}
	if err := queue.Add(ctx, "sch_5m", t2); err != nil {
		t.Fatalf("Add sch_5m: %v", err)
	}
	if err := queue.Add(ctx, "sch_15m", t3); err != nil {
		t.Fatalf("Add sch_15m: %v", err)
	}

	// 3. Next deve retornar o mais próximo (sch_5m)
	next, err = queue.Next(ctx)
	if err != nil {
		t.Fatalf("Next retornou erro: %v", err)
	}
	if next == nil || next.ScheduleID != "sch_5m" {
		t.Fatalf("Next = %+v, esperado sch_5m", next)
	}
	if !next.ScheduledAt.Equal(t2) {
		t.Fatalf("ScheduledAt = %v, esperado %v", next.ScheduledAt, t2)
	}

	// 4. Checa tamanho
	size, err := queue.Size(ctx)
	if err != nil || size != 3 {
		t.Fatalf("Size = %d, err = %v, esperado 3", size, err)
	}
}

func TestQueueUpdateScore(t *testing.T) {
	mr, queue := setupTestQueue(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	// Adiciona sch_1 para daqui 10min e sch_2 para daqui 20min
	_ = queue.Add(ctx, "sch_1", now.Add(10*time.Minute))
	_ = queue.Add(ctx, "sch_2", now.Add(20*time.Minute))

	// Atualiza sch_2 para daqui 2min (deve virar o primeiro)
	newT2 := now.Add(2 * time.Minute)
	if err := queue.Add(ctx, "sch_2", newT2); err != nil {
		t.Fatalf("Add (update) sch_2: %v", err)
	}

	next, err := queue.Next(ctx)
	if err != nil {
		t.Fatalf("Next: %v", err)
	}
	if next == nil || next.ScheduleID != "sch_2" {
		t.Fatalf("Next = %+v, esperado sch_2 após atualização de score", next)
	}

	// Tamanho ainda deve ser 2
	size, _ := queue.Size(ctx)
	if size != 2 {
		t.Fatalf("Size = %d, esperado 2 (sem duplicação)", size)
	}
}

func TestQueueRemove(t *testing.T) {
	mr, queue := setupTestQueue(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	_ = queue.Add(ctx, "sch_1", now.Add(5*time.Minute))
	_ = queue.Add(ctx, "sch_2", now.Add(10*time.Minute))

	if err := queue.Remove(ctx, "sch_1"); err != nil {
		t.Fatalf("Remove: %v", err)
	}

	next, _ := queue.Next(ctx)
	if next == nil || next.ScheduleID != "sch_2" {
		t.Fatalf("Next = %+v, esperado sch_2 após remover sch_1", next)
	}

	_ = queue.Remove(ctx, "sch_2")
	next, _ = queue.Next(ctx)
	if next != nil {
		t.Fatalf("Next deveria ser nil após remover todos")
	}
}

func TestQueueDue(t *testing.T) {
	mr, queue := setupTestQueue(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	// 2 vencidos e 2 futuros
	_ = queue.Add(ctx, "overdue_1", now.Add(-5*time.Minute))
	_ = queue.Add(ctx, "overdue_2", now.Add(-1*time.Minute))
	_ = queue.Add(ctx, "future_1", now.Add(5*time.Minute))
	_ = queue.Add(ctx, "future_2", now.Add(10*time.Minute))

	due, err := queue.Due(ctx, now, 10)
	if err != nil {
		t.Fatalf("Due: %v", err)
	}
	if len(due) != 2 {
		t.Fatalf("len(due) = %d, esperado 2", len(due))
	}
	if due[0].ScheduleID != "overdue_1" || due[1].ScheduleID != "overdue_2" {
		t.Fatalf("Due order = %+v, esperado [overdue_1, overdue_2]", due)
	}

	// Testando limite
	dueLimit1, err := queue.Due(ctx, now, 1)
	if err != nil {
		t.Fatalf("Due limit 1: %v", err)
	}
	if len(dueLimit1) != 1 || dueLimit1[0].ScheduleID != "overdue_1" {
		t.Fatalf("Due limit 1 = %+v, esperado [overdue_1]", dueLimit1)
	}
}

func TestQueueClear(t *testing.T) {
	mr, queue := setupTestQueue(t)
	defer mr.Close()

	ctx := context.Background()
	now := time.Now().Truncate(time.Second).UTC()

	_ = queue.Add(ctx, "sch_1", now.Add(5*time.Minute))
	_ = queue.Add(ctx, "sch_2", now.Add(10*time.Minute))

	if err := queue.Clear(ctx); err != nil {
		t.Fatalf("Clear: %v", err)
	}

	size, _ := queue.Size(ctx)
	if size != 0 {
		t.Fatalf("Size = %d após Clear, esperado 0", size)
	}
}

func TestChannelPhotoCache(t *testing.T) {
	mr, err := miniredis.Run()
	if err != nil {
		t.Fatalf("miniredis: %v", err)
	}
	defer mr.Close()

	svc := cache.NewService()
	ctx := context.Background()

	// 1. Miss inicial
	photo, err := svc.GetChannelPhoto(ctx, 12345)
	if err == nil || photo != nil {
		t.Fatalf("expected cache miss, got %+v", photo)
	}

	// 2. Set
	sample := &cache.CachedChannelPhoto{
		Data:        []byte("fake-image-bytes"),
		ContentType: "image/jpeg",
		ETag:        `W/"photo-12345-abc"`,
	}
	if err := svc.SetChannelPhoto(ctx, 12345, sample, time.Hour); err != nil {
		t.Fatalf("SetChannelPhoto: %v", err)
	}

	// 3. Get
	cached, err := svc.GetChannelPhoto(ctx, 12345)
	if err != nil || cached == nil {
		t.Fatalf("GetChannelPhoto: %v", err)
	}
	if string(cached.Data) != "fake-image-bytes" || cached.ContentType != "image/jpeg" || cached.ETag != `W/"photo-12345-abc"` {
		t.Fatalf("unexpected cached photo: %+v", cached)
	}

	// 4. Invalidate
	_ = svc.InvalidateChannelPhoto(ctx, 12345)
	after, _ := svc.GetChannelPhoto(ctx, 12345)
	if after != nil {
		t.Fatalf("expected nil after invalidate, got %+v", after)
	}
}
