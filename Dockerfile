# ========== Этап сборки ==========
FROM golang:1.25-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git make g++ cmake curl bash

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Создаём директорию для бинарников
RUN mkdir -p /app/bin

# Собираем whisper-cli статически
RUN git clone --depth 1 https://github.com/ggerganov/whisper.cpp /tmp/whisper && \
    cd /tmp/whisper && \
    cmake -B build -DBUILD_SHARED_LIBS=OFF -DGGML_STATIC=ON && \
    cmake --build build -j$(nproc) --config Release && \
    cp build/bin/whisper-cli /app/bin/whisper-cli

# Копируем модель
RUN CGO_ENABLED=0 GOOS=linux go build -o server ./cmd/main.go

# ========== Финальный образ ==========
FROM alpine:3.19

WORKDIR /app

# Зависимости для запуска
RUN apk add --no-cache libstdc++ libgcc ffmpeg

COPY --from=builder /app/server .
COPY --from=builder /app/bin ./bin

EXPOSE 8080

CMD ["./server"]