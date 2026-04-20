package domain

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           int64  `gorm:"primaryKey"`
	Email        string `gorm:"uniqueIndex;not null"`
	PasswordHash string `gorm:"not null"`
	SurName      string
	Name         string
	Patronymic   *string        // может быть NULL
	AvatarURL    *string        // может быть NULL
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"` // мягкое удаление

	Tasks []Task `gorm:"foreignKey:UserID"` // явное указание внешнего ключа
}
