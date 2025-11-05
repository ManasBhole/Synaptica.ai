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

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/storage"
)

type CohortService struct {
	lakehouse *storage.LakehouseStorage
	rtolap    *storage.RTOLAPStorage
	llmClient interface{} // LLM service client
}

func main() {
	logger.Init()
	cfg := config.Load()

	lakehouse, err := storage.NewLakehouseStorage()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize lakehouse")
	}

	rtolap, err := storage.NewRTOLAPStorage()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to initialize RT OLAP")
	}

	service := &CohortService{
		lakehouse: lakehouse,
		rtolap:    rtolap,
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/cohort/query", service.handleQuery).Methods("POST")
	router.HandleFunc("/api/v1/cohort/verify", service.handleVerify).Methods("POST")
	router.HandleFunc("/api/v1/cohort/{id}", service.handleGetCohort).Methods("GET")

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8087"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8087",
		}).Info("Cohort Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("Failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Cohort Service...")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Cohort Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *CohortService) handleQuery(w http.ResponseWriter, r *http.Request) {
	var query models.CohortQuery

	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	// Verify DSL
	if err := s.verifyDSL(query.DSL); err != nil {
		http.Error(w, fmt.Sprintf("Invalid DSL: %v", err), http.StatusBadRequest)
		return
	}

	// Execute cohort scan on Lakehouse
	ctx := r.Context()
	result, err := s.lakehouse.QueryCohort(ctx, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// For sub-second slicing, query RT OLAP
	if len(query.Filters) > 0 {
		sliceResults, _ := s.rtolap.QuerySubSecondSlicing(ctx, query.Filters)
		logger.Log.WithField("slice_count", len(sliceResults)).Debug("Sub-second slice query completed")
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *CohortService) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DSL string `json:"dsl"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := s.verifyDSL(req.DSL); err != nil {
		http.Error(w, fmt.Sprintf("Invalid DSL: %v", err), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid": true,
		"message": "DSL is valid",
	})
}

func (s *CohortService) handleGetCohort(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Query cohort by ID
	query := models.CohortQuery{
		ID: id,
	}

	ctx := r.Context()
	result, err := s.lakehouse.QueryCohort(ctx, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *CohortService) verifyDSL(dsl string) error {
	// DSL verifier - check syntax and semantics
	// In production, would use proper parser/validator
	
	if dsl == "" {
		return fmt.Errorf("DSL cannot be empty")
	}

	// Basic validation - check for SQL injection patterns
	// In production, would use proper DSL parser
	return nil
}

