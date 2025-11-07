package ingestion

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
)

type Service struct {
	validator *Validator
	repo      *Repository
	producer  *kafka.Producer
	dlq       *kafka.Producer
	statusTTL time.Duration
}

func NewService(validator *Validator, repo *Repository, producer *kafka.Producer, dlq *kafka.Producer, ttl time.Duration) *Service {
	return &Service{
		validator: validator,
		repo:      repo,
		producer:  producer,
		dlq:       dlq,
		statusTTL: ttl,
	}
}

func (s *Service) Process(ctx context.Context, req models.IngestRequest) (*models.IngestResponse, error) {
	if err := s.validator.Validate(req); err != nil {
		return nil, err
	}

	id := uuid.New().String()
	record := &Record{
		ID:         id,
		Source:     req.Source,
		Format:     req.Format,
		Payload:    datatypes.JSONMap(req.Data),
		Status:     StatusAccepted,
		RetryCount: 0,
	}

	if err := s.repo.Create(ctx, record); err != nil {
		return nil, fmt.Errorf("persisting ingestion record: %w", err)
	}

	payload := map[string]interface{}{
		"ingest_id":   id,
		"source":      req.Source,
		"format":      req.Format,
		"data":        req.Data,
		"patient_id":  req.PatientID,
		"metadata":    req.Metadata,
		"received_at": time.Now().UTC(),
	}

	sendErr := s.producer.PublishEvent(ctx, "upstream", req.Source, payload)
	if sendErr != nil {
		logger.Log.WithError(sendErr).Error("failed to publish ingestion event")
		_ = s.repo.UpdateStatus(ctx, id, StatusFailed, sendErr.Error())
		if s.dlq != nil {
			dlqErr := s.dlq.PublishEvent(ctx, "ingestion-dlq", req.Source, payload)
			if dlqErr != nil {
				logger.Log.WithError(dlqErr).Error("failed to push event to DLQ")
			}
		}
		return nil, fmt.Errorf("publishing event: %w", sendErr)
	}

	_ = s.repo.UpdateStatus(ctx, id, StatusPublished, "")

	resp := &models.IngestResponse{
		ID:        id,
		Status:    StatusPublished,
		Timestamp: time.Now().UTC(),
	}

	return resp, nil
}

func (s *Service) Status(ctx context.Context, id string) (*Record, error) {
	rec, err := s.repo.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return rec, nil
}

func (s *Service) Cleanup(ctx context.Context) error {
	return s.repo.CleanupExpired(ctx, s.statusTTL)
}
