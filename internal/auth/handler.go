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

// @Summary      Register step 1
// @Description  Email + password, returns registration token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerStep1Request true "Email and password"
// @Success      201  {object}  map[string]string
// @Failure      400  {object}  respond.ErrorResponse
// @Failure      409  {object}  respond.ErrorResponse
// @Router       /auth/register/step1 [post]
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

// @Summary      Register step 2
// @Description  Name and surname, returns status
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerStep2Request true "Profile info"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  respond.ErrorResponse
// @Router       /auth/register/step2 [post]
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

// @Summary      Register step 3
// @Description  Avatar URL, finalizes registration and returns tokens + user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body registerStep3Request true "Avatar URL"
// @Success      201  {object}  domain.AuthResponse
// @Failure      400  {object}  respond.ErrorResponse
// @Router       /auth/register/step3 [post]
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

// @Summary      Login
// @Description  Returns access + refresh tokens and user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body loginRequest true "Credentials"
// @Success      200  {object}  domain.AuthResponse
// @Failure      401  {object}  respond.ErrorResponse
// @Router       /auth/login [post]
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

// @Summary      Refresh tokens
// @Description  Returns new token pair
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body body refreshRequest true "Refresh token"
// @Success      200  {object}  domain.TokenPair
// @Failure      401  {object}  respond.ErrorResponse
// @Router       /auth/refresh [post]
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

// @Summary      Get current user
// @Tags         auth
// @Produce      json
// @Security     BearerAuth
// @Success      200  {object}  domain.UserResponse
// @Failure      401  {object}  respond.ErrorResponse
// @Router       /auth/me [get]
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

// @Summary      Logout
// @Tags         auth
// @Security     BearerAuth
// @Success      200  {object}  map[string]string
// @Failure      500  {object}  respond.ErrorResponse
// @Router       /auth/logout [post]
func (h *Handler) Logout(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	if err := h.svc.Logout(r.Context(), userID); err != nil {
		respond.Error(w, http.StatusInternalServerError, "internal server error")
		return
	}

	respond.OK(w, map[string]string{"status": "logged out"})
}

// @Summary      Update current user
// @Tags         auth
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body updateMeRequest true "Fields to update"
// @Success      200  {object}  domain.UserResponse
// @Failure      400  {object}  respond.ErrorResponse
// @Router       /auth/me [patch]
func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromCtx(r.Context())

	var body struct {
		Name       *string `json:"name"`
		SurName    *string `json:"surName"`
		Patronymic *string `json:"patronymic"`
		AvatarURL  *string `json:"avatarUrl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		respond.Error(w, http.StatusBadRequest, "invalid body")
		return
	}

	user, err := h.svc.UpdateProfile(r.Context(), userID, UpdateProfileInput{
		Name:       body.Name,
		SurName:    body.SurName,
		Patronymic: body.Patronymic,
		AvatarURL:  body.AvatarURL,
	})
	if err != nil {
		code, msg := mapDomainError(err)
		respond.Error(w, code, msg)
		return
	}

	respond.OK(w, user)
}

// Для сваггера
type updateMeRequest struct {
	Name       *string `json:"name"`
	SurName    *string `json:"surName"`
	Patronymic *string `json:"patronymic"`
	AvatarURL  *string `json:"avatarUrl"`
}
