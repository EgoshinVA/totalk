package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"
)

// Инициализируем один раз на пакет
var validate = validator.New()

type registerRequest struct {
	// Добавляем теги валидации: обязательное поле, формат email, мин. длина пароля
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
}

type loginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type ApiError struct {
	Message string `json:"message"`
}

type Handler struct {
	service *Service
}

// NewHandler — тот самый конструктор для main.go
func NewHandler(s *Service) *Handler {
	return &Handler{service: s}
}

// Хелпер для JSON-ответов с ошибкой
func sendError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(ApiError{Message: message})
}

func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 1048576)

	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Неверный формат JSON")
		return
	}

	// Нормализация email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	// 1. Автоматическая валидация по тегам
	if err := validate.Struct(req); err != nil {
		sendError(w, http.StatusBadRequest, "Ошибка валидации: проверьте email и длину пароля (мин. 8 симв.)")
		return
	}

	if err := h.service.Register(r.Context(), req.Email, req.Password); err != nil {
		// Здесь можно проверять тип ошибки (например, "user already exists")
		sendError(w, http.StatusInternalServerError, "Пользователь с таким email уже существует")
		return
	}

	// 2. Возвращаем JSON даже при успехе (чтобы фронт не падал при парсинге)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(map[string]string{"message": "User registered successfully"})
}

func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		sendError(w, http.StatusBadRequest, "Неверный формат запроса")
		return
	}

	// Нормализация email
	req.Email = strings.TrimSpace(strings.ToLower(req.Email))

	if err := validate.Struct(req); err != nil {
		sendError(w, http.StatusBadRequest, "Введите корректный email и пароль")
		return
	}

	accessToken, refreshToken, err := h.service.Login(r.Context(), req.Email, req.Password)
	if err != nil {
		// Не говорим точно, что неверно (пароль или email) в целях безопасности
		sendError(w, http.StatusUnauthorized, "Неверный email или пароль")
		return
	}

	resp := map[string]string{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
