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
	Patronymic   *string        `gorm:"default:null"`
	AvatarURL    *string        `gorm:"default:null"`
	CreatedAt    time.Time      `gorm:"autoCreateTime"`
	UpdatedAt    time.Time      `gorm:"autoUpdateTime"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`

	Tasks []Task `gorm:"foreignKey:UserID"`
}
