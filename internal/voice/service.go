package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

type AIService struct {
}

func NewAIService() *AIService {
	return &AIService{}
}

type TaskExtraction struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ScheduledAt string `json:"scheduledAt"`
	IsRecurring bool   `json:"isRecurring"`
}

func (s *AIService) Transcribe(audioData []byte) (string, error) {
	tempFile := "input_temp.wav"
	wavData := addWavHeader(audioData, 16000)

	if err := os.WriteFile(tempFile, wavData, 0644); err != nil {
		return "", err
	}

	defer os.Remove(tempFile)

	// Используем полные или чистые относительные пути
	cmd := exec.Command("bin/whisper-cli.exe",
		"-m", "bin/ggml-small.bin",
		"-f", tempFile,
		"-nt",
		"-l", "ru",
	)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr // Это критично!

	if err := cmd.Run(); err != nil {
		// Если упало, выводим что именно сказал Whisper перед смертью
		fmt.Printf("Whisper Error: %s\n", stderr.String())
		return "", fmt.Errorf("whisper error: %v, details: %s", err, stderr.String())
	}

	return strings.TrimSpace(out.String()), nil
}

func addWavHeader(pcmData []byte, sampleRate int) []byte {
	dataLen := len(pcmData)
	header := make([]byte, 44)

	copy(header[0:4], "RIFF")
	fileSize := uint32(dataLen + 36)
	header[4] = byte(fileSize)
	header[5] = byte(fileSize >> 8)
	header[6] = byte(fileSize >> 16)
	header[7] = byte(fileSize >> 24)

	copy(header[8:12], "WAVE")
	copy(header[12:16], "fmt ")
	header[16] = 16 // Subchunk1Size
	header[20] = 1  // PCM format
	header[22] = 1  // Mono

	header[24] = byte(sampleRate)
	header[25] = byte(sampleRate >> 8)
	header[26] = byte(sampleRate >> 16)
	header[27] = byte(sampleRate >> 24)

	byteRate := uint32(sampleRate * 1 * 2)
	header[28] = byte(byteRate)
	header[29] = byte(byteRate >> 8)
	header[30] = byte(byteRate >> 16)
	header[31] = byte(byteRate >> 24)

	header[32] = 2  // BlockAlign
	header[34] = 16 // BitsPerSample
	copy(header[36:40], "data")
	header[40] = byte(dataLen)
	header[41] = byte(dataLen >> 8)
	header[42] = byte(dataLen >> 16)
	header[43] = byte(dataLen >> 24)

	return append(header, pcmData...)
}

func (s *AIService) ExtractTask(rawText string) (*TaskExtraction, error) {
	url := "http://localhost:1234/v1/chat/completions"

	nowDate := time.Now().Format("2006-01-02")
	nowTime := time.Now().Format("15:04")
	nowDay := time.Now().Format("Monday")

	prompt := fmt.Sprintf(
		"Сегодня %s (%s), время %s.\nТекст: \"%s\"\n\nВерни JSON. Только JSON, без пояснений.",
		nowDate, nowDay, nowTime, rawText,
	)

	requestBody := map[string]interface{}{
		"messages": []map[string]string{
			{
				"role": "system",
				"content": `Ты — экспертный парсер времени. Твоя задача: извлечь действие и вычислить ТОЧНОЕ время.
Текущая точка отсчета (сегодня и сейчас) передается пользователем.

ПРАВИЛА ВЫЧИСЛЕНИЙ:
1. "через X минут/часов": ПРИБАВЬ X к текущему времени. Округлять ЗАПРЕЩЕНО. Если сейчас 16:41 и просят "через 30 минут", должно быть 17:11.
2. "каждый день в HH:mm": поставь ближайшее время (сегодня, если еще не наступило, или завтра).
3. Формат scheduledAt: YYYY-MM-DDTHH:mm:ss.
4. Title: Всегда инфинитив (Что сделать?).

ОТВЕЧАЙ ТОЛЬКО JSON: {"title": string, "description": string, "scheduledAt": string, "isRecurring": boolean}`,
			},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.0,
	}

	jsonData, _ := json.Marshal(requestBody)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to LM Studio: %v", err)
	}
	defer resp.Body.Close()

	var apiResponse struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode AI response: %v", err)
	}

	if len(apiResponse.Choices) == 0 {
		return nil, fmt.Errorf("AI returned empty result")
	}

	content := apiResponse.Choices[0].Message.Content
	fmt.Printf("🤖 LM response: %s\n", content)

	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimSpace(content)

	var task TaskExtraction
	if err := json.Unmarshal([]byte(content), &task); err != nil {
		return nil, fmt.Errorf("failed to parse task JSON: %v, content: %s", err, content)
	}

	return &task, nil
}
