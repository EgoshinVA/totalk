package domain

import (
	"context"
	"time"
)

type RegistrationData struct {
	Email        string `json:"email"`
	PasswordHash string `json:"password_hash"`
	FullName     string `json:"full_name,omitempty"`
	SurName      string `json:"sur_name,omitempty"`
	Patronymic   string `json:"patronymic,omitempty"`
	AvatarURL    string `json:"avatar_url,omitempty"`
}

// UserRepository — доступ к таблице users.
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
	CreateUser(ctx context.Context, email, passwordHash string) (int64, error)
	UpdateProfile(ctx context.Context, id int64, fields map[string]any) error
}

// TokenRepository — refresh-токены в Redis.
type TokenRepository interface {
	SetRefreshToken(ctx context.Context, userID int64, token string) error
	GetRefreshToken(ctx context.Context, userID int64) (string, error)
	DeleteRefreshToken(ctx context.Context, userID int64) error
}

// RegistrationTokenRepository — временные токены регистрации.
type RegistrationTokenRepository interface {
	SetRegistrationToken(ctx context.Context, token string, data *RegistrationData) error
	GetRegistrationToken(ctx context.Context, token string) (*RegistrationData, error)
	DeleteRegistrationToken(ctx context.Context, token string) error
}

// RateLimiterStore — используется middleware.
type RateLimiterStore interface {
	Allow(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
}
