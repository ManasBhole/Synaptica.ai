package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type NormalizerService struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	// Code mappings (SNOMED, LOINC, ICD)
	codeMappings map[string]map[string]string
}

func main() {
	logger.Init()
	cfg := config.Load()

	service := &NormalizerService{
		codeMappings: loadCodeMappings(),
	}

	// Kafka producer
	service.producer = kafka.NewProducer("normalized-events")
	defer service.producer.Close()

	// Kafka consumer
	service.consumer = kafka.NewConsumer("deidentified-events", "normalizer-service")
	defer service.consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := service.consumer.Consume(ctx, service.processEvent); err != nil {
			logger.Log.WithError(err).Fatal("Consumer error")
		}
	}()

	// HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/normalize", service.handleNormalize).Methods("POST")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8084"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8084",
		}).Info("Normalizer Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Normalizer Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Normalizer Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *NormalizerService) processEvent(ctx context.Context, event models.Event) error {
	logger.Log.WithFields(map[string]interface{}{
		"event_id": event.ID,
	}).Info("Processing event for normalization")

	// Extract tokenized data
	tokenizedData, ok := event.Data["tokenized_data"].(map[string]interface{})
	if !ok {
		logger.Log.Warn("No tokenized data found in event")
		return nil
	}

	// Normalize to canonical FHIR format
	normalized := s.normalizeToFHIR(tokenizedData)

	// Publish normalized event
	return s.producer.PublishEvent(ctx, "normalize", "normalizer-service", map[string]interface{}{
		"original_event_id": event.Data["original_event_id"],
		"normalized":        normalized,
	})
}

func (s *NormalizerService) handleNormalize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	normalized := s.normalizeToFHIR(req.Data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(normalized)
}

func (s *NormalizerService) normalizeToFHIR(data map[string]interface{}) models.NormalizedRecord {
	// Convert to canonical FHIR format
	// This is a simplified version - in production, would have full FHIR mapping logic

	record := models.NormalizedRecord{
		ID:          uuid.New().String(),
		PatientID:   extractPatientID(data),
		ResourceType: extractResourceType(data),
		Canonical:   make(map[string]interface{}),
		Codes:       make(map[string]string),
		Timestamp:   time.Now(),
	}

	// Map fields to canonical FHIR structure
	for key, value := range data {
		// Map common fields
		switch key {
		case "observation", "lab_result", "vital":
			record.Canonical["resourceType"] = "Observation"
			record.Canonical["value"] = value
		case "condition", "diagnosis":
			record.Canonical["resourceType"] = "Condition"
			record.Canonical["code"] = value
		case "procedure":
			record.Canonical["resourceType"] = "Procedure"
			record.Canonical["code"] = value
		}
	}

	// Map codes (SNOMED, LOINC, ICD)
	if code, ok := data["code"].(string); ok {
		record.Codes["SNOMED"] = s.mapCode(code, "SNOMED")
		record.Codes["LOINC"] = s.mapCode(code, "LOINC")
		record.Codes["ICD"] = s.mapCode(code, "ICD")
	}

	return record
}

func extractPatientID(data map[string]interface{}) string {
	if id, ok := data["patient_id"].(string); ok {
		return id
	}
	return uuid.New().String()
}

func extractResourceType(data map[string]interface{}) string {
	if rt, ok := data["resource_type"].(string); ok {
		return rt
	}
	return "Observation" // Default
}

func (s *NormalizerService) mapCode(code, system string) string {
	// Code mapping logic - simplified
	if mappings, ok := s.codeMappings[system]; ok {
		if mapped, ok := mappings[code]; ok {
			return mapped
		}
	}
	return code
}

func loadCodeMappings() map[string]map[string]string {
	// In production, load from database or external service
	return map[string]map[string]string{
		"SNOMED": {
			"example": "123456789",
		},
		"LOINC": {
			"example": "12345-6",
		},
		"ICD": {
			"example": "E11.9",
		},
	}
}

