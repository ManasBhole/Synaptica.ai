package ingestion

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
)

type HTTPHandler struct {
	service *Service
	maxBody int64
}

func NewHTTPHandler(service *Service, maxBody int64) *HTTPHandler {
	return &HTTPHandler{service: service, maxBody: maxBody}
}

func (h *HTTPHandler) Register(router *mux.Router) {
	router.HandleFunc("/ingest", h.handleIngest).Methods(http.MethodPost)
	router.HandleFunc("/ingest/status/{id}", h.handleStatus).Methods(http.MethodGet)
}

func (h *HTTPHandler) handleIngest(w http.ResponseWriter, r *http.Request) {
	if h.maxBody > 0 {
		r.Body = http.MaxBytesReader(w, r.Body, h.maxBody)
	}

	var req RequestWrapper
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logger.Log.WithError(err).Warn("invalid ingestion payload")
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	resp, err := h.service.Process(r.Context(), req.ToModel())
	if err != nil {
		if IsValidationError(err) {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "ingestion not found", http.StatusNotFound)
			return
		}
		logger.Log.WithError(err).Error("failed to process ingestion")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(resp)
}

func (h *HTTPHandler) handleStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id := vars["id"]

	rec, err := h.service.Status(r.Context(), id)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			http.Error(w, "ingestion not found", http.StatusNotFound)
			return
		}
		logger.Log.WithError(err).Error("failed to fetch ingestion status")
		http.Error(w, "internal error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(rec)
}
