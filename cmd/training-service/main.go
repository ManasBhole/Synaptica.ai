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
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/storage"
)

type TrainingService struct {
	lakehouse   *storage.LakehouseStorage
	featureStore *storage.FeatureStore
}

func main() {
	logger.Init()
	cfg := config.Load()

	lakehouse, err := storage.NewLakehouseStorage()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize lakehouse")
	}

	featureStore, err := storage.NewFeatureStore()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize feature store")
	}

	service := &TrainingService{
		lakehouse:    lakehouse,
		featureStore: featureStore,
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/training/jobs", service.handleCreateJob).Methods("POST")
	router.HandleFunc("/api/v1/training/jobs/{id}", service.handleGetJob).Methods("GET")
	router.HandleFunc("/api/v1/training/jobs/{id}/status", service.handleGetStatus).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8088"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8088",
		}).Info("Training Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Training Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Training Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *TrainingService) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelType string                 `json:"model_type"`
		Config    map[string]interface{} `json:"config"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	job := models.TrainingJob{
		ID:        uuid.New(),
		ModelType: req.ModelType,
		Config:    req.Config,
		Status:    "queued",
		CreatedAt: time.Now(),
	}

	// Start training in background
	go s.trainModel(context.Background(), job)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(job)
}

func (s *TrainingService) handleGetJob(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// In production, would fetch from database
	job := models.TrainingJob{
		ID:        uuid.MustParse(id),
		Status:    "completed",
		CreatedAt: time.Now(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (s *TrainingService) handleGetStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// In production, would fetch from database
	status := map[string]interface{}{
		"job_id": id,
		"status": "completed",
		"progress": 100,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *TrainingService) trainModel(ctx context.Context, job models.TrainingJob) {
	logger.Log.WithFields(map[string]interface{}{
		"job_id": job.ID,
		"model_type": job.ModelType,
	}).Info("Starting model training")

	// Get training data from Lakehouse
	trainingData, err := s.lakehouse.GetTrainingData(ctx, job.Config)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get training data")
		return
	}

	// Get feature views from Feature Store
	featureNames := []string{} // Extract from config
	featureViews, err := s.featureStore.GetFeatureViews(ctx, featureNames)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get feature views")
		return
	}

	logger.Log.WithFields(map[string]interface{}{
		"training_samples": len(trainingData),
		"feature_views": len(featureViews),
	}).Info("Training data prepared")

	// In production, would:
	// 1. Prepare features
	// 2. Split train/validation/test
	// 3. Train model (TensorFlow, PyTorch, AutoML)
	// 4. Evaluate model
	// 5. Save model artifacts
	// 6. Update job status

	logger.Log.WithField("job_id", job.ID).Info("Model training completed")
}

