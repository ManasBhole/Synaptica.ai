package routes

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/analytics/cohort"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type CohortHandler struct {
	service *cohort.Service
}

func NewCohortHandler(service *cohort.Service) *CohortHandler {
	return &CohortHandler{service: service}
}

func (h *CohortHandler) Register(r *mux.Router) {
	r.HandleFunc("/cohort/query", h.handleQuery).Methods(http.MethodPost)
	r.HandleFunc("/cohort/verify", h.handleVerify).Methods(http.MethodPost)
}

func (h *CohortHandler) handleQuery(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req models.CohortQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid cohort query", http.StatusBadRequest)
		return
	}
	if req.DSL == "" {
		http.Error(w, "dsl is required", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		req.ID = generateCohortID()
	}

	result, err := h.service.Execute(r.Context(), req)
	if err != nil {
		logger.Log.WithError(err).Error("failed to execute cohort query")
		http.Error(w, "failed to execute cohort query", http.StatusBadRequest)
		return
	}

	writeJSON(w, result)
}

func (h *CohortHandler) handleVerify(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var payload struct {
		DSL string `json:"dsl"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if payload.DSL == "" {
		http.Error(w, "dsl is required", http.StatusBadRequest)
		return
	}
	if err := h.service.VerifyDSL(payload.DSL); err != nil {
		logger.Log.WithError(err).Warn("cohort DSL verification failed")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, map[string]string{"status": "ok"})
}

func generateCohortID() string {
	return "cohort-" + time.Now().UTC().Format("20060102-150405.000")
}
