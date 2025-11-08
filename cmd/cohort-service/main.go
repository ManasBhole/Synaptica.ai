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
	"github.com/synaptica-ai/platform/pkg/analytics/cohort"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/linkage"
	"github.com/synaptica-ai/platform/pkg/storage"
)

type CohortApp struct {
	service *cohort.Service
}

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to database")
	}

	redisClient := database.GetRedis()

	lakehouse := storage.NewLakehouseWriter(db)
	if err := lakehouse.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to migrate lakehouse tables")
	}

	olapWriter := storage.NewOLAPWriter(db)
	if err := olapWriter.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("Failed to migrate OLAP tables")
	}

	featureStore := storage.NewFeatureStore(db, redisClient, cfg.FeatureOnlinePrefix, cfg.FeatureCacheTTL)
	linkageRepo := linkage.NewRepository(db)
	app := &CohortApp{
		service: cohort.NewService(
			lakehouse,
			olapWriter,
			cohort.WithFeatureStore(featureStore),
			cohort.WithLinkageRepository(linkageRepo),
		),
	}

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/cohort/query", app.handleQuery).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/cohort/verify", app.handleVerify).Methods(http.MethodPost)
	router.HandleFunc("/api/v1/cohort/{id}", app.handleDrilldown).Methods(http.MethodPost)

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

func (a *CohortApp) handleQuery(w http.ResponseWriter, r *http.Request) {
	var query models.CohortQuery
	if err := json.NewDecoder(r.Body).Decode(&query); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	result, err := a.service.Execute(ctx, query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (a *CohortApp) handleVerify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DSL string `json:"dsl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	if err := a.service.VerifyDSL(req.DSL); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":   true,
		"message": "DSL is valid",
	})
}

func (a *CohortApp) handleDrilldown(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	id := mux.Vars(r)["id"]
	if id == "" {
		http.Error(w, "cohort id is required", http.StatusBadRequest)
		return
	}

	var req models.CohortDrilldownRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	req.CohortID = id

	ctx := r.Context()
	result, err := a.service.Drilldown(ctx, req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}
