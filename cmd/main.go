package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
	"totalk/internal/tasks"
	"totalk/internal/voice"

	"github.com/go-chi/chi/v5"
	"gorm.io/gorm"

	"totalk/internal/auth"
	"totalk/internal/domain"
	"totalk/internal/middleware"
	"totalk/internal/platform/cache"
	"totalk/internal/platform/config"
	"totalk/internal/platform/database"
	"totalk/pkg/jwt"

	_ "totalk/docs"

	httpSwagger "github.com/swaggo/http-swagger"
)

// @title ToTalk API
// @version 1.0
// @description Voice-to-task API
// @host localhost:8080
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var (
		wg         sync.WaitGroup
		redisStore *cache.RedisStore
		dbConn     *gorm.DB
	)

	wg.Add(2)

	go func() {
		defer wg.Done()
		var err error
		dbConn, err = database.Init(ctx)
		if err != nil {
			log.Fatalf("❌ DB failed: %v", err)
		}
		if err := dbConn.AutoMigrate(&domain.User{}, &domain.Task{}); err != nil {
			log.Fatalf("❌ AutoMigrate failed: %v", err)
		}
		fmt.Println("✅ Postgres connected")
	}()

	go func() {
		defer wg.Done()
		client, err := cache.Init(ctx)
		if err != nil {
			log.Fatalf("❌ Redis failed: %v", err)
		}
		redisStore = cache.NewRedisStore(client)
		fmt.Println("✅ Redis connected")
	}()

	wg.Wait()
	fmt.Println("🚀 All systems GO. Starting HTTP server...")

	// ── Dependency injection ─────────────────────────────────────────────────

	tm := jwt.NewTokenManager(cfg.JWTSecret)

	userRepo := auth.NewRepository(dbConn)
	authService := auth.NewService(userRepo, redisStore, redisStore, tm)
	authHandler := auth.NewHandler(authService)

	taskRepo := tasks.NewRepository(dbConn)
	taskService := tasks.NewService(taskRepo)
	taskHandler := tasks.NewHandler(taskService)

	aiService := voice.NewAIService()

	voiceHandler := voice.NewHandler(aiService, taskService, tm)

	// ── Router ───────────────────────────────────────────────────────────────

	r := chi.NewRouter()
	r.Use(middleware.RateLimiter(redisStore, 100, time.Minute))

	r.Route("/api/v1", func(r chi.Router) {
		auth.RegisterRoutes(r, authHandler, tm)
		tasks.RegisterRoutes(r, taskHandler, tm)
		r.HandleFunc("/ws/audio", voiceHandler.HandleWS)
	})

	r.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// ── Server ───────────────────────────────────────────────────────────────

	srv := &http.Server{
		Addr:         "0.0.0.0:8080",
		Handler:      r,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	fmt.Println("🌍 Server running on http://localhost:8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal(err)
	}
}
