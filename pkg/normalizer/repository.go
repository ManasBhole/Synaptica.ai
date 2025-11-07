package normalizer

import (
	"context"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type RecordModel struct {
	ID           string            `gorm:"primaryKey;column:id"`
	PatientID    string            `gorm:"column:patient_id"`
	ResourceType string            `gorm:"column:resource_type"`
	Canonical    datatypes.JSONMap `gorm:"column:canonical"`
	Codes        datatypes.JSONMap `gorm:"column:codes"`
	Timestamp    time.Time         `gorm:"column:timestamp"`
	CreatedAt    time.Time         `gorm:"column:created_at"`
}

func (RecordModel) TableName() string {
	return "normalized_records"
}

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&RecordModel{})
}

func (r *Repository) Save(ctx context.Context, rec *RecordModel) error {
	rec.CreatedAt = time.Now().UTC()
	return r.db.WithContext(ctx).Create(rec).Error
}
