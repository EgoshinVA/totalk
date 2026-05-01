package tasks

import (
	"context"
	"totalk/internal/domain"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, task *domain.Task) error {
	return r.db.WithContext(ctx).Create(task).Error
}

func (r *Repository) ListByUser(ctx context.Context, userID int64, status string) ([]domain.Task, error) {
	var tasks []domain.Task
	q := r.db.WithContext(ctx).
		Where("user_id = ? AND deleted_at IS NULL", userID)
	if status != "" {
		q = q.Where("status = ?", status)
	}
	err := q.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (r *Repository) GetByID(ctx context.Context, id, userID int64) (*domain.Task, error) {
	var task domain.Task
	err := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ? AND deleted_at IS NULL", id, userID).
		First(&task).Error
	if err == gorm.ErrRecordNotFound {
		return nil, domain.ErrTaskNotFound
	}
	return &task, err
}

func (r *Repository) Update(ctx context.Context, task *domain.Task) error {
	return r.db.WithContext(ctx).Save(task).Error
}

func (r *Repository) SoftDelete(ctx context.Context, id, userID int64) error {
	res := r.db.WithContext(ctx).
		Where("id = ? AND user_id = ?", id, userID).
		Delete(&domain.Task{})
	if res.RowsAffected == 0 {
		return domain.ErrTaskNotFound
	}
	return res.Error
}
