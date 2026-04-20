package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"totalk/internal/platform/config"
	"totalk/pkg/jwt"

	"totalk/internal/auth"
	"totalk/internal/domain"
	"totalk/internal/middleware"
	"totalk/internal/platform/cache"
	"totalk/internal/platform/database"

	"github.com/go-chi/chi/v5"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func main() {
	cfg := config.Load()

	// Общий контекст для старта (если за 10 сек не поднялись — падаем)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		wg          sync.WaitGroup
		redisClient *redis.Client
		dbConn      *gorm.DB // Явно объявляем тут
	)

	wg.Add(2)

	// 1. Параллельное подключение к БД
	go func() {
		defer wg.Done()
		var err error
		dbConn, err = database.Init(ctx) // database.Init должен возвращать (*gorm.DB, error)
		if err != nil {
			log.Fatalf("❌ DB failed: %v", err)
		}

		// Автомиграция — это удобно для монолита
		dbConn.AutoMigrate(&domain.User{})
		fmt.Println("✅ Postgres connected")
	}()

	// 2. Параллельное подключение к Redis
	go func() {
		defer wg.Done()
		var err error
		redisClient, err = cache.Init(ctx) // cache.Init должен возвращать (*redis.Client, error)
		if err != nil {
			log.Fatalf("❌ Redis failed: %v", err)
		}
		fmt.Println("✅ Redis connected")
	}()

	wg.Wait()
	fmt.Println("🚀 All systems GO. Starting HTTP server...")

	// --- Инициализация слоев (Dependency Injection) ---

	tm := jwt.NewTokenManager(cfg.JWTSecret)

	// Стор для лимитера
	redisStore := cache.NewRedisStore(redisClient)

	// Собираем Auth модуль
	// Передаем dbConn напрямую в репозиторий
	authRepo := auth.NewRepository(dbConn)

	// В сервис передаем и репо (для БД) и redisClient (для токенов)
	authService := auth.NewService(authRepo, redisStore, tm)

	// В хендлер передаем сервис
	authHandler := auth.NewHandler(authService)

	r := chi.NewRouter()

	// Глобальные мидлвары
	r.Use(middleware.RateLimiter(redisStore, 50, time.Minute))

	// Роуты
	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, authHandler)
	})

	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}

	fmt.Println("🌍 Server running on http://localhost:8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
