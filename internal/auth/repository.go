package auth

import (
	"context"
	"gorm.io/gorm"
	"totalk/internal/domain"
)

type repo struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) UserRepository {
	return &repo{db: db}
}

func (r *repo) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	var user domain.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	return &user, err
}

func (r *repo) CreateUser(ctx context.Context, email, password string) error {
	user := domain.User{Email: email, PasswordHash: password}
	return r.db.WithContext(ctx).Create(&user).Error
}
