package kafka

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/segmentio/kafka-go"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type Consumer struct {
	reader *kafka.Reader
}

type EventHandler func(ctx context.Context, event models.Event) error

func NewConsumer(topic string, groupID string) *Consumer {
	cfg := config.Load()
	if groupID == "" {
		groupID = cfg.KafkaGroupID
	}

	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  cfg.KafkaBrokers,
		Topic:    topic,
		GroupID:  groupID,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})

	return &Consumer{reader: reader}
}

func (c *Consumer) Consume(ctx context.Context, handler EventHandler) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			message, err := c.reader.FetchMessage(ctx)
			if err != nil {
				logger.Log.WithError(err).Error("Failed to fetch message")
				continue
			}

			var event models.Event
			if err := json.Unmarshal(message.Value, &event); err != nil {
				logger.Log.WithError(err).Error("Failed to unmarshal event")
				c.reader.CommitMessages(ctx, message)
				continue
			}

			if err := handler(ctx, event); err != nil {
				logger.Log.WithError(err).WithFields(map[string]interface{}{
					"event_id": event.ID,
				}).Error("Failed to process event")
				// Don't commit on error, will retry
				continue
			}

			if err := c.reader.CommitMessages(ctx, message); err != nil {
				logger.Log.WithError(err).Error("Failed to commit message")
			}
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

