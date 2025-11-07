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
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/dlp"
)

type DLPService struct {
	detector *dlp.Detector
	producer *kafka.Producer
	consumer *kafka.Consumer
}

func main() {
	logger.Init()
	cfg := config.Load()

	rulesCfg, err := dlp.LoadRules(cfg.DLPRulesPath)
	if err != nil {
		logger.Log.WithError(err).Warn("failed to load custom DLP rules, using defaults")
		rulesCfg = dlp.DefaultRules()
	}

	detector, err := dlp.NewDetector(rulesCfg)
	if err != nil {
		logger.Log.WithError(err).Fatal("unable to compile DLP rules")
	}

	service := &DLPService{detector: detector}
	service.producer = kafka.NewProducer("sanitized-events")
	defer service.producer.Close()

	service.consumer = kafka.NewConsumer("upstream-events", "dlp-service")
	defer service.consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := service.consumer.Consume(ctx, service.processEvent); err != nil {
			logger.Log.WithError(err).Fatal("consumer error")
		}
	}()

	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods(http.MethodGet)
	router.HandleFunc("/ready", readinessCheck).Methods(http.MethodGet)
	router.HandleFunc("/api/v1/detect", service.handleDetect).Methods(http.MethodPost)

	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.ServerHost, "8082"),
		Handler: router,
	}

	go func() {
		logger.Log.WithFields(map[string]interface{}{
			"host": cfg.ServerHost,
			"port": "8082",
		}).Info("DLP Service started")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Log.WithError(err).Fatal("failed to start server")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Log.Info("Shutting down DLP Service...")
	cancel()

	ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancelShutdown()

	if err := server.Shutdown(ctxShutdown); err != nil {
		logger.Log.WithError(err).Error("server forced to shutdown")
	}

	logger.Log.Info("DLP Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func readinessCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"ready"}`))
}

func (s *DLPService) processEvent(ctx context.Context, event models.Event) error {
	detection := s.detector.Detect(event.Data)
	sanitized := s.detector.Sanitize(event.Data)

	payload := map[string]interface{}{
		"original_event_id": event.ID,
		"phi_detection":     detection,
		"sanitized_data":    sanitized,
		"raw_data":          event.Data,
	}

	return s.producer.PublishEvent(ctx, "sanitize", "dlp-service", payload)
}

func (s *DLPService) handleDetect(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Data map[string]interface{} `json:"data"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}

	detection := s.detector.Detect(req.Data)
	sanitized := s.detector.Sanitize(req.Data)

	response := map[string]interface{}{
		"detection": detection,
		"sanitized": sanitized,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
