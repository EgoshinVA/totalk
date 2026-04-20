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
	ID            int64  `gorm:"primaryKey"`
	UserID        int64  `gorm:"index;not null"`
	Title         string `gorm:"not null"`
	Description   string
	RawText       string
	ScheduledAt   *time.Time     `gorm:"index:idx_status_scheduled,priority:2"` // время напоминания/планирования
	IsRecurring   bool           `gorm:"default:false"`
	RecurringRule string         // например "daily", "weekly", "monthly" или cron-строка
	RecurringEnd  *time.Time     // дата окончания повторений (если nil — бесконечно)
	Status        TaskStatus     `gorm:"default:'pending';index:idx_status_scheduled,priority:1"`
	CompletedAt   *time.Time     // когда задача реально выполнена
	CreatedAt     time.Time      `gorm:"autoCreateTime"`
	UpdatedAt     time.Time      `gorm:"autoUpdateTime"`
	DeletedAt     gorm.DeletedAt `gorm:"index"` // мягкое удаление
}
