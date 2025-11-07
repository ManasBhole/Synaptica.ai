package ingestion

import (
	"time"

	"gorm.io/datatypes"
)

const (
	StatusAccepted  = "accepted"
	StatusPublished = "published"
	StatusFailed    = "failed"
)

type Record struct {
	ID          string            `json:"id" gorm:"primaryKey;column:id"`
	Source      string            `json:"source" gorm:"column:source"`
	Format      string            `json:"format" gorm:"column:format"`
	Payload     datatypes.JSONMap `json:"payload" gorm:"column:payload"`
	Status      string            `json:"status" gorm:"column:status"`
	Error       string            `json:"error,omitempty" gorm:"column:error"`
	CreatedAt   time.Time         `json:"created_at" gorm:"column:created_at"`
	UpdatedAt   time.Time         `json:"updated_at" gorm:"column:updated_at"`
	RetryCount  int               `json:"retry_count" gorm:"column:retry_count"`
	LastAttempt *time.Time        `json:"last_attempt,omitempty" gorm:"column:last_attempt"`
}

func (Record) TableName() string {
	return "ingestion_requests"
}
