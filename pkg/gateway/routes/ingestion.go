package routes

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

func RegisterIngestionRoutes(router *mux.Router) {
	router.HandleFunc("/ingest", handleIngest).Methods("POST")
	router.HandleFunc("/ingest/status/{id}", handleIngestStatus).Methods("GET")
}

func handleIngest(w http.ResponseWriter, r *http.Request) {
	var req models.IngestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Forward to ingestion service
	// In production, use service discovery or load balancer
	ingestionServiceURL := "http://localhost:8081/api/v1/ingest"

	resp, err := forwardRequest(r, ingestionServiceURL, req)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to forward to ingestion service")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func handleIngestStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	// Forward to ingestion service
	ingestionServiceURL := "http://localhost:8081/api/v1/ingest/status/" + id

	resp, err := forwardRequest(r, ingestionServiceURL, nil)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to forward to ingestion service")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func forwardRequest(r *http.Request, url string, body interface{}) (interface{}, error) {
	// Placeholder - in production, use proper HTTP client with retries, timeouts
	logger.Log.WithField("url", url).Debug("Forwarding request")
	return map[string]interface{}{"status": "forwarded"}, nil
}

