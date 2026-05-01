package domain

import (
	"time"

	"gorm.io/gorm"
)

// TaskStatus — тип-перечисление для статуса задачи
type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusDone     TaskStatus = "done"
	TaskStatusCanceled TaskStatus = "canceled"
)

type Task struct {
	ID            int64          `gorm:"primaryKey" json:"id"`
	UserID        int64          `gorm:"index;not null" json:"userId"`
	Title         string         `gorm:"not null" json:"title"`
	Description   string         `json:"description"`
	RawText       string         `json:"rawText"`
	ScheduledAt   *time.Time     `json:"scheduledAt"`
	IsRecurring   bool           `gorm:"default:false" json:"isRecurring"`
	RecurringRule string         `json:"recurringRule"`
	RecurringEnd  *time.Time     `json:"recurringEnd"`
	Status        TaskStatus     `gorm:"default:'pending'" json:"status"`
	CompletedAt   *time.Time     `json:"completedAt"`
	CreatedAt     time.Time      `json:"createdAt"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"-"`
}
