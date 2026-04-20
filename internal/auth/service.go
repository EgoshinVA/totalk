package auth

import (
	"context"
	"errors"
	"time"

	"totalk/internal/domain"

	authjwt "totalk/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

// Описываем интерфейсы прямо тут или в domain
type UserRepository interface {
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	CreateUser(ctx context.Context, email, password string) error
}

type TokenRepository interface {
	SetRefreshToken(ctx context.Context, userID uint, token string) error
}

type Service struct {
	repo  UserRepository
	cache TokenRepository
	tm    *authjwt.TokenManager
}

// Конструктор сервиса
func NewService(repo UserRepository, cache TokenRepository, tm *authjwt.TokenManager) *Service {
	return &Service{repo: repo, cache: cache, tm: tm}
}

func (s *Service) Login(ctx context.Context, email, password string) (string, string, error) {
	user, err := s.repo.GetByEmail(ctx, email)
	if err != nil {
		return "", "", errors.New("user not found")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return "", "", errors.New("invalid password")
	}

	// Генерация токенов
	accessToken, _ := s.tm.GenerateToken(user.ID, 15*time.Minute)
	refreshToken, _ := s.tm.GenerateToken(user.ID, 7*24*time.Hour)

	if err := s.cache.SetRefreshToken(ctx, user.ID, refreshToken); err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) Register(ctx context.Context, email, password string) error {
	// 1. Генерируем хэш (возвращает []byte)
	hashedPasswordBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	// 2. Преобразуем []byte в string прямо при передаче в репозиторий
	// Это важно, так как в БД мы храним хэш как VARCHAR/TEXT
	return s.repo.CreateUser(ctx, email, string(hashedPasswordBytes))
}
