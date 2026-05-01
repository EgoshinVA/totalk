package middleware

import (
	"context"
	"net/http"
	"strings"

	"totalk/pkg/jwt"
	"totalk/pkg/respond"
)

type contextKey string

const userIDKey contextKey = "userID"

// Auth — middleware для защищённых роутов. Читает Bearer-токен из заголовка,
// валидирует его и кладёт userID в контекст.
func Auth(tm *jwt.TokenManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractBearer(r)
			if token == "" {
				respond.Error(w, http.StatusUnauthorized, "authorization header required")
				return
			}

			claims, err := tm.ParseToken(token)
			if err != nil {
				respond.Error(w, http.StatusUnauthorized, "invalid or expired token")
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// UserIDFromCtx достаёт userID из контекста. Паникует если middleware не применён —
// это программная ошибка, не рантайм-ошибка.
func UserIDFromCtx(ctx context.Context) int64 {
	id, ok := ctx.Value(userIDKey).(int64)
	if !ok {
		panic("middleware.Auth must be applied before calling UserIDFromCtx")
	}
	return id
}

func extractBearer(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if !strings.HasPrefix(header, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(header, "Bearer ")
}
