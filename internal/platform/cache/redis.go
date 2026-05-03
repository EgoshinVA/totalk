package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"totalk/internal/domain"

	"github.com/redis/go-redis/v9"
)

// RedisStore реализует TokenRepository, RegistrationTokenRepository и RateLimiterStore.
type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

func Init(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: "redis:6379",
	})

	// Проверяем соединение
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

// ── TokenRepository ───────────────────────────────────────────────────────────

const refreshKeyPrefix = "refresh:"

func refreshKey(userID int64) string {
	return fmt.Sprintf("%s%d", refreshKeyPrefix, userID)
}

// SetRefreshToken сохраняет refresh-токен в Redis с TTL 7 дней.
func (s *RedisStore) SetRefreshToken(ctx context.Context, userID int64, token string) error {
	return s.client.Set(ctx, refreshKey(userID), token, 7*24*time.Hour).Err()
}

// GetRefreshToken возвращает сохранённый refresh-токен.
func (s *RedisStore) GetRefreshToken(ctx context.Context, userID int64) (string, error) {
	val, err := s.client.Get(ctx, refreshKey(userID)).Result()
	if err == redis.Nil {
		return "", nil // не найден — не ошибка, вернём пустую строку
	}
	return val, err
}

// DeleteRefreshToken используется при logout.
func (s *RedisStore) DeleteRefreshToken(ctx context.Context, userID int64) error {
	return s.client.Del(ctx, refreshKey(userID)).Err()
}

// ── RegistrationTokenRepository ──────────────────────────────────────────────

const regKeyPrefix = "reg:"

func regKey(token string) string {
	return regKeyPrefix + token
}

// SetRegistrationToken сохраняет RegistrationData в Redis.
// TTL 15 минут — достаточно для прохождения 3 шагов.
func (s *RedisStore) SetRegistrationToken(ctx context.Context, token string, data *domain.RegistrationData) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, regKey(token), jsonData, 15*time.Minute).Err()
}

// GetRegistrationToken возвращает RegistrationData по registration_token.
func (s *RedisStore) GetRegistrationToken(ctx context.Context, token string) (*domain.RegistrationData, error) {
	val, err := s.client.Get(ctx, regKey(token)).Result()
	if err == redis.Nil {
		return nil, nil // не найден — не ошибка, вернём nil
	}
	if err != nil {
		return nil, err
	}

	var data domain.RegistrationData
	if err := json.Unmarshal([]byte(val), &data); err != nil {
		return nil, err
	}
	return &data, nil
}

// DeleteRegistrationToken удаляет токен после финализации.
func (s *RedisStore) DeleteRegistrationToken(ctx context.Context, token string) error {
	return s.client.Del(ctx, regKey(token)).Err()
}

// ── RateLimiter (sliding window) ─────────────────────────────────────────────

// Allow проверяет лимит запросов для ключа key (например IP).
// Возвращает true, если запрос разрешён.
func (s *RedisStore) Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	now := time.Now().UnixMilli()
	windowMs := window.Milliseconds()
	pipe := s.client.Pipeline()

	rKey := "rl:" + key
	pipe.ZRemRangeByScore(ctx, rKey, "0", fmt.Sprintf("%d", now-windowMs))
	pipe.ZCard(ctx, rKey)
	pipe.ZAdd(ctx, rKey, redis.Z{Score: float64(now), Member: now})
	pipe.Expire(ctx, rKey, window)

	cmds, err := pipe.Exec(ctx)
	if err != nil {
		return false, err
	}

	count := cmds[1].(*redis.IntCmd).Val()
	return count < int64(limit), nil
}
