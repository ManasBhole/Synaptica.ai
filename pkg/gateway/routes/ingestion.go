package routes

import (
    "bytes"
    "context"
    "encoding/json"
    "io"
    "net/http"
    "os"
    "time"

    "github.com/google/uuid"
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

    // Forward to ingestion service (configurable via env)
    ingestionBase := os.Getenv("INGESTION_BASE_URL")
    if ingestionBase == "" { ingestionBase = "http://localhost:8081" }
    ingestionServiceURL := ingestionBase + "/api/v1/ingest"

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

    ingestionBase := os.Getenv("INGESTION_BASE_URL")
    if ingestionBase == "" { ingestionBase = "http://localhost:8081" }
    ingestionServiceURL := ingestionBase + "/api/v1/ingest/status/" + id

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
    // Correlation ID
    corrID := r.Header.Get("X-Request-ID")
    if corrID == "" { corrID = uuid.New().String() }

    // Prepare JSON body
    var reqBody io.Reader
    if body != nil {
        b, err := json.Marshal(body)
        if err != nil { return nil, err }
        reqBody = bytes.NewBuffer(b)
    }

    // New request with context timeout
    ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
    defer cancel()
    outReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, reqBody)
    if err != nil { return nil, err }

    // Copy headers and set correlation id
    outReq.Header = make(http.Header)
    for k, v := range r.Header { outReq.Header[k] = v }
    outReq.Header.Set("Content-Type", "application/json")
    outReq.Header.Set("X-Request-ID", corrID)

    // HTTP client with timeouts
    client := &http.Client{ Timeout: 12 * time.Second }
    resp, err := client.Do(outReq)
    if err != nil { return nil, err }
    defer resp.Body.Close()

    // Read response
    var out interface{}
    if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
        // if non-JSON, return status only
        out = map[string]interface{}{"status": resp.Status}
    }

    logger.Log.WithFields(map[string]interface{}{
        "url": url,
        "status": resp.StatusCode,
        "request_id": corrID,
    }).Info("Forwarded request to ingestion service")

    return out, nil
}

