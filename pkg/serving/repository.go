package serving

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// PredictionLog is the persistence model for serving analytics.
type PredictionLog struct {
	ID         uuid.UUID         `gorm:"primaryKey;column:id"`
	PatientID  string            `gorm:"column:patient_id"`
	ModelName  string            `gorm:"column:model_name"`
	Request    datatypes.JSONMap `gorm:"column:request"`
	Response   datatypes.JSONMap `gorm:"column:response"`
	LatencyMs  float64           `gorm:"column:latency_ms"`
	Confidence float64           `gorm:"column:confidence"`
	CreatedAt  time.Time         `gorm:"column:created_at"`
}

// TableName overrides gorm naming.
func (PredictionLog) TableName() string {
	return "prediction_logs"
}

// Repository handles prediction logs queries.
type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&PredictionLog{})
}

func (r *Repository) RecordPrediction(ctx context.Context, req models.PredictionRequest, features map[string]interface{}, resp models.PredictionResponse) error {
	log := PredictionLog{
		ID:         uuid.New(),
		PatientID:  req.PatientID,
		ModelName:  req.ModelName,
		Request:    datatypes.JSONMap(features),
		Response:   datatypes.JSONMap(resp.Predictions),
		LatencyMs:  float64(resp.Latency.Microseconds()) / 1000.0,
		Confidence: resp.Confidence,
		CreatedAt:  time.Now().UTC(),
	}
	return r.db.WithContext(ctx).Create(&log).Error
}

// Recent returns the most recent prediction logs up to limit.
func (r *Repository) Recent(ctx context.Context, limit int) ([]PredictionLog, error) {
	if limit <= 0 {
		limit = 50
	}
	var logs []PredictionLog
	err := r.db.WithContext(ctx).
		Order("created_at DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}
