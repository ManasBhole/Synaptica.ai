package ingestion

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"
)

var ErrNotFound = errors.New("ingestion record not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&Record{})
}

func (r *Repository) Create(ctx context.Context, rec *Record) error {
	rec.CreatedAt = time.Now().UTC()
	rec.UpdatedAt = rec.CreatedAt
	return r.db.WithContext(ctx).Create(rec).Error
}

func (r *Repository) UpdateStatus(ctx context.Context, id, status, errMsg string) error {
	return r.db.WithContext(ctx).Model(&Record{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       status,
			"error":        errMsg,
			"updated_at":   time.Now().UTC(),
			"last_attempt": time.Now().UTC(),
		}).Error
}

func (r *Repository) IncrementRetry(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Model(&Record{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"retry_count": gorm.Expr("retry_count + 1"),
			"updated_at":  time.Now().UTC(),
		}).Error
}

func (r *Repository) Get(ctx context.Context, id string) (*Record, error) {
	var rec Record
	result := r.db.WithContext(ctx).First(&rec, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &rec, result.Error
}

func (r *Repository) CleanupExpired(ctx context.Context, ttl time.Duration) error {
	if ttl <= 0 {
		return nil
	}
	cutoff := time.Now().UTC().Add(-ttl)
	return r.db.WithContext(ctx).Where("created_at < ?", cutoff).Delete(&Record{}).Error
}
