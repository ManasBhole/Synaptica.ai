package training

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

var ErrJobNotFound = errors.New("training job not found")

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) AutoMigrate() error {
	return r.db.AutoMigrate(&JobModel{})
}

func (r *Repository) Create(ctx context.Context, job *JobModel) error {
	return r.db.WithContext(ctx).Create(job).Error
}

func (r *Repository) Update(ctx context.Context, job *JobModel) error {
	return r.db.WithContext(ctx).Save(job).Error
}

func (r *Repository) UpdateStatus(ctx context.Context, jobID uuid.UUID, status string, metrics map[string]interface{}, artifactPath, errorMessage string) error {
	updates := map[string]interface{}{
		"status":        status,
		"artifact_path": artifactPath,
		"error_message": errorMessage,
		"updated_at":    time.Now().UTC(),
	}
	if metrics != nil {
		updates["metrics"] = datatypes.JSONMap(metrics)
	}
	return r.db.WithContext(ctx).Model(&JobModel{}).Where("id = ?", jobID).Updates(updates).Error
}

func (r *Repository) SetTimestamps(ctx context.Context, jobID uuid.UUID, startedAt, completedAt *time.Time) error {
	updates := map[string]interface{}{"updated_at": time.Now().UTC()}
	if startedAt != nil {
		updates["started_at"] = *startedAt
	}
	if completedAt != nil {
		updates["completed_at"] = *completedAt
	}
	return r.db.WithContext(ctx).Model(&JobModel{}).Where("id = ?", jobID).Updates(updates).Error
}

func (r *Repository) Get(ctx context.Context, jobID uuid.UUID) (*JobModel, error) {
	var job JobModel
	result := r.db.WithContext(ctx).First(&job, "id = ?", jobID)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, ErrJobNotFound
	}
	return &job, result.Error
}

func (r *Repository) List(ctx context.Context, limit int) ([]JobModel, error) {
	if limit <= 0 {
		limit = 50
	}
	var jobs []JobModel
	result := r.db.WithContext(ctx).Order("created_at desc").Limit(limit).Find(&jobs)
	return jobs, result.Error
}
