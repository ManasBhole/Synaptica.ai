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

type IngestionService struct {
	producer *kafka.Producer
	statuses map[string]models.IngestResponse
}

func main() {
	logger.Init()
	cfg := config.Load()

	// Initialize Kafka producer
	producer := kafka.NewProducer("upstream-events")
	defer producer.Close()

	service := &IngestionService{
		producer: producer,
		statuses: make(map[string]models.IngestResponse),
	}

	// Setup router
	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/ingest", service.handleIngest).Methods("POST")
	router.HandleFunc("/api/v1/ingest/status/{id}", service.handleStatus).Methods("GET")

	// Server
	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, "8081"),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8081",
		}).Info("Ingestion Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Ingestion Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Ingestion Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *IngestionService) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req models.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate ingestion ID
	ingestID := uuid.New().String()

	// Create response
	resp := models.IngestResponse{
		ID:        ingestID,
		Status:    "accepted",
		Timestamp: time.Now(),
	}

	// Store status
	s.statuses[ingestID] = resp

	// Publish to event bus
	ctx := r.Context()
	if err := s.producer.PublishEvent(ctx, "upstream", req.Source, map[string]interface{}{
		"ingest_id": ingestID,
		"source":    req.Source,
		"format":    req.Format,
		"data":      req.Data,
		"patient_id": req.PatientID,
		"metadata":  req.Metadata,
	}); err != nil {
		logger.Log.WithError(err).Error("Failed to publish event")
		http.Error(w, "Failed to process ingestion", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func (s *IngestionService) handleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	status, exists := s.statuses[id]
	if !exists {
		http.Error(w, "Ingestion not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

