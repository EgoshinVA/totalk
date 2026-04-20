package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	JWTSecret string
}

func Load() *Config {
	_ = godotenv.Load() // Загружаем .env если он есть

	return &Config{
		JWTSecret: getEnv("JWT_SECRET", "super-secret-fallback"),
	}
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
