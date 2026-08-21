package cache

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const SchedulerQueueKey = "scheduler:queue:v1"

type QueueItem struct {
	ScheduleID  string
	ScheduledAt time.Time
}

type SchedulerQueue interface {
	Add(ctx context.Context, scheduleID string, scheduledAt time.Time) error
	Remove(ctx context.Context, scheduleID string) error
	Next(ctx context.Context) (*QueueItem, error)
	Due(ctx context.Context, now time.Time, limit int) ([]QueueItem, error)
	Clear(ctx context.Context) error
	Size(ctx context.Context) (int64, error)
}

type RedisSchedulerQueue struct {
	client *redis.Client
	key    string
}

func NewRedisSchedulerQueue() *RedisSchedulerQueue {
	return &RedisSchedulerQueue{
		key: SchedulerQueueKey,
	}
}

// NewRedisSchedulerQueueWithClient permite injetar um redis.Client e chave customizados (útil para testes com miniredis).
func NewRedisSchedulerQueueWithClient(client *redis.Client, key string) *RedisSchedulerQueue {
	if key == "" {
		key = SchedulerQueueKey
	}
	return &RedisSchedulerQueue{
		client: client,
		key:    key,
	}
}

func (q *RedisSchedulerQueue) getClient() *redis.Client {
	if q.client != nil {
		return q.client
	}
	return GetRedisClient()
}

func (q *RedisSchedulerQueue) Add(ctx context.Context, scheduleID string, scheduledAt time.Time) error {
	client := q.getClient()
	if client == nil {
		return fmt.Errorf("redis client unavailable")
	}
	return client.ZAdd(ctx, q.key, redis.Z{
		Score:  float64(scheduledAt.Unix()),
		Member: scheduleID,
	}).Err()
}

func (q *RedisSchedulerQueue) Remove(ctx context.Context, scheduleID string) error {
	client := q.getClient()
	if client == nil {
		return fmt.Errorf("redis client unavailable")
	}
	return client.ZRem(ctx, q.key, scheduleID).Err()
}

func (q *RedisSchedulerQueue) Next(ctx context.Context) (*QueueItem, error) {
	client := q.getClient()
	if client == nil {
		return nil, fmt.Errorf("redis client unavailable")
	}

	res, err := client.ZRangeWithScores(ctx, q.key, 0, 0).Result()
	if err != nil {
		return nil, err
	}
	if len(res) == 0 {
		return nil, nil
	}

	id, ok := res[0].Member.(string)
	if !ok {
		id = fmt.Sprintf("%v", res[0].Member)
	}

	return &QueueItem{
		ScheduleID:  id,
		ScheduledAt: time.Unix(int64(res[0].Score), 0).UTC(),
	}, nil
}

func (q *RedisSchedulerQueue) Due(ctx context.Context, now time.Time, limit int) ([]QueueItem, error) {
	client := q.getClient()
	if client == nil {
		return nil, fmt.Errorf("redis client unavailable")
	}

	if limit <= 0 {
		limit = 20
	}

	opt := &redis.ZRangeBy{
		Min:    "-inf",
		Max:    strconv.FormatInt(now.Unix(), 10),
		Offset: 0,
		Count:  int64(limit),
	}

	res, err := client.ZRangeByScoreWithScores(ctx, q.key, opt).Result()
	if err != nil {
		return nil, err
	}

	items := make([]QueueItem, 0, len(res))
	for _, z := range res {
		id, ok := z.Member.(string)
		if !ok {
			id = fmt.Sprintf("%v", z.Member)
		}
		items = append(items, QueueItem{
			ScheduleID:  id,
			ScheduledAt: time.Unix(int64(z.Score), 0).UTC(),
		})
	}
	return items, nil
}

func (q *RedisSchedulerQueue) Clear(ctx context.Context) error {
	client := q.getClient()
	if client == nil {
		return fmt.Errorf("redis client unavailable")
	}
	return client.Del(ctx, q.key).Err()
}

func (q *RedisSchedulerQueue) Size(ctx context.Context) (int64, error) {
	client := q.getClient()
	if client == nil {
		return 0, fmt.Errorf("redis client unavailable")
	}
	return client.ZCard(ctx, q.key).Result()
}
