package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/database"
	"github.com/synaptica-ai/platform/pkg/common/kafka"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type LinkageService struct {
	producer *kafka.Producer
	consumer *kafka.Consumer
	db       interface{} // For storing master patient IDs
}

func main() {
	logger.Init()
	cfg := config.Load()

	service := &LinkageService{}

	// Initialize database
	_, err := database.GetPostgres()
	if err != nil {
		logger.Log.WithError(err).Fatal("Failed to connect to database")
	}

	// Kafka producer
	service.producer = kafka.NewProducer("linked-events")
	defer service.producer.Close()

	// Kafka consumer
	service.consumer = kafka.NewConsumer("normalized-events", "linkage-service")
	defer service.consumer.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := service.consumer.Consume(ctx, service.processEvent); err != nil {
			logger.Log.WithError(err).Fatal("Consumer error")
		}
	}()

	// HTTP server
	router := mux.NewRouter()
	router.HandleFunc("/health", healthCheck).Methods("GET")
	router.HandleFunc("/api/v1/link", service.handleLink).Methods("POST")

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
			logger.Log.WithError(err).Fatal("Failed to start server")
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
		logger.Log.WithError(err).Error("Server forced to shutdown")
	}

	logger.Log.Info("Linkage Service stopped")
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"healthy"}`))
}

func (s *LinkageService) processEvent(ctx context.Context, event models.Event) error {
	logger.Log.WithFields(map[string]interface{}{
		"event_id": event.ID,
	}).Info("Processing event for record linkage")

	// Extract normalized record
	normalizedData, ok := event.Data["normalized"].(map[string]interface{})
	if !ok {
		logger.Log.Warn("No normalized data found")
		return nil
	}

	// Convert to NormalizedRecord
	record := models.NormalizedRecord{
		ID:          getString(normalizedData, "id"),
		PatientID:   getString(normalizedData, "patient_id"),
		ResourceType: getString(normalizedData, "resource_type"),
		Timestamp:   time.Now(),
	}

	// Perform linkage
	result := s.linkRecord(record)

	// Publish linked event to multiple downstream systems
	go func() {
		// To Lakehouse (immutable facts)
		s.producer.PublishEvent(ctx, "downstream", "lakehouse", map[string]interface{}{
			"type": "immutable_facts",
			"linkage": result,
			"record": record,
		})

		// To RT OLAP (denormalized facts/rollups)
		s.producer.PublishEvent(ctx, "downstream", "rt-olap", map[string]interface{}{
			"type": "denormalized_facts",
			"linkage": result,
			"record": record,
		})

		// To OLTP (consents/ids)
		s.producer.PublishEvent(ctx, "downstream", "oltp", map[string]interface{}{
			"type": "consents_ids",
			"linkage": result,
		})
	}()

	return nil
}

func (s *LinkageService) handleLink(w http.ResponseWriter, r *http.Request) {
	var req models.LinkageRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	var results []models.LinkageResult
	for _, record := range req.Records {
		result := s.linkRecord(record)
		results = append(results, result)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

func (s *LinkageService) linkRecord(record models.NormalizedRecord) models.LinkageResult {
	// Deterministic matching
	if masterID := s.deterministicMatch(record); masterID != "" {
		return models.LinkageResult{
			MasterPatientID: masterID,
			LinkedIDs:       []string{record.PatientID},
			Confidence:      1.0,
			Method:          "deterministic",
		}
	}

	// Probabilistic matching
	masterID, confidence := s.probabilisticMatch(record)
	if masterID != "" {
		return models.LinkageResult{
			MasterPatientID: masterID,
			LinkedIDs:       []string{record.PatientID},
			Confidence:      confidence,
			Method:          "probabilistic",
		}
	}

	// Create new master patient ID
	masterID = uuid.New().String()
	return models.LinkageResult{
		MasterPatientID: masterID,
		LinkedIDs:       []string{record.PatientID},
		Confidence:      1.0,
		Method:          "new",
	}
}

func (s *LinkageService) deterministicMatch(record models.NormalizedRecord) string {
	// Exact match on patient identifiers
	// In production, would query database for exact matches
	// For now, return empty (no match found)
	return ""
}

func (s *LinkageService) probabilisticMatch(record models.NormalizedRecord) (string, float64) {
	// Fuzzy matching using similarity algorithms (Jaro-Winkler, Levenshtein, etc.)
	// In production, would use sophisticated matching algorithms

	// Simplified: check if patient ID is similar to existing ones
	// This is a placeholder - real implementation would use proper similarity algorithms
	similarity := 0.85 // Placeholder

	if similarity > 0.8 {
		return uuid.New().String(), similarity
	}

	return "", 0.0
}

// Helper functions
func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

// String similarity functions (simplified)
func jaroWinkler(s1, s2 string) float64 {
	if s1 == s2 {
		return 1.0
	}

	// Simplified Jaro-Winkler
	commonChars := 0
	for _, c1 := range s1 {
		if strings.ContainsRune(s2, c1) {
			commonChars++
		}
	}

	if len(s1) == 0 || len(s2) == 0 {
		return 0.0
	}

	jaro := float64(commonChars) / float64(max(len(s1), len(s2)))

	// Winkler modification
	prefix := 0
	minLen := min(len(s1), len(s2))
	for i := 0; i < minLen && i < 4; i++ {
		if s1[i] == s2[i] {
			prefix++
		} else {
			break
		}
	}

	return jaro + float64(prefix)*0.1*(1-jaro)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func sortStrings(strs []string) {
	sort.Strings(strs)
}

