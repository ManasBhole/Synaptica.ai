package routes

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/config"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/gateway/httpclient"
)

type IngestionProxy struct {
	Client *http.Client
	Cfg    *config.Config
}

func RegisterIngestionRoutes(router *mux.Router, proxy *IngestionProxy) {
	if proxy == nil || proxy.Client == nil || proxy.Cfg == nil {
		panic("ingestion proxy requires client and config")
	}

	router.HandleFunc("/ingest", proxy.handleIngest).Methods(http.MethodPost)
	router.HandleFunc("/ingest/status/{id}", proxy.handleIngestStatus).Methods(http.MethodGet)
}

func (p *IngestionProxy) handleIngest(w http.ResponseWriter, r *http.Request) {
	var req models.IngestRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, p.Cfg.MaxRequestBody)).Decode(&req); err != nil {
		logger.Log.WithError(err).Error("Failed to decode request")
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ingestionServiceURL := fmt.Sprintf("%s/api/v1/ingest", p.Cfg.IngestionBaseURL)

	resp, status, err := p.forwardRequest(r, http.MethodPost, ingestionServiceURL, req)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to forward to ingestion service")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (p *IngestionProxy) handleIngestStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	ingestionServiceURL := fmt.Sprintf("%s/api/v1/ingest/status/%s", p.Cfg.IngestionBaseURL, id)

	resp, status, err := p.forwardRequest(r, http.MethodGet, ingestionServiceURL, nil)
	if err != nil {
		logger.Log.WithError(err).Error("Failed to forward to ingestion service")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(resp)
}

func (p *IngestionProxy) forwardRequest(r *http.Request, method, url string, body interface{}) (interface{}, int, error) {
	// Correlation ID
	corrID := r.Header.Get("X-Request-ID")
	if corrID == "" {
		corrID = uuid.New().String()
	}

	// Prepare JSON body
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, 0, err
		}
		reqBody = bytes.NewBuffer(b)
	}

	ctx, cancel := context.WithTimeout(r.Context(), p.Cfg.GatewayRequestTimeout)
	defer cancel()
	outReq, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, 0, err
	}

	outReq.Header = make(http.Header)
	for k, v := range r.Header {
		outReq.Header[k] = v
	}
	outReq.Header.Set("Content-Type", "application/json")
	outReq.Header.Set("X-Request-ID", corrID)

	var resp *http.Response
	reqErr := httpclient.Retry(ctx, 3, 200*time.Millisecond, func() error {
		var doErr error
		resp, doErr = p.Client.Do(outReq)
		if doErr != nil && httpclient.IsRetriable(doErr) {
			return doErr
		}
		return doErr
	})
	if reqErr != nil {
		return nil, 0, reqErr
	}
	defer resp.Body.Close()

	var out interface{}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		out = map[string]interface{}{"status": resp.Status}
	}

	logger.Log.WithFields(map[string]interface{}{
		"url":        url,
		"status":     resp.StatusCode,
		"request_id": corrID,
	}).Info("Forwarded request to ingestion service")

	return out, resp.StatusCode, nil
}
