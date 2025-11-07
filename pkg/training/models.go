package training

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

const (
	StatusQueued    = "queued"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

type JobModel struct {
	ID           uuid.UUID         `gorm:"type:uuid;primaryKey;column:id"`
	ModelType    string            `gorm:"column:model_type"`
	Config       datatypes.JSONMap `gorm:"column:config"`
	Filters      datatypes.JSONMap `gorm:"column:filters"`
	Status       string            `gorm:"column:status"`
	Metrics      datatypes.JSONMap `gorm:"column:metrics"`
	ArtifactPath string            `gorm:"column:artifact_path"`
	ErrorMessage string            `gorm:"column:error_message"`
	CreatedAt    time.Time         `gorm:"column:created_at"`
	UpdatedAt    time.Time         `gorm:"column:updated_at"`
	StartedAt    *time.Time        `gorm:"column:started_at"`
	CompletedAt  *time.Time        `gorm:"column:completed_at"`
}

func (JobModel) TableName() string {
	return "training_jobs"
}

type CreateJobInput struct {
	ModelType string
	Config    map[string]interface{}
	Filters   map[string]interface{}
}

type Artifact struct {
	JobID   uuid.UUID              `json:"job_id"`
	Path    string                 `json:"path"`
	Metrics map[string]interface{} `json:"metrics"`
}
