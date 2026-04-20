package database

import (
	"context"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var DB *gorm.DB

func Init(ctx context.Context) (*gorm.DB, error) {
	dsn := "host=localhost user=user password=pass dbname=totalk port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	DB = db
	return db, nil
}
