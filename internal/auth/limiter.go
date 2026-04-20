package auth

import (
	"context"
	"time"
)

type LimitStore interface {
	// Increment увеличивает счетчик и возвращает текущее значение
	Increment(ctx context.Context, key string, window time.Duration) (int, error)
}
