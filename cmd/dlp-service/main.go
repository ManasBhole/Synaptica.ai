package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type DLPService struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	// PHI patterns
	ssnPattern     *regexp.Regexp
	dobPattern    *regexp.Regexp
	emailPattern  *regexp.Regexp
	phonePattern  *regexp.Regexp
}

func main() {
	logger.Init()
	cfg := config.Load()

	// Initialize patterns
	service := &DLPService{
		ssnPattern:    regexp.MustCompile(`\b\d{3}-\d{2}-\d{4}\b`),
		dobPattern:    regexp.MustCompile(`\b\d{1,2}/\d{1,2}/\d{4}\b`),
		emailPattern:  regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`),
		phonePattern:  regexp.MustCompile(`\b\d{3}-\d{3}-\d{4}\b|\b\(\d{3}\)\s?\d{3}-\d{4}\b`),
	}

	// Kafka producer for downstream events
	service.producer = kafka.NewProducer("sanitized-events")
	defer service.producer.Close()

	// Kafka consumer for upstream events
	service.consumer = kafka.NewConsumer("upstream-events", "dlp-service")
	defer service.consumer.Close()

	// Start consumer in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := service.consumer.Consume(ctx, service.processEvent); err != nil {
			logger.Log.WithError(err).Fatal("Consumer error")
		}
	}()

	// HTTP server for direct API calls
	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/detect", service.handleDetect).Methods("POST")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8082"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8082",
		}).Info("DLP Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down DLP Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("DLP Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *DLPService) processEvent(ctx context.Context, event models.Event) error {
	logger.Log.WithFields(map[string]interface{}{
		"event_id": event.ID,
		"type":     event.Type,
	}).Info("Processing event for PHI detection")

	// Detect PHI in event data
	dataStr := fmt.Sprintf("%v", event.Data)
	result := s.detectPHI(dataStr, event.Data)

	// Publish sanitized event
	return s.producer.PublishEvent(ctx, "sanitize", "dlp-service", map[string]interface{}{
		"original_event_id": event.ID,
		"phi_detection":     result,
		"data":              event.Data,
	})
}

func (s *DLPService) handleDetect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	dataStr := fmt.Sprintf("%v", req.Data)
	result := s.detectPHI(dataStr, req.Data)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *DLPService) detectPHI(text string, data map[string]interface{}) models.PHIDetectionResult {
	var positions []models.PHIPosition
	var phiTypes []string
	detected := false

	// SSN detection
	if matches := s.ssnPattern.FindAllStringIndex(text, -1); len(matches) > 0 {
		detected = true
		phiTypes = append(phiTypes, "SSN")
		for _, match := range matches {
			positions = append(positions, models.PHIPosition{
				Start: match[0],
				End:   match[1],
				Type:  "SSN",
				Value: text[match[0]:match[1]],
			})
		}
	}

	// DOB detection
	if matches := s.dobPattern.FindAllStringIndex(text, -1); len(matches) > 0 {
		detected = true
		if !contains(phiTypes, "DOB") {
			phiTypes = append(phiTypes, "DOB")
		}
		for _, match := range matches {
			positions = append(positions, models.PHIPosition{
				Start: match[0],
				End:   match[1],
				Type:  "DOB",
				Value: text[match[0]:match[1]],
			})
		}
	}

	// Email detection
	if matches := s.emailPattern.FindAllStringIndex(text, -1); len(matches) > 0 {
		detected = true
		if !contains(phiTypes, "Email") {
			phiTypes = append(phiTypes, "Email")
		}
		for _, match := range matches {
			positions = append(positions, models.PHIPosition{
				Start: match[0],
				End:   match[1],
				Type:  "Email",
				Value: text[match[0]:match[1]],
			})
		}
	}

	// Phone detection
	if matches := s.phonePattern.FindAllStringIndex(text, -1); len(matches) > 0 {
		detected = true
		if !contains(phiTypes, "Phone") {
			phiTypes = append(phiTypes, "Phone")
		}
		for _, match := range matches {
			positions = append(positions, models.PHIPosition{
				Start: match[0],
				End:   match[1],
				Type:  "Phone",
				Value: text[match[0]:match[1]],
			})
		}
	}

	// LLM-based detection for names, addresses (would call LLM service)
	// For now, placeholder
	if strings.Contains(strings.ToLower(text), "patient") || strings.Contains(strings.ToLower(text), "name") {
		detected = true
		if !contains(phiTypes, "Name") {
			phiTypes = append(phiTypes, "Name")
		}
	}

	confidence := 0.0
	if detected {
		confidence = 0.85 // Base confidence, would be higher with LLM
	}

	return models.PHIDetectionResult{
		Detected:   detected,
		Confidence: confidence,
		PHITypes:   phiTypes,
		Positions:  positions,
	}
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

