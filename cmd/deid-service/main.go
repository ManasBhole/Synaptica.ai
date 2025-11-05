package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type DeIDService struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	db       interface{} // Token vault storage
}

func main() {
	logger.Init()
	cfg := config.Load()

	service := &DeIDService{}

	// Initialize database for token vault
	_, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to database")
	}

	// Kafka producer
	service.producer = kafka.NewProducer("deidentified-events")
	defer service.producer.Close()

	// Kafka consumer
	service.consumer = kafka.NewConsumer("sanitized-events", "deid-service")
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
	router.HandleFunc("/api/v1/deid", service.handleDeID).Methods("POST")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8083"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8083",
		}).Info("De-ID Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down De-ID Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("De-ID Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *DeIDService) processEvent(ctx context.Context, event models.Event) error {
	logger.Log.WithFields(map[string]interface{}{
		"event_id": event.ID,
	}).Info("Processing event for de-identification")

	// Extract PHI detection results
	phiDetection, ok := event.Data["phi_detection"].(map[string]interface{})
	if !ok {
		// No PHI detected, pass through
		return s.producer.PublishEvent(ctx, "deidentify", "deid-service", event.Data)
	}

	// Perform de-identification
	tokenizedData, tokenVault := s.deidentify(event.Data, phiDetection)

	// Check k/l-diversity (simplified)
	anonymityLevel := s.checkAnonymityLevel(tokenizedData)

	// Publish de-identified event
	return s.producer.PublishEvent(ctx, "deidentify", "deid-service", map[string]interface{}{
		"original_event_id": event.Data["original_event_id"],
		"tokenized_data":    tokenizedData,
		"token_vault":       tokenVault,
		"anonymity_level":   anonymityLevel,
	})
}

func (s *DeIDService) handleDeID(w http.ResponseWriter, r *http.Request) {
	var req models.DeIDRequest
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// This would parse JSON body properly
	// For now, placeholder
    tokenizedData, _ := s.deidentify(req.Data, map[string]interface{}{})
	anonymityLevel := s.checkAnonymityLevel(tokenizedData)

	w.Header().Set("Content-Type", "application/json")
	fmt.Fprintf(w, `{"tokenized_data":%v,"anonymity_level":"%s"}`, tokenizedData, anonymityLevel)
}

func (s *DeIDService) deidentify(data map[string]interface{}, phiDetection map[string]interface{}) (map[string]interface{}, map[string]string) {
	tokenized := make(map[string]interface{})
	vault := make(map[string]string)

	for key, value := range data {
		valueStr := fmt.Sprintf("%v", value)

		// Generate token for PHI values
		token := s.generateToken(valueStr)
		tokenized[key] = token
		vault[token] = valueStr
	}

	return tokenized, vault
}

func (s *DeIDService) generateToken(value string) string {
	// Use SHA256 hash for tokenization
	hash := sha256.Sum256([]byte(value + uuid.New().String()))
	return "token_" + hex.EncodeToString(hash[:])[:16]
}

func (s *DeIDService) checkAnonymityLevel(data map[string]interface{}) string {
	// Simplified k/l-diversity check
	// In production, would check actual k-anonymity and l-diversity requirements
	if len(data) >= 3 {
		return "k-3"
	}
	return "k-1"
}

