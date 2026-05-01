package middleware

import (
	"context"
	"net/http"
	"time"

	"totalk/pkg/respond"
)

type rateLimitStore interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}

// RateLimiter ограничивает количество запросов по IP.
// limit — максимум запросов за window.
func RateLimiter(store rateLimitStore, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := realIP(r)

			ok, err := store.Allow(r.Context(), ip, limit, window)
			if err != nil {
				// Redis недоступен — не блокируем трафик, просто пропускаем
				next.ServeHTTP(w, r)
				return
			}
			if !ok {
				respond.Error(w, http.StatusTooManyRequests, "too many requests, slow down")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func realIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
