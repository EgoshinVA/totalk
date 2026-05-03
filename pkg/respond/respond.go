package respond

import (
	"encoding/json"
	"net/http"
)

// ErrorResponse — структура ошибки для swagger документации
type ErrorResponse struct {
	Message string `json:"message"`
}

// JSON пишет произвольный объект как JSON с нужным статусом.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// Error пишет {"message": msg} с нужным статусом.
func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, map[string]string{"message": msg})
}

// OK пишет 200 с телом.
func OK(w http.ResponseWriter, v any) {
	JSON(w, http.StatusOK, v)
}

// Created пишет 201 с телом.
func Created(w http.ResponseWriter, v any) {
	JSON(w, http.StatusCreated, v)
}
