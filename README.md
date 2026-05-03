# ToTalk — Backend

> Voice-to-task API. Converts speech into structured tasks using local AI.

## Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.22 |
| HTTP Router | chi |
| Database | PostgreSQL 15 + GORM |
| Cache | Redis 7 |
| Auth | JWT (HS256) |
| Speech-to-text | Whisper.cpp (local) |
| LLM | Ollama / LM Studio (qwen2.5:3b) |
| Real-time | WebSocket (gorilla/websocket) |
| Docs | Swagger (swaggo) |

## Architecture

```
cmd/
  main.go              — entry point, DI, router
internal/
  auth/                — registration, login, JWT, profile
  tasks/               — CRUD tasks
  voice/               — WebSocket audio handler, Whisper, LLM
  domain/              — entities, errors, interfaces
  middleware/           — auth, rate limiter
  platform/
    config/            — env config
    database/          — postgres init
    cache/             — redis init
pkg/
  jwt/                 — token manager
  respond/             — JSON helpers
docs/                  — swagger generated files
bin/
  whisper-cli          — whisper.cpp binary (linux)
  whisper-cli.exe      — whisper.cpp binary (windows)
  ggml-small.bin       — whisper model
```

## How it works

1. Mobile client connects via **WebSocket** and streams raw PCM audio
2. Server buffers audio chunks until `END_OF_STREAM` signal
3. **Whisper.cpp** transcribes audio to text
4. **LLM (Ollama/qwen2.5:3b)** extracts structured task JSON from text
5. Task is saved to **PostgreSQL** and returned to client
6. Client schedules a **push notification** based on user preferences

## Quick Start

### Prerequisites
- Go 1.22+
- Docker & Docker Compose
- NVIDIA GPU (optional, for Ollama acceleration)

### 1. Clone and configure

```bash
git clone https://github.com/yourname/totalk-backend
cd totalk-backend
cp .env.example .env
```

### 2. Fill `.env`

```env
DB_HOST=localhost
DB_PORT=5432
DB_USER=user
DB_PASSWORD=pass
DB_NAME=totalk

JWT_SECRET=your-secret-key

LLM_URL=http://localhost:11434/v1/chat/completions
```

### 3. Run with Docker Compose

```bash
docker compose up --build
```

This will start:
- PostgreSQL
- Redis
- Ollama (pulls qwen2.5:3b automatically)
- The Go server

### 4. Run locally (without Docker)

```bash
# Start postgres and redis separately
docker compose up postgres redis -d

# Run server
go run cmd/main.go
```

### Whisper model

Download the model and place it in `bin/`:

```bash
curl -L https://huggingface.co/ggerganov/whisper.cpp/resolve/main/ggml-small.bin \
     -o bin/ggml-small.bin
```

## API

Swagger UI available at:
```
http://localhost:8080/swagger/index.html
```

### WebSocket

```
ws://host/api/v1/ws/audio?token=<access_token>
```

Send raw PCM audio chunks (base64), then `END_OF_STREAM` to trigger processing.

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DB_HOST` | PostgreSQL host | `localhost` |
| `DB_PORT` | PostgreSQL port | `5432` |
| `DB_USER` | Database user | — |
| `DB_PASSWORD` | Database password | — |
| `DB_NAME` | Database name | — |
| `JWT_SECRET` | JWT signing secret | — |
| `LLM_URL` | LLM API endpoint | `http://localhost:1234/v1/chat/completions` |