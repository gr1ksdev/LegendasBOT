package cache

import (
	"context"
	"sync"
	"time"

	"github.com/leirbagxis/FreddyBot/pkg/config"
	"github.com/leirbagxis/FreddyBot/pkg/logger"
	"github.com/redis/go-redis/v9"
)

var (
	redisClient *redis.Client
	redisMu     sync.Mutex
)

func GetRedisClient() *redis.Client {
	redisMu.Lock()
	defer redisMu.Unlock()

	if redisClient != nil {
		return redisClient
	}

	opt, err := redis.ParseURL(config.RedisAddr)
	if err != nil {
		logger.Error("REDIS", "URL do Redis inválida (%s): %v", config.RedisAddr, err)
		return nil
	}
	opt.PoolSize = 10
	opt.MinIdleConns = 5
	opt.MaxRetries = 5
	opt.DialTimeout = 5 * time.Second
	opt.ReadTimeout = 5 * time.Second
	opt.WriteTimeout = 3 * time.Second

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		logger.Error("REDIS", "Falha ao conectar no Redis (%s): %v", config.RedisAddr, err)
		_ = client.Close()
		return nil
	}

	redisClient = client
	logger.Bot("✅ Redis conectado com sucesso")
	return redisClient
}

func CloseRedis() error {
	redisMu.Lock()
	defer redisMu.Unlock()

	if redisClient != nil {
		err := redisClient.Close()
		redisClient = nil
		return err
	}
	return nil
}

func HealthCheck(ctx context.Context) error {
	client := GetRedisClient()
	if client == nil {
		return redis.Nil
	}
	return client.Ping(ctx).Err()
}
