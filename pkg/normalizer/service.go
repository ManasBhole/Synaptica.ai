package normalizer

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
)

type Service struct {
	transformer *Transformer
	repo        *Repository
	producer    *kafka.Producer
	dlq         *kafka.Producer
	topic       string
}

func NewService(transformer *Transformer, repo *Repository, producer *kafka.Producer, dlq *kafka.Producer, topic string) *Service {
	return &Service{
		transformer: transformer,
		repo:        repo,
		producer:    producer,
		dlq:         dlq,
		topic:       topic,
	}
}

func (s *Service) Process(ctx context.Context, event models.Event) (*models.NormalizedRecord, error) {
	sanitized, err := extractTokenizedData(event.Data)
	if err != nil {
		return nil, err
	}
	record, err := s.transformer.Transform(sanitized)
	if err != nil {
		return nil, err
	}

	repoRecord := &RecordModel{
		ID:           record.ID,
		PatientID:    record.PatientID,
		ResourceType: record.ResourceType,
		Canonical:    datatypes.JSONMap(record.Canonical),
		Codes:        mapToJSONMap(record.Codes),
		Timestamp:    record.Timestamp,
	}
	if err := s.repo.Save(ctx, repoRecord); err != nil {
		return nil, err
	}

	payload := map[string]interface{}{
		"normalized_record": record,
	}

	if err := s.producer.PublishEvent(ctx, "normalize", "normalizer-service", payload); err != nil {
		logger.Log.WithError(err).Error("failed to publish normalized event")
		if s.dlq != nil {
			_ = s.dlq.PublishEvent(ctx, "normalize", "normalizer-service", payload)
		}
		return nil, err
	}

	return record, nil
}

func (s *Service) TransformOnly(data map[string]interface{}) (*models.NormalizedRecord, error) {
	return s.transformer.Transform(data)
}

func extractTokenizedData(data map[string]interface{}) (map[string]interface{}, error) {
	if data == nil {
		return nil, fmt.Errorf("event data missing")
	}

	tokenized, ok := data["tokenized_data"].(map[string]interface{})
	if !ok {
		// fallback to sanitized data field
		tokenized, ok = data["sanitized_data"].(map[string]interface{})
		if !ok {
			// attempt to decode raw JSON string
			if raw, found := data["tokenized_data"].(string); found {
				var tmp map[string]interface{}
				if err := json.Unmarshal([]byte(raw), &tmp); err == nil {
					return tmp, nil
				}
			}
			return nil, fmt.Errorf("tokenized data not present")
		}
	}

	return tokenized, nil
}

func mapToJSONMap(in map[string]string) datatypes.JSONMap {
	out := make(datatypes.JSONMap, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}
