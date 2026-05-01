package auth

import (
	"totalk/internal/middleware"
	"totalk/pkg/jwt"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(r chi.Router, h *Handler, tm *jwt.TokenManager) {
	r.Route("/auth", func(r chi.Router) {
		// ── Публичные ─────────────────────────────────────────────────────────
		r.Post("/register/step1", h.RegisterStep1)
		r.Post("/register/step2", h.RegisterStep2)
		r.Post("/register/step3", h.RegisterStep3) // finalize → возвращает токены
		r.Post("/login", h.Login)
		r.Post("/refresh", h.Refresh)

		// ── Защищённые (требуют valid access token) ───────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(tm))
			r.Get("/me", h.Me)
			r.Post("/logout", h.Logout)
		})
	})
}
