package kafka

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type Producer struct {
	writer *kafka.Writer
}

func NewProducer(topic string) *Producer {
	cfg := config.Load()
	writer := &kafka.Writer{
		Addr:         kafka.TCP(cfg.KafkaBrokers...),
		Topic:        topic,
		Balancer:     &kafka.LeastBytes{},
		RequiredAcks: kafka.RequireAll,
		Async:        false,
		BatchSize:    1,
		BatchTimeout: 10 * time.Millisecond,
	}

	return &Producer{writer: writer}
}

func (p *Producer) PublishEvent(ctx context.Context, eventType string, source string, data map[string]interface{}) error {
	event := models.Event{
		ID:        uuid.New().String(),
		Type:      eventType,
		Source:    source,
		Data:      data,
		Timestamp: time.Now(),
	}

	eventBytes, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	message := kafka.Message{
		Key:   []byte(event.ID),
		Value: eventBytes,
		Headers: []kafka.Header{
			{Key: "event-type", Value: []byte(eventType)},
			{Key: "source", Value: []byte(source)},
		},
	}

	if err := p.writer.WriteMessages(ctx, message); err != nil {
		logger.Log.WithError(err).WithFields(map[string]interface{}{
			"event_id":   event.ID,
			"event_type": eventType,
		}).Error("Failed to publish event")
		return err
	}

	logger.Log.WithFields(map[string]interface{}{
		"event_id":   event.ID,
		"event_type": eventType,
		"topic":      p.writer.Topic,
	}).Info("Event published successfully")

	return nil
}

func (p *Producer) Close() error {
	return p.writer.Close()
}

