package auth

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"totalk/pkg/respond"

	"totalk/internal/domain"
	"totalk/internal/middleware"

	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

// ── Request structs ───────────────────────────────────────────────────────────

type registerStep1Request struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type registerStep2Request struct {
	RegistrationToken string  `json:"registrationToken" validate:"required"`
	Name              string  `json:"name"               validate:"required"`
	SurName           string  `json:"surName"           validate:"required"`
	Patronymic        *string `json:"patronymic"`
}

type registerStep3Request struct {
	RegistrationToken string  `json:"registrationToken" validate:"required"`
	AvatarURL         *string `json:"avatarUrl"`
}

type loginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type refreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// ── Handler ───────────────────────────────────────────────────────────────────

type Handler struct {
	svc *Service
}

func NewHandler(svc *Service) *Handler {
	return &Handler{svc: svc}
}

// ── Helpers ───────────────────────────────────────────────────────────────────

func decode[T any](r *http.Request, w http.ResponseWriter) (T, bool) {
	var v T
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20) // 1 MB
	if err := json.NewDecoder(r.Body).Decode(&v); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid JSON body")
		return v, false
	}
	if err := validate.Struct(v); err != nil {
		respond.Error(w, http.StatusBadRequest, "validation failed: "+err.Error())
		return v, false
	}
	return v, true
}

func mapDomainError(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrUserAlreadyExists):
		return http.StatusConflict, "user with this email already exists"
	case errors.Is(err, domain.ErrUserNotFound):
		return http.StatusNotFound, "user not found"
	case errors.Is(err, domain.ErrInvalidCredentials):
		return http.StatusUnauthorized, "invalid email or password"
	case errors.Is(err, domain.ErrTokenInvalid):
		return http.StatusUnauthorized, "token is invalid or expired"
	case errors.Is(err, domain.ErrTokenNotFound):
		return http.StatusUnauthorized, "refresh token not found, please login again"
	default:
		return http.StatusInternalServerError, "internal server error"
	}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// POST /auth/register/step1
func (h *Handler) RegisterStep1(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[registerStep1Request](r, w)
	if !ok {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	token, err := h.svc.RegisterStep1(r.Context(), req.Email, req.Password)
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.Created(w, map[string]string{"registrationToken": token})
}

// POST /auth/register/step2
func (h *Handler) RegisterStep2(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[registerStep2Request](r, w)
	if !ok {
		return
	}

	err := h.svc.RegisterStep2(r.Context(), req.RegistrationToken, ProfileInput{
		FullName:   strings.TrimSpace(req.Name),
		SurName:    strings.TrimSpace(req.SurName),
		Patronymic: req.Patronymic,
	})
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.OK(w, map[string]string{"status": "profile updated"})
}

// POST /auth/register/step3  (finalize)
func (h *Handler) RegisterStep3(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[registerStep3Request](r, w)
	if !ok {
		return
	}

	authResp, err := h.svc.RegisterStep3(r.Context(), req.RegistrationToken, req.AvatarURL)
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.Created(w, authResp)
}

// POST /auth/login
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[loginRequest](r, w)
	if !ok {
		return
	}
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	authResp, err := h.svc.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.OK(w, authResp)
}

// POST /auth/refresh
func (h *Handler) Refresh(w http.ResponseWriter, r *http.Request) {
	req, ok := decode[refreshRequest](r, w)
	if !ok {
		return
	}

	pair, err := h.svc.Refresh(r.Context(), req.RefreshToken)
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.OK(w, pair)
}

// GET /auth/me  (protected)
func (h *Handler) Me(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	user, err := h.svc.Me(r.Context(), userID)
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.OK(w, domain.UserFromModel(user))
}

// POST /auth/logout  (protected)
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	if err := h.svc.Logout(r.Context(), userID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respond.OK(w, map[string]string{"status": "logged out"})
}
