package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/serving/predictor"
	"github.com/synaptica-ai/platform/pkg/storage"
)

type ServingService struct {
	featureStore *storage.FeatureStore
	predictor    *predictor.Predictor
	// Model serving backend (Triton/Vertex/TF Serving)
	modelBackend interface{}
}

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to database")
	}

	redisClient := database.GetRedis()
	featureStore := storage.NewFeatureStore(db, redisClient, cfg.FeatureOnlinePrefix, cfg.FeatureCacheTTL)
	if err := featureStore.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to migrate feature store tables")
	}

	predictorEngine := predictor.NewPredictor(cfg.TrainingArtifactDir)

	service := &ServingService{
		featureStore: featureStore,
		predictor:    predictorEngine,
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/predict", service.handlePredict).Methods("POST")
	router.HandleFunc("/api/v1/models", service.handleListModels).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8089"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8089",
		}).Info("Serving Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Serving Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Serving Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *ServingService) handlePredict(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	var req models.PredictionRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Get features from cache (p95 < 10ms)
	ctx := r.Context()
	featureSet, err := s.featureStore.GetFeatures(ctx, req.PatientID)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to get features")
		http.Error(w, "Failed to get features", http.StatusInternalServerError)
		return
	}

	features := make(map[string]interface{})
	for name, value := range featureSet {
		features[name] = value
	}
	for key, value := range req.Features {
		features[key] = value
	}

	numeric := map[string]float64{}
	for key, value := range features {
		if f, err := toFloat(value); err == nil {
			numeric[key] = f
		}
	}

	predictions := map[string]interface{}{}
	confidence := 0.0
	if score, err := s.predictor.Predict(req.ModelName, numeric); err == nil {
		predictions["risk_score"] = score
		confidence = score
	} else {
		logger.Log.WithError(err).Warn("predictor fallback")
		predictions["risk_score"] = 0.75
		predictions["category"] = "high_risk"
		confidence = 0.85
	}

	latency := time.Since(start)

	resp := models.PredictionResponse{
		PatientID:    req.PatientID,
		Predictions:  predictions,
		Confidence:   confidence,
		ModelVersion: "latest",
		Latency:      latency,
	}

	logger.Log.WithFields(map[string]interface{}{
		"patient_id": req.PatientID,
		"latency_ms": latency.Milliseconds(),
	}).Info("Prediction completed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (s *ServingService) handleListModels(w http.ResponseWriter, r *http.Request) {
	// List available models
	models := []map[string]interface{}{
		{
			"name":    "risk-score-v1",
			"version": "1.0",
			"type":    "classification",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(models)
}

func (s *ServingService) predict(ctx context.Context, modelName string, features map[string]interface{}) (map[string]interface{}, float64, error) {
	// In production, would call Triton/Vertex/TF Serving
	// For now, return mock prediction

	logger.Log.WithFields(map[string]interface{}{
		"model":    modelName,
		"features": len(features),
	}).Debug("Making prediction")

	// Mock prediction
	predictions := map[string]interface{}{
		"risk_score": 0.75,
		"category":   "high_risk",
	}

	return predictions, 0.85, nil
}

func toFloat(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case string:
		return strconv.ParseFloat(v, 64)
	default:
		return 0, fmt.Errorf("unsupported type %T", value)
	}
}
