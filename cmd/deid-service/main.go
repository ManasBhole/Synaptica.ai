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
	"github.com/synaptica-ai/platform/pkg/deid"
)

type DeIDApp struct {
	service  *deid.Service
	producer *kafka.Producer
	consumer *kafka.Consumer
}

func main() {
	logger.Init()
	cfg := config.Load()

	db, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("failed to connect to postgres")
	}

	repo := deid.NewRepository(db)
	if err := repo.AutoMigrate(); err != nil {
		logger.Log.WithError(err).Fatal("failed to migrate deid tables")
	}

	service := deid.NewService(repo, cfg.DeIDTokenSalt)

	app := &DeIDApp{service: service}
	app.producer = kafka.NewProducer("deidentified-events")
	defer app.producer.Close()

	app.consumer = kafka.NewConsumer("sanitized-events", "deid-service")
	defer app.consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := app.consumer.Consume(ctx, app.processEvent); err != nil {
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

	router.HandleFunc("/api/v1/deid", app.handleDeID).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8083"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8083",
		}).Info("De-ID Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down De-ID Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("server forced to shutdown")
	}

	logger.Log.Info("De-ID Service stopped")
}

func (a *DeIDApp) processEvent(ctx context.Context, event models.Event) error {
	detection, sanitized, err := parseDLPPayload(event.Data)
	if err != nil {
		logger.Log.WithError(err).Error("invalid DLP payload")
		return err
	}

	tokenized, vault, anonymity, err := a.service.Tokenize(ctx, sanitized, detection)
	if err != nil {
		logger.Log.WithError(err).Error("failed to tokenize data")
		return err
	}

	payload := map[string]interface{}{
		"original_event_id": event.ID,
		"tokenized_data":    tokenized,
		"token_vault":       vault,
		"anonymity_level":   anonymity,
	}

	return a.producer.PublishEvent(ctx, "deidentify", "deid-service", payload)
}

func (a *DeIDApp) handleDeID(w http.ResponseWriter, r *http.Request) {
	var req models.DeIDRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	tokenized, vault, anonymity, err := a.service.Tokenize(r.Context(), req.Data, req.PHIResults)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	resp := models.DeIDResponse{
		TokenizedData:  tokenized,
		TokenVault:     vault,
		AnonymityLevel: anonymity,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func parseDLPPayload(data map[string]interface{}) (models.PHIDetectionResult, map[string]interface{}, error) {
	detectionMap, ok := data["phi_detection"].(map[string]interface{})
	if !ok {
		return models.PHIDetectionResult{}, nil, fmt.Errorf("phi_detection missing")
	}

	detectionBytes, err := json.Marshal(detectionMap)
	if err != nil {
		return models.PHIDetectionResult{}, nil, err
	}
	var detection models.PHIDetectionResult
	if err := json.Unmarshal(detectionBytes, &detection); err != nil {
		return models.PHIDetectionResult{}, nil, err
	}

	raw, ok := data["raw_data"].(map[string]interface{})
	if !ok {
		// fallback to sanitized
		raw, ok = data["sanitized_data"].(map[string]interface{})
		if !ok {
			return models.PHIDetectionResult{}, nil, fmt.Errorf("no data payload present")
		}
	}

	return detection, raw, nil
}
