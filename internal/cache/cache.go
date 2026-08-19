package cache

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"time"

	"github.com/leirbagxis/FreddyBot/internal/database/models"
	"github.com/redis/go-redis/v9"
)

type Service struct{}

func NewService() *Service {
	GetRedisClient()
	return &Service{}
}

func (s *Service) CreateSession(ctx context.Context, payload ChannelPayload) (*Session, error) {
	client := GetRedisClient()

	key, err := generateSessionKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate session key: %w", err)
	}

	data, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	err = client.Set(ctx, key, data, 10*time.Minute).Err()
	if err != nil {
		return nil, fmt.Errorf("failed to store in cache: %w", err)
	}

	return &Session{
		Key:     key,
		Payload: payload,
	}, nil
}

func (s *Service) GetSession(ctx context.Context, key string) (*ChannelPayload, error) {
	client := GetRedisClient()

	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, fmt.Errorf("session not found or expired")
		}
		return nil, fmt.Errorf("failed to get from cache: %w", err)
	}

	var payload ChannelPayload
	err = json.Unmarshal([]byte(data), &payload)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	return &payload, nil
}

func (s *Service) Get(ctx context.Context, key string, dest interface{}) error {
	// 1. Tenta L1 (Local)
	if val, found := localCache.Get(key); found {
		if data, ok := val.([]byte); ok {
			return json.Unmarshal(data, dest)
		}
	}

	client := GetRedisClient()

	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return fmt.Errorf("cache miss")
		}
		return fmt.Errorf("failed to get from cache: %w", err)
	}

	// Salva no L1 para a próxima (serializado para manter consistência no Get genérico)
	localCache.Set(key, []byte(data), 5*time.Minute)

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	return nil
}

func (s *Service) GetChannel(ctx context.Context, channelID int64) (*models.Channel, error) {
	key := fmt.Sprintf("channel:v2:%d", channelID)

	// 1. Tenta L1
	if val, found := localCache.Get(key); found {
		if channel, ok := val.(*models.Channel); ok {
			data, err := json.Marshal(channel)
			if err == nil {
				var copyChannel models.Channel
				if err := json.Unmarshal(data, &copyChannel); err == nil {
					return &copyChannel, nil
				}
			}
		}
	}

	// 2. Tenta L2 (Redis)
	var channel models.Channel
	err := s.Get(ctx, key, &channel)
	if err == nil {
		// Salva no L1 como objeto real para velocidade máxima
		localCache.Set(key, &channel, 5*time.Minute)
		return &channel, nil
	}

	return nil, err
}

func (s *Service) SetChannel(ctx context.Context, channel *models.Channel) error {
	key := fmt.Sprintf("channel:v2:%d", channel.ID)
	// Salva no L1
	localCache.Set(key, channel, 5*time.Minute)
	// Salva no L2 (Redis)
	return s.Set(ctx, key, channel, 1*time.Hour)
}

func (s *Service) InvalidateChannel(ctx context.Context, channelID int64) error {
	key := fmt.Sprintf("channel:v2:%d", channelID)
	localCache.Delete(key)

	client := GetRedisClient()
	client.Del(ctx, key)

	// Também limpa o debounce de atualização
	updateKey := fmt.Sprintf("last_update:channel:%d", channelID)
	return client.Del(ctx, updateKey).Err()
}

func (s *Service) Delete(ctx context.Context, key string) error {
	localCache.Delete(key)
	client := GetRedisClient()
	return client.Del(ctx, key).Err()
}

func (s *Service) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	client := GetRedisClient()

	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	// Salva no L1 também para consistência
	localCache.Set(key, data, expiration)

	return client.Set(ctx, key, data, expiration).Err()
}

// ### SELECTED CHANNEL ## \\

func (s *Service) SetSelectedChannel(ctx context.Context, userID, channelID int64) error {
	client := GetRedisClient()

	key := fmt.Sprintf("selected_channel:%d", userID)
	return client.Set(ctx, key, channelID, 24*time.Hour).Err()
}

func (s *Service) GetSelectedChannel(ctx context.Context, userID int64) (int64, error) {
	client := GetRedisClient()

	key := fmt.Sprintf("selected_channel:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, fmt.Errorf("session not found or expired")
		}
		return 0, fmt.Errorf("failed to get from cache: %w", err)
	}

	channelID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}

	return channelID, nil
}

// ### SEPARATOR CHANNEL ## \\

