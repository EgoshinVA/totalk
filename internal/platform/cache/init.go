package cache

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

func Init(ctx context.Context) (*redis.Client, error) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Проверяем соединение
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	return client, nil
}

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) *RedisStore {
	return &RedisStore{client: client}
}

// Increment реализует наш интерфейс LimitStore
func (s *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int, error) {
	// Мы используем пайплайн (очередь команд), чтобы за один сетевой поход в Redis
	// и увеличить счетчик, и обновить время жизни (TTL).
	pipe := s.client.Pipeline()

	// 1. Увеличиваем значение ключа на 1.
	// Если ключа нет, Redis создаст его со значением 1.
	incr := pipe.Incr(ctx, key)

	// 2. Устанавливаем время жизни ключа (например, на 1 минуту).
	// Это гарантирует, что через минуту счетчик сбросится сам.
	pipe.Expire(ctx, key, window)

	// Выполняем обе команды одной пачкой
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	// Возвращаем результат инкремента
	return int(incr.Val()), nil
}

func (s *RedisStore) SetRefreshToken(ctx context.Context, userID uint, token string) error {
	// Ключ вида "refresh_token:1"
	key := fmt.Sprintf("refresh_token:%d", userID)

	// Сохраняем токен в Redis на 7 дней
	// Мы используем userID как ключ, чтобы один юзер = один активный рефреш-токен
	return s.client.Set(ctx, key, token, 7*24*time.Hour).Err()
}
