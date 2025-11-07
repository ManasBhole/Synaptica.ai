package deid

import (
	"context"
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&TokenRecord{})
}

func (r *Repository) Save(ctx context.Context, token, value string) error {
	record := TokenRecord{
		Token:     token,
		Value:     value,
		CreatedAt: time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Save(&record).Error
}

func (r *Repository) Lookup(ctx context.Context, token string) (string, error) {
	var record TokenRecord
	if err := r.db.WithContext(ctx).First(&record, "token = ?", token).Error; err != nil {
		return "", err
	}
	return record.Value, nil
}