func (s *Service) SetAwaitingStickerSeparator(ctx context.Context, userID, channelID int64) error {
	client := GetRedisClient()

	key := fmt.Sprintf("awaiting_sticker:%d", userID)
	return client.Set(ctx, key, channelID, 5*time.Minute).Err()
}

func (s *Service) GetAwaitingStickerSeparator(ctx context.Context, userID int64) (int64, error) {
	client := GetRedisClient()

	key := fmt.Sprintf("awaiting_sticker:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, fmt.Errorf("session not found or expired")
		}
		return 0, fmt.Errorf("failed to get from cache: %w", err)
	}

	channelID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}

	return channelID, nil
}

func (s *Service) DeleteAwaitingStickerSeparator(ctx context.Context, userID int64) error {
	client := GetRedisClient()

	key := fmt.Sprintf("awaiting_sticker:%d", userID)

	return client.Del(ctx, key).Err()
}

// ### AWAITING CAPTION ## \\

func (s *Service) SetAwaitingCaption(ctx context.Context, userID, channelID int64) error {
	client := GetRedisClient()
	key := fmt.Sprintf("awaiting_caption:%d", userID)
	return client.Set(ctx, key, channelID, 5*time.Minute).Err()
}

func (s *Service) GetAwaitingCaption(ctx context.Context, userID int64) (int64, error) {
	client := GetRedisClient()
	key := fmt.Sprintf("awaiting_caption:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, fmt.Errorf("session not found or expired")
		}
		return 0, fmt.Errorf("failed to get from cache: %w", err)
	}
	channelID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}
	return channelID, nil
}

func (s *Service) DeleteAwaitingCaption(ctx context.Context, userID int64) error {
	client := GetRedisClient()
	key := fmt.Sprintf("awaiting_caption:%d", userID)
	return client.Del(ctx, key).Err()
}

// ### DELETE CHANNEL ## \\

func (s *Service) SetDeleteChannel(ctx context.Context, userID, channelID int64) error {
	client := GetRedisClient()

	key := fmt.Sprintf("delete_channel:%d", userID)
	return client.Set(ctx, key, channelID, 5*time.Minute).Err()
}

func (s *Service) GetDeleteChannel(ctx context.Context, userID int64) (int64, error) {
	client := GetRedisClient()

	key := fmt.Sprintf("delete_channel:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, fmt.Errorf("session not found or expired")
		}
		return 0, fmt.Errorf("failed to get from cache: %w", err)
	}

	channelID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}

	return channelID, nil
}

// ### TRANSFER CHANNEL ## \\

func (s *Service) SetTransferChannel(ctx context.Context, userID, channelID int64) error {
	client := GetRedisClient()

	key := fmt.Sprintf("transfer_channel:%d", userID)
	return client.Set(ctx, key, channelID, 5*time.Minute).Err()
}

func (s *Service) GetTransferChannel(ctx context.Context, userID int64) (int64, error) {
	client := GetRedisClient()

	key := fmt.Sprintf("transfer_channel:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return 0, fmt.Errorf("session not found or expired")
		}
		return 0, fmt.Errorf("failed to get from cache: %w", err)
	}

	channelID, err := strconv.ParseInt(data, 10, 64)
	if err != nil {
		return 0, err
	}

	return channelID, nil
}

// ### POST BUILDER ### \\

func (s *Service) SetPostBuilderState(ctx context.Context, userID int64, state PostBuilderState) error {
	client := GetRedisClient()

	key := fmt.Sprintf("post_builder:%d", userID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, data, 60*time.Minute).Err()
}

func (s *Service) GetPostBuilderState(ctx context.Context, userID int64) (*PostBuilderState, error) {
	client := GetRedisClient()

	key := fmt.Sprintf("post_builder:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}

	var state PostBuilderState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func (s *Service) DeletePostBuilderState(ctx context.Context, userID int64) error {
	client := GetRedisClient()

	key := fmt.Sprintf("post_builder:%d", userID)
	return client.Del(ctx, key).Err()
}

func (s *Service) SetScheduleState(ctx context.Context, userID int64, state ScheduleState) error {
	client := GetRedisClient()
	key := fmt.Sprintf("schedule_state:%d", userID)
	data, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, data, 10*time.Minute).Err()
}

func (s *Service) GetScheduleState(ctx context.Context, userID int64) (*ScheduleState, error) {
	client := GetRedisClient()
	key := fmt.Sprintf("schedule_state:%d", userID)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return nil, nil
		}
		return nil, err
	}
	var state ScheduleState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Service) DeleteScheduleState(ctx context.Context, userID int64) error {
	client := GetRedisClient()
	key := fmt.Sprintf("schedule_state:%d", userID)
	return client.Del(ctx, key).Err()
}

