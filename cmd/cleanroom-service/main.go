package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
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
)

type CleanRoomService struct {
	dpBudgets map[string]float64 // Track DP budgets per query
	lineage   map[string][]string // Track data lineage
}

func main() {
	logger.Init()
	cfg := config.Load()

	service := &CleanRoomService{
		dpBudgets: make(map[string]float64),
		lineage:   make(map[string][]string),
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/cleanroom/query", service.handleQuery).Methods("POST")
	router.HandleFunc("/api/v1/cleanroom/query/{id}", service.handleGetQuery).Methods("GET")
	router.HandleFunc("/api/v1/cleanroom/lineage/{id}", service.handleGetLineage).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8090"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8090",
		}).Info("Clean Room Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Clean Room Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Clean Room Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *CleanRoomService) handleQuery(w http.ResponseWriter, r *http.Request) {
	var req models.CleanRoomQuery

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	queryID := uuid.New().String()
	req.ID = queryID

	// Execute query with differential privacy
	ctx := r.Context()
	result, err := s.executeQuery(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Track lineage
	s.lineage[queryID] = req.Lineage

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *CleanRoomService) handleGetQuery(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// In production, would fetch from database
	result := models.CleanRoomResult{
		QueryID:      id,
		Aggregates:   map[string]interface{}{},
		DPBudgetUsed: s.dpBudgets[id],
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *CleanRoomService) handleGetLineage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	lineage := s.lineage[id]
	if lineage == nil {
		lineage = []string{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"query_id": id,
		"lineage":  lineage,
	})
}

func (s *CleanRoomService) executeQuery(ctx context.Context, query models.CleanRoomQuery) (models.CleanRoomResult, error) {
	logger.Log.WithFields(map[string]interface{}{
		"query_id": query.ID,
		"dp_budget": query.DPBudget,
	}).Info("Executing clean room query")

	// Execute query (would query actual data store)
	aggregates := map[string]interface{}{
		"count":     1000,
		"avg_age":   45.5,
		"avg_bp":    120.5,
	}

	// Add differential privacy noise
	noise := s.addDPNoise(query.DPBudget)
	noisyAggregates := make(map[string]interface{})
	for key, value := range aggregates {
		if num, ok := value.(float64); ok {
			noisyAggregates[key] = num + noise
		} else {
			noisyAggregates[key] = value
		}
	}

	// Track DP budget usage
	s.dpBudgets[query.ID] = query.DPBudget

	result := models.CleanRoomResult{
		QueryID:      query.ID,
		Aggregates:   noisyAggregates,
		NoiseAdded:   noise,
		DPBudgetUsed: query.DPBudget,
	}

	return result, nil
}

func (s *CleanRoomService) addDPNoise(epsilon float64) float64 {
	// Add Laplace noise for differential privacy
	// In production, would use proper DP library
	if epsilon == 0 {
		return 0
	}

	// Simplified Laplace noise
	scale := 1.0 / epsilon
	noise := rand.NormFloat64() * scale

	return noise
}

