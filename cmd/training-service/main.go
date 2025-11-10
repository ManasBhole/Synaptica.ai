package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/storage"
	"github.com/synaptica-ai/platform/pkg/training"
)

type TrainingApp struct {
	service *training.Service
}

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to database")
	}

	repo := training.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to migrate training tables")
	}

	lakehouse := storage.NewLakehouseWriter(db)
	if err := lakehouse.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to migrate lakehouse tables")
	}

	redisClient := database.GetRedis()
	featureStore := storage.NewFeatureStore(db, redisClient, cfg.FeatureOnlinePrefix, cfg.FeatureCacheTTL)
	if err := featureStore.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to migrate feature store tables")
	}

	service, err := training.NewService(repo, lakehouse, featureStore, cfg.TrainingArtifactDir, cfg.TrainingMaxWorkers, cfg.TrainingSimulationDelay)
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize training service")
	}

	app := &TrainingApp{service: service}

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/training/jobs", app.handleCreateJob).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/training/jobs", app.handleListJobs).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/training/jobs/{id}", app.handleGetJob).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/training/jobs/{id}/status", app.handleGetJob).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/training/jobs/{id}/artifact", app.handleArtifact).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/training/jobs/{id}/promote", app.handlePromoteJob).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/training/jobs/{id}/deprecate", app.handleDeprecateJob).Methods(http.MethodPost)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, "8088"),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
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

func (a *TrainingApp) handleCreateJob(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ModelType string                 `json:"model_type"`
		Config    map[string]interface{} `json:"config"`
		Filters   map[string]interface{} `json:"filters"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ModelType == "" {
		http.Error(w, "model_type is required", http.StatusBadRequest)
		return
	}

	job, err := a.service.Create(r.Context(), training.CreateJobInput{
		ModelType: req.ModelType,
		Config:    req.Config,
		Filters:   req.Filters,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(job)
}

func (a *TrainingApp) handleGetJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	job, err := a.service.Get(r.Context(), jobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (a *TrainingApp) handleListJobs(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	jobs, err := a.service.List(r.Context(), limit)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"jobs": jobs})
}

func (a *TrainingApp) handleArtifact(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	artifact, err := a.service.GetArtifact(jobID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	if artifact.Path == "" {
		http.Error(w, "artifact not available yet", http.StatusNotFound)
		return
	}

	content, err := os.ReadFile(artifact.Path)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(content)
}

func (a *TrainingApp) handlePromoteJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		PromotedBy       string `json:"promoted_by"`
		Notes            string `json:"notes"`
		DeploymentTarget string `json:"deployment_target"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	job, err := a.service.Promote(r.Context(), jobID, training.PromotionInput{
		PromotedBy:       req.PromotedBy,
		Notes:            req.Notes,
		DeploymentTarget: req.DeploymentTarget,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func (a *TrainingApp) handleDeprecateJob(w http.ResponseWriter, r *http.Request) {
	jobID, err := parseJobID(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	var req struct {
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err != io.EOF {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	job, err := a.service.Deprecate(r.Context(), jobID, req.Notes)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

func parseJobID(r *http.Request) (uuid.UUID, error) {
	id := mux.Vars(r)["id"]
	if id == "" {
		return uuid.Nil, fmt.Errorf("missing job id")
	}
	jobID, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid job id")
	}
	return jobID, nil
}
