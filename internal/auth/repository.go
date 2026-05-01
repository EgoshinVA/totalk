package auth

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"totalk/internal/domain"
)

type repo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) domain.UserRepository {
	return &repo{db: db}
}

func (r *repo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	return &user, err
}

func (r *repo) GetByID(ctx context.Context, id int64) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).First(&user, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, domain.ErrUserNotFound
	}
	return &user, err
}

// CreateUser создаёт запись и возвращает присвоенный ID.
func (r *repo) CreateUser(ctx context.Context, email, passwordHash string) (int64, error) {
	user := domain.User{Email: email, PasswordHash: passwordHash}
	if err := r.db.WithContext(ctx).Create(&user).Error; err != nil {
		return 0, err
	}
	return user.ID, nil
}

// UpdateProfile обновляет только переданные поля (PATCH-семантика через map).
func (r *repo) UpdateProfile(ctx context.Context, id int64, fields map[string]any) error {
	return r.db.WithContext(ctx).
		Model(&domain.User{}).
		Where("id = ?", id).
		Updates(fields).Error
}
