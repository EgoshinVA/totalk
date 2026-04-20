package middleware

import (
	"context"
	"net/http"
	"time"
	"totalk/internal/auth"
)

func RateLimiter(store auth.LimitStore, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// В качестве ключа берем IP (в реале можно ID юзера)
			key := r.RemoteAddr

			// Создаем контекст с таймаутом для запроса в Redis
			ctx, cancel := context.WithTimeout(r.Context(), 500*time.Millisecond)
			defer cancel()

			count, err := store.Increment(ctx, key, window)
			if err != nil {
				// Если Redis упал, в Highload обычно "пропускают" запрос (fail-open),
				// чтобы не класть весь сервис из-за кэша.
				next.ServeHTTP(w, r)
				return
			}

			if count > limit {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte("Слишком много запросов. Остынь!"))
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
