package voice

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"
	"totalk/internal/tasks"
	"totalk/pkg/jwt"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

type Handler struct {
	aiService   *AIService
	taskService *tasks.Service
	tm          *jwt.TokenManager
}

func NewHandler(aiService *AIService, taskService *tasks.Service, tm *jwt.TokenManager) *Handler {
	return &Handler{aiService: aiService, taskService: taskService, tm: tm}
}

type wsResponse struct {
	Type    string      `json:"type"` // "task_created" | "error"
	Payload interface{} `json:"payload"`
}

func (h *Handler) HandleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	if token == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	claims, err := h.tm.ParseToken(token)
	if err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	userID := claims.UserID

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	var audioBuffer []byte
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		// Сигнал конца записи
		if string(message) == "END_OF_STREAM" {
			break
		}

		decoded, _ := base64.StdEncoding.DecodeString(string(message))
		audioBuffer = append(audioBuffer, decoded...)
	}

	if len(audioBuffer) == 0 {
		return
	}

	sendError := func(msg string) {
		_ = conn.WriteJSON(wsResponse{Type: "error", Payload: map[string]string{"message": msg}})
	}

	// 1. Whisper
	rawText, err := h.aiService.Transcribe(audioBuffer)
	if err != nil || rawText == "" {
		sendError("Не удалось распознать речь")
		return
	}
	fmt.Printf("🗣️ Услышал: %s\n", rawText)

	// 2. LLM → JSON
	extracted, err := h.aiService.ExtractTask(rawText)
	if err != nil || extracted.Title == "" {
		sendError("Не удалось понять задачу. Попробуй сформулировать чётче.")
		return
	}

	// 3. Сохраняем в БД
	var scheduledAt *time.Time
	if extracted.ScheduledAt != "" {
		// 1. Берем локальную таймзону сервера
		loc := time.Now().Location()

		// 2. Используем ParseInLocation вместо обычного Parse
		t, err := time.ParseInLocation("2006-01-02T15:04:05", extracted.ScheduledAt, loc)

		if err != nil {
			t, err = time.ParseInLocation("2006-01-02T15:04", extracted.ScheduledAt, loc)
		}

		if err == nil {
			scheduledAt = &t
		}
	}

	task, err := h.taskService.Create(context.Background(), tasks.CreateInput{
		UserID:      userID,
		Title:       extracted.Title,
		Description: extracted.Description,
		RawText:     rawText,
		ScheduledAt: scheduledAt,
		IsRecurring: extracted.IsRecurring,
	})
	if err != nil {
		sendError("Ошибка сохранения задачи")
		return
	}

	fmt.Printf("✅ Сохранена задача %v\n", task)

	// 4. Отправляем обратно клиенту
	_ = conn.WriteJSON(wsResponse{Type: "task_created", Payload: task})
}
