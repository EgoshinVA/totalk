package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"totalk/internal/domain"
	"totalk/pkg/jwt"

	"golang.org/x/crypto/bcrypt"
)

const (
	accessTTL  = 15 * time.Minute
	refreshTTL = 7 * 24 * time.Hour
)

type Service struct {
	users domain.UserRepository
	cache domain.TokenRepository
	reg   domain.RegistrationTokenRepository
	tm    *jwt.TokenManager
}

func NewService(
	users domain.UserRepository,
	cache domain.TokenRepository,
	reg domain.RegistrationTokenRepository,
	tm *jwt.TokenManager,
) *Service {
	return &Service{users: users, cache: cache, reg: reg, tm: tm}
}

// ── Step 1: email + password ──────────────────────────────────────────────────

// RegisterStep1 хэширует пароль, сохраняет email и хэш в Redis и возвращает
// временный registration_token. Пользователь в БД НЕ создаётся.
func (s *Service) RegisterStep1(ctx context.Context, email, password string) (string, error) {
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return "", domain.ErrUserAlreadyExists
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	token, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}

	data := &domain.RegistrationData{
		Email:        email,
		PasswordHash: string(hash),
	}

	if err := s.reg.SetRegistrationToken(ctx, token, data); err != nil {
		return "", err
	}

	return token, nil
}

// ── Step 2: ФИО + город ──────────────────────────────────────────────────────

type ProfileInput struct {
	FullName   string
	SurName    string
	Patronymic *string
}

func (s *Service) RegisterStep2(ctx context.Context, regToken string, input ProfileInput) error {
	data, err := s.resolveRegData(ctx, regToken)
	if err != nil {
		return err
	}

	data.FullName = input.FullName
	data.SurName = input.SurName
	if input.Patronymic != nil {
		data.Patronymic = *input.Patronymic
	}

	return s.reg.SetRegistrationToken(ctx, regToken, data)
}

// ── Step 3: аватар + финализация ─────────────────────────────────────────────

// RegisterStep3 добавляет аватар, создаёт пользователя в БД, удаляет данные из Redis
// и возвращает AuthResponse.
func (s *Service) RegisterStep3(ctx context.Context, regToken string, avatarURL *string) (*domain.AuthResponse, error) {
	data, err := s.resolveRegData(ctx, regToken)
	if err != nil {
		return nil, err
	}

	if avatarURL != nil {
		data.AvatarURL = *avatarURL
	}

	// Создаём пользователя в БД
	userID, err := s.users.CreateUser(ctx, data.Email, data.PasswordHash)
	if err != nil {
		return nil, err
	}

	// Заполняем профиль
	profileFields := map[string]any{
		"name":     data.FullName,
		"sur_name": data.SurName,
	}
	if data.Patronymic != "" {
		profileFields["patronymic"] = data.Patronymic
	}
	if data.AvatarURL != "" {
		profileFields["avatar_url"] = data.AvatarURL
	}

	if err := s.users.UpdateProfile(ctx, userID, profileFields); err != nil {
		// Если обновление профиля упало — пользователь уже создан.
		// В реальном проекте стоит подумать о компенсации или идемпотентности.
		return nil, err
	}

	// Удаляем регистрационные данные из Redis
	_ = s.reg.DeleteRegistrationToken(ctx, regToken)

	return s.issueAuth(ctx, userID)
}

// ── Login ─────────────────────────────────────────────────────────────────────

func (s *Service) Login(ctx context.Context, email, password string) (*domain.AuthResponse, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, domain.ErrInvalidCredentials
	}

	return s.issueAuth(ctx, user.ID)
}

// ── Update profile ────────────────────────────────────────────────────────────────────────

type UpdateProfileInput struct {
	Name       *string
	SurName    *string
	Patronymic *string
	AvatarURL  *string
}

func (s *Service) UpdateProfile(ctx context.Context, userID int64, input UpdateProfileInput) (domain.UserResponse, error) {
	fields := map[string]any{}
	if input.Name != nil {
		fields["name"] = *input.Name
	}
	if input.SurName != nil {
		fields["sur_name"] = *input.SurName
	}
	if input.Patronymic != nil {
		fields["patronymic"] = *input.Patronymic
	}
	if input.AvatarURL != nil {
		fields["avatar_url"] = *input.AvatarURL
	}

	if len(fields) > 0 {
		if err := s.users.UpdateProfile(ctx, userID, fields); err != nil {
			return domain.UserResponse{}, err
		}
	}

	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return domain.UserResponse{}, err
	}
	return domain.UserFromModel(user), nil
}

// ── Me ────────────────────────────────────────────────────────────────────────

func (s *Service) Me(ctx context.Context, userID int64) (*domain.User, error) {
	return s.users.GetByID(ctx, userID)
}

// ── Refresh ───────────────────────────────────────────────────────────────────

// Refresh валидирует refresh-токен, проверяет совпадение с Redis, ротирует пару.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (*domain.TokenPair, error) {
	claims, err := s.tm.ParseToken(refreshToken)
	if err != nil {
		return nil, domain.ErrTokenInvalid
	}

	stored, err := s.cache.GetRefreshToken(ctx, claims.UserID)
	if err != nil || stored != refreshToken {
		return nil, domain.ErrTokenNotFound
	}

	access, err := s.tm.GenerateToken(claims.UserID, accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tm.GenerateToken(claims.UserID, refreshTTL)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetRefreshToken(ctx, claims.UserID, refresh); err != nil {
		return nil, err
	}

	return &domain.TokenPair{AccessToken: access, RefreshToken: refresh}, nil
}

// ── Logout ────────────────────────────────────────────────────────────────────

func (s *Service) Logout(ctx context.Context, userID int64) error {
	return s.cache.DeleteRefreshToken(ctx, userID)
}

// ── private helpers ───────────────────────────────────────────────────────────

// issueAuth генерирует токены и собирает AuthResponse.
func (s *Service) issueAuth(ctx context.Context, userID int64) (*domain.AuthResponse, error) {
	user, err := s.users.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	access, err := s.tm.GenerateToken(userID, accessTTL)
	if err != nil {
		return nil, err
	}
	refresh, err := s.tm.GenerateToken(userID, refreshTTL)
	if err != nil {
		return nil, err
	}

	if err := s.cache.SetRefreshToken(ctx, userID, refresh); err != nil {
		return nil, err
	}

	return &domain.AuthResponse{
		TokenPair: domain.TokenPair{AccessToken: access, RefreshToken: refresh},
		User:      domain.UserFromModel(user),
	}, nil
}

// resolveRegData достаёт RegistrationData из Redis по registration_token.
func (s *Service) resolveRegData(ctx context.Context, token string) (*domain.RegistrationData, error) {
	if token == "" {
		return nil, domain.ErrTokenInvalid
	}
	data, err := s.reg.GetRegistrationToken(ctx, token)
	if err != nil {
		return nil, err
	}
	if data == nil {
		return nil, domain.ErrTokenInvalid
	}
	return data, nil
}

// generateOpaqueToken возвращает криптостойкий hex-токен (32 байта = 64 символа).
func generateOpaqueToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", errors.New("failed to generate token")
	}
	return hex.EncodeToString(b), nil
}
