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
	"github.com/synaptica-ai/platform/pkg/linkage"
)

type LinkageApp struct {
	service  *linkage.Service
	consumer *kafka.Consumer
}

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("failed to connect to postgres")
	}

	repo := linkage.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate linkage tables")
	}

	matcher := linkage.NewMatcher(cfg.LinkageDeterministicKeys, cfg.LinkageThreshold)

	producer := kafka.NewProducer(cfg.LinkageOutputTopic)
	defer producer.Close()

	var dlq *kafka.Producer
	if cfg.LinkageDLQTopic != "" {
		dlq = kafka.NewProducer(cfg.LinkageDLQTopic)
		defer dlq.Close()
	}

	svc := linkage.NewService(repo, matcher, producer, dlq, cfg.LinkageOutputTopic)

	app := &LinkageApp{service: svc}
	app.consumer = kafka.NewConsumer(cfg.NormalizerOutputTopic, "linkage-service")
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

	router.HandleFunc("/api/v1/link", app.handleLink).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8085"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8085",
		}).Info("Linkage Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down Linkage Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("server forced to shutdown")
	}

	logger.Log.Info("Linkage Service stopped")
}

func (a *LinkageApp) handleEvent(ctx context.Context, event models.Event) error {
	record, err := parseNormalizedRecord(event.Data)
	if err != nil {
		return err
	}
	_, err = a.service.Process(ctx, record)
	return err
}

func (a *LinkageApp) handleLink(w http.ResponseWriter, r *http.Request) {
	var req models.LinkageRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	results := make([]models.LinkageResult, 0, len(req.Records))
	for _, rec := range req.Records {
		record := rec
		res, err := a.service.Process(r.Context(), &record)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		results = append(results, *res)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func parseNormalizedRecord(data map[string]interface{}) (*models.NormalizedRecord, error) {
	payload, ok := data["normalized_record"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("normalized_record payload missing")
	}
	bytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	var record models.NormalizedRecord
	if err := json.Unmarshal(bytes, &record); err != nil {
		return nil, err
	}
	return &record, nil
}
