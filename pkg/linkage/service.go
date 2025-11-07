package linkage

import (
	"context"
	"fmt"

	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
)

type Service struct {
	repo     *Repository
	matcher  *Matcher
	producer *kafka.Producer
	dlq      *kafka.Producer
	topic    string
}

func NewService(repo *Repository, matcher *Matcher, producer, dlq *kafka.Producer, topic string) *Service {
	return &Service{repo: repo, matcher: matcher, producer: producer, dlq: dlq, topic: topic}
}

func (s *Service) Process(ctx context.Context, record *models.NormalizedRecord) (*models.LinkageResult, error) {
	if record == nil {
		return nil, fmt.Errorf("nil record")
	}
	normalized := map[string]interface{}{
		"patient_id":    record.PatientID,
		"resource_type": record.ResourceType,
		"canonical":     record.Canonical,
	}

	dKey := s.matcher.DeterministicKey(normalized)
	var masterID string
	var err error

	if dKey != "" {
		masterID, err = s.repo.FindMasterByDeterministicKey(ctx, dKey)
		if err != nil && err != ErrNoMatch {
			return nil, err
		}
	}

	if masterID == "" {
		masterID, err = s.repo.CreateMaster(ctx)
		if err != nil {
			return nil, err
		}
	}

	method := "deterministic"
	score := 1.0

	if dKey == "" {
		candidates, err := s.repo.RecentLinks(ctx, 100)
		if err != nil {
			return nil, err
		}
		match := s.matcher.Probabilistic(candidates, normalized)
		method = match.Method
		score = match.Score
		if match.Method == "new" {
			if newMaster, err := s.repo.CreateMaster(ctx); err == nil {
				masterID = newMaster
				score = 1.0
				method = "deterministic"
			} else {
				logger.Log.WithError(err).Warn("failed to create master for new match")
			}
		} else {
			masterID = match.MasterID
		}
	}

	link := &PatientLink{
		MasterID:         masterID,
		PatientID:        record.PatientID,
		DeterministicKey: dKey,
		Score:            score,
		Method:           method,
		Attributes:       datatypes.JSONMap(record.Canonical),
	}
	if err := s.repo.SaveLink(ctx, link); err != nil {
		return nil, err
	}

	result := &models.LinkageResult{
		MasterPatientID: masterID,
		LinkedIDs:       []string{record.PatientID},
		Confidence:      score,
		Method:          method,
	}

	payload := map[string]interface{}{
		"linkage":   result,
		"canonical": record,
	}

	if err := s.producer.PublishEvent(ctx, "link", "linkage-service", payload); err != nil {
		logger.Log.WithError(err).Error("failed to publish linkage event")
		if s.dlq != nil {
			_ = s.dlq.PublishEvent(ctx, "link", "linkage-service", payload)
		}
	}

	return result, nil
}