func (s *Service) SavePostBuilderSession(ctx context.Context, state PostBuilderState) (string, error) {
	id := generateShortID(8)
	if err := s.SetPostBuilderSession(ctx, id, state, 24*time.Hour); err != nil {
		return "", err
	}
	return id, nil
}

func (s *Service) SetPostBuilderSession(ctx context.Context, id string, state PostBuilderState, expiration time.Duration) error {
	client := GetRedisClient()
	key := fmt.Sprintf("pb_session:%s", id)

	data, err := json.Marshal(state)
	if err != nil {
		return err
	}

	return client.Set(ctx, key, data, expiration).Err()
}

func (s *Service) DeletePostBuilderSession(ctx context.Context, id string) error {
	client := GetRedisClient()
	key := fmt.Sprintf("pb_session:%s", id)
	return client.Del(ctx, key).Err()
}

func (s *Service) GetPostBuilderSession(ctx context.Context, id string) (*PostBuilderState, error) {
	client := GetRedisClient()

	key := fmt.Sprintf("pb_session:%s", id)
	data, err := client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil
		}
		return nil, err
	}

	var state PostBuilderState
	if err := json.Unmarshal([]byte(data), &state); err != nil {
		return nil, err
	}

	return &state, nil
}

func generateShortID(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		num, err := rand.Int(rand.Reader, big.NewInt(int64(len(charset))))
		if err != nil {
			randByte := make([]byte, 1)
			_, _ = rand.Read(randByte)
			b[i] = charset[int(randByte[0])%len(charset)]
			continue
		}
		b[i] = charset[num.Int64()]
	}
	return string(b)
}

// ### CHANNEL UPDATE DEBOUNCE ### \\

func (s *Service) SetLastChannelUpdate(ctx context.Context, channelID int64) error {
	client := GetRedisClient()
	key := fmt.Sprintf("last_update:channel:%d", channelID)
	// Cache por 1 hora para evitar chamadas excessivas ao GetChat
	return client.Set(ctx, key, time.Now().Unix(), 60*time.Minute).Err()
}

func (s *Service) ShouldUpdateChannel(ctx context.Context, channelID int64) bool {
	client := GetRedisClient()
	key := fmt.Sprintf("last_update:channel:%d", channelID)
	exists, err := client.Exists(ctx, key).Result()
	if err != nil {
		return true // Na dúvida, atualiza
	}
	return exists == 0
}

// ### DELETE ALL SESSIONS ### \\\
func (s *Service) DeleteAllUserSessionsBySuffix(ctx context.Context, userID int64) (int64, error) {
	// 1. Limpa o cache local (RAM)
	suffix := ":" + strconv.FormatInt(userID, 10)
	for k := range localCache.Items() {
		if strings.HasSuffix(k, suffix) {
			localCache.Delete(k)
		}
	}

	// 2. Limpa o Redis
	client := GetRedisClient()
	pattern := "*" + suffix

	var totalDeleted int64
	var cursor uint64
	const page = 1000
	const batch = 500

	for {
		keys, next, err := client.Scan(ctx, cursor, pattern, page).Result()
		if err != nil {
			return totalDeleted, err
		}
		cursor = next

		for i := 0; i < len(keys); i += batch {
			end := i + batch
			if end > len(keys) {
				end = len(keys)
			}
			chunk := keys[i:end]
			n, err := unlinkOrDel(ctx, client, chunk)
			if err != nil {
				return totalDeleted, err
			}
			totalDeleted += n
		}

		if cursor == 0 {
			break
		}
	}

	return totalDeleted, nil
}

func unlinkOrDel(ctx context.Context, client *redis.Client, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}
	pipe := client.Pipeline()
	unlink := pipe.Unlink(ctx, keys...)
	_, err := pipe.Exec(ctx)
	if err != nil {
		if strings.Contains(strings.ToLower(err.Error()), "unknown command") ||
			strings.Contains(strings.ToLower(err.Error()), "unlink") {
			pipe2 := client.Pipeline()
			for _, k := range keys {
				pipe2.Del(ctx, k)
			}
			_, err2 := pipe2.Exec(ctx)
			if err2 != nil {
				return 0, err2
			}
			return int64(len(keys)), nil
		}
		return 0, err
	}
	return unlink.Val(), nil
}

func generateSessionKey() (string, error) {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(bytes), nil
}
