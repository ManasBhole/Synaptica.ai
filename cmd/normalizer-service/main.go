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
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/normalizer"
	"github.com/synaptica-ai/platform/pkg/terminology"
)

type NormalizerApp struct {
	service  *normalizer.Service
	producer *kafka.Producer
	consumer *kafka.Consumer
}

func main() {
	logger.Init()
	cfg := config.Load()

	catalog, err := terminology.Load(cfg.TerminologyCatalogPath)
	if err != nil {
		logger.Log.WithError(err).Warn("failed to load terminology catalog, using defaults")
		catalog = terminology.DefaultCatalog()
	}

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("failed to connect to postgres")
	}

	repo := normalizer.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate normalized records table")
	}

	transformer := normalizer.NewTransformer(catalog, cfg.NormalizerAllowedResources)

	producer := kafka.NewProducer(cfg.NormalizerOutputTopic)
	defer producer.Close()

	var dlq *kafka.Producer
	if cfg.NormalizerDLQTopic != "" {
		dlq = kafka.NewProducer(cfg.NormalizerDLQTopic)
		defer dlq.Close()
	}

	service := normalizer.NewService(transformer, repo, producer, dlq, cfg.NormalizerOutputTopic)

	app := &NormalizerApp{service: service}
	app.producer = producer
	app.consumer = kafka.NewConsumer("deidentified-events", "normalizer-service")
	defer app.consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := app.consumer.Consume(ctx, app.handleEvent); err != nil {
			logger.Log.WithError(err).Fatal("consumer error")
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods(http.MethodGet)

	router.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ready"}`))
	}).Methods(http.MethodGet)

	router.HandleFunc("/api/v1/normalize", app.handleNormalize).Methods(http.MethodPost)

	server := &http.Server{
		Addr:         fmt.Sprintf("%s:%s", cfg.ServerHost, "8084"),
		Handler:      router,
		ReadTimeout:  cfg.ReadTimeout,
		WriteTimeout: cfg.WriteTimeout,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8084",
		}).Info("Normalizer Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Normalizer Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("server forced to shutdown")
	}

	logger.Log.Info("Normalizer Service stopped")
}

func (a *NormalizerApp) handleEvent(ctx context.Context, event models.Event) error {
	_, err := a.service.Process(ctx, event)
	return err
}

func (a *NormalizerApp) handleNormalize(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	record, err := a.service.TransformOnly(req.Data)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(record)
}
