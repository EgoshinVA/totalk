package tasks

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"
	"totalk/internal/domain"
	"totalk/internal/middleware"
	"totalk/pkg/jwt"
	"totalk/pkg/respond"

	"github.com/go-chi/chi/v5"
)

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	fmt.Printf("📋 List tasks for userID: %d\n", userID)

	status := r.URL.Query().Get("status")
	tasks, err := h.svc.List(r.Context(), userID, status)
	if err != nil {
		fmt.Printf("❌ List error: %v\n", err)
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	fmt.Printf("✅ Found %d tasks\n", len(tasks))
	respond.OK(w, tasks)
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	task, err := h.svc.Complete(r.Context(), id, userID)
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			respond.Error(w, http.StatusNotFound, "task not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respond.OK(w, task)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}

	var body struct {
		Title       *string            `json:"title"`
		Description *string            `json:"description"`
		ScheduledAt *time.Time         `json:"scheduledAt"`
		Status      *domain.TaskStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	task, err := h.svc.Update(r.Context(), id, userID, UpdateInput{
		Title:       body.Title,
		Description: body.Description,
		ScheduledAt: body.ScheduledAt,
		Status:      body.Status,
	})
	if err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			respond.Error(w, http.StatusNotFound, "task not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respond.OK(w, task)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid id")
		return
	}
	if err := h.svc.Delete(r.Context(), id, userID); err != nil {
		if errors.Is(err, domain.ErrTaskNotFound) {
			respond.Error(w, http.StatusNotFound, "task not found")
			return
		}
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}
	respond.OK(w, map[string]string{"status": "deleted"})
}

func RegisterRoutes(r chi.Router, h *Handler, tm *jwt.TokenManager) {
	r.Route("/tasks", func(r chi.Router) {
		r.Use(middleware.Auth(tm))
		r.Get("/", h.List)
		r.Patch("/{id}/complete", h.Complete)
		r.Patch("/{id}", h.Update)
		r.Delete("/{id}", h.Delete)
	})
}
