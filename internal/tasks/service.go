package tasks

import (
	"context"
	"time"
	"totalk/internal/domain"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

type CreateInput struct {
	UserID      int64
	Title       string
	Description string
	RawText     string
	ScheduledAt *time.Time
	IsRecurring bool
}

type UpdateInput struct {
	Title       *string
	Description *string
	ScheduledAt *time.Time
	Status      *domain.TaskStatus
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*domain.Task, error) {
	task := &domain.Task{
		UserID:      in.UserID,
		Title:       in.Title,
		Description: in.Description,
		RawText:     in.RawText,
		ScheduledAt: in.ScheduledAt,
		IsRecurring: in.IsRecurring,
		Status:      domain.TaskStatusPending,
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return task, nil
}

func (s *Service) List(ctx context.Context, userID int64, status string) ([]domain.Task, error) {
	return s.repo.ListByUser(ctx, userID, status)
}

func (s *Service) Complete(ctx context.Context, id, userID int64) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	task.Status = domain.TaskStatusDone
	task.CompletedAt = &now
	return task, s.repo.Update(ctx, task)
}

func (s *Service) Update(ctx context.Context, id, userID int64, in UpdateInput) (*domain.Task, error) {
	task, err := s.repo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, err
	}
	if in.Title != nil {
		task.Title = *in.Title
	}
	if in.Description != nil {
		task.Description = *in.Description
	}
	if in.ScheduledAt != nil {
		task.ScheduledAt = in.ScheduledAt
	}
	if in.Status != nil {
		task.Status = *in.Status
	}
	return task, s.repo.Update(ctx, task)
}

func (s *Service) Delete(ctx context.Context, id, userID int64) error {
	return s.repo.SoftDelete(ctx, id, userID)
}
