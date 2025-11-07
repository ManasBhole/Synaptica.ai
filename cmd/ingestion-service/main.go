package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/ingestion"
)

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("failed to connect to postgres")
	}

	repo := ingestion.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate ingestion tables")
	}

	validator := ingestion.NewValidator(cfg.IngestionAllowedSources, []string{"fhir", "hl7", "json", "csv", "dicom"})

	producer := kafka.NewProducer(cfg.IngestionKafkaTopic)
	defer producer.Close()

	var dlqProducer *kafka.Producer
	if cfg.IngestionDLQTopic != "" {
		dlqProducer = kafka.NewProducer(cfg.IngestionDLQTopic)
		defer dlqProducer.Close()
	}

	svc := ingestion.NewService(validator, repo, producer, dlqProducer, cfg.IngestionStatusTTL)
	handler := ingestion.NewHTTPHandler(svc, cfg.MaxRequestBody)

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)

	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)

	api := router.PathPrefix("/api/v1").Subrouter()
	handler.Register(api)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, "8081"),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8081",
		}).Info("Ingestion Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start server")
		}
	}()

	go func() {
		ticker := time.NewTicker(12 * time.Hour)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if err := svc.Cleanup(context.Background()); err != nil {
					logger.Log.WithError(err).Warn("cleanup job failed")
				}
			case <-ctx.Done():
				return
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Ingestion Service...")
	cancel()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		logger.Log.WithError(err).Error("server forced to shutdown")
	}

	logger.Log.Info("Ingestion Service stopped")
}
