package edc

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Register(r *mux.Router) {
	r.HandleFunc("/studies", h.handleCreateStudy).Methods(http.MethodPost)
	r.HandleFunc("/studies", h.handleListStudies).Methods(http.MethodGet)
	r.HandleFunc("/studies/{id}", h.handleGetStudy).Methods(http.MethodGet)
	r.HandleFunc("/studies/{id}/status", h.handleUpdateStudyStatus).Methods(http.MethodPatch)
	r.HandleFunc("/studies/{id}/sites", h.handleCreateSite).Methods(http.MethodPost)
	r.HandleFunc("/studies/{id}/forms", h.handleCreateForm).Methods(http.MethodPost)
	r.HandleFunc("/studies/{id}/visits", h.handleCreateVisitTemplate).Methods(http.MethodPost)
	r.HandleFunc("/studies/{id}/subjects", h.handleEnrollSubject).Methods(http.MethodPost)
	r.HandleFunc("/studies/{id}/subjects", h.handleListSubjects).Methods(http.MethodGet)
	r.HandleFunc("/studies/{id}/consents", h.handleCreateConsentVersion).Methods(http.MethodPost)
	r.HandleFunc("/studies/{id}/consents", h.handleListConsentVersions).Methods(http.MethodGet)
	r.HandleFunc("/studies/{id}/audit", h.handleListAuditLogs).Methods(http.MethodGet)
	r.HandleFunc("/subjects/{id}/consents", h.handleRecordConsent).Methods(http.MethodPost)
}

func (h *Handler) handleCreateStudy(w http.ResponseWriter, r *http.Request) {
	var req models.CreateStudyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Code == "" || req.Name == "" {
		http.Error(w, "code and name are required", http.StatusBadRequest)
		return
	}
	study, err := h.service.CreateStudy(r.Context(), req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to create study")
		http.Error(w, "failed to create study", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"study": study})
}

func (h *Handler) handleListStudies(w http.ResponseWriter, r *http.Request) {
	limit := parseLimit(r, 50)
	studies, err := h.service.ListStudies(r.Context(), limit)
	if err != nil {
		logger.Log.WithError(err).Error("failed to list studies")
		http.Error(w, "failed to list studies", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": studies})
}

func (h *Handler) handleGetStudy(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	study, err := h.service.GetStudy(r.Context(), id)
	if err != nil {
		logger.Log.WithError(err).Error("failed to get study")
		http.Error(w, "study not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"study": study})
}

func (h *Handler) handleUpdateStudyStatus(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	var payload models.UpdateStudyStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if payload.Status == "" {
		http.Error(w, "status is required", http.StatusBadRequest)
		return
	}
	if err := h.service.UpdateStudyStatus(r.Context(), id, payload.Status, resolveActor(r)); err != nil {
		logger.Log.WithError(err).Error("failed to update study status")
		http.Error(w, "failed to update status", http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) handleCreateSite(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	var req models.CreateStudySiteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.SiteCode == "" || req.Name == "" {
		http.Error(w, "site_code and name are required", http.StatusBadRequest)
		return
	}
	site, err := h.service.CreateSite(r.Context(), studyID, req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to create site")
		http.Error(w, "failed to create site", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"site": site})
}

func (h *Handler) handleCreateForm(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	var req models.CreateStudyFormRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" || req.Slug == "" || req.Schema == nil {
		http.Error(w, "name, slug, and schema are required", http.StatusBadRequest)
		return
	}
	form, err := h.service.CreateForm(r.Context(), studyID, req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to create form")
		http.Error(w, "failed to create form", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"form": form})
}

func (h *Handler) handleCreateVisitTemplate(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	var req models.CreateVisitTemplateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	visit, err := h.service.CreateVisitTemplate(r.Context(), studyID, req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to create visit template")
		http.Error(w, "failed to create visit", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"visit": visit})
}

func (h *Handler) handleEnrollSubject(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	var req models.EnrollSubjectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.SubjectCode == "" {
		http.Error(w, "subject_code is required", http.StatusBadRequest)
		return
	}
	subject, err := h.service.EnrollSubject(r.Context(), studyID, req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to enroll subject")
		http.Error(w, "failed to enroll subject", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"subject": subject})
}

func (h *Handler) handleListSubjects(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	limit := parseLimit(r, 50)
	subjects, err := h.service.ListSubjects(r.Context(), studyID, limit)
	if err != nil {
		logger.Log.WithError(err).Error("failed to list subjects")
		http.Error(w, "failed to list subjects", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": subjects})
}

func (h *Handler) handleCreateConsentVersion(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	var req models.CreateConsentVersionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.Version == "" {
		http.Error(w, "version is required", http.StatusBadRequest)
		return
	}
	version, err := h.service.CreateConsentVersion(r.Context(), studyID, req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to create consent version")
		http.Error(w, "failed to create consent version", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"version": version})
}

func (h *Handler) handleListConsentVersions(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	limit := parseLimit(r, 50)
	versions, err := h.service.ListConsentVersions(r.Context(), studyID, limit)
	if err != nil {
		logger.Log.WithError(err).Error("failed to list consent versions")
		http.Error(w, "failed to list consent versions", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": versions})
}

func (h *Handler) handleRecordConsent(w http.ResponseWriter, r *http.Request) {
	subjectID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid subject id", http.StatusBadRequest)
		return
	}
	var req models.ConsentSignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request", http.StatusBadRequest)
		return
	}
	if req.ConsentVersionID == uuid.Nil {
		http.Error(w, "consent_version_id is required", http.StatusBadRequest)
		return
	}
	signature, err := h.service.RecordConsent(r.Context(), subjectID, req, resolveActor(r))
	if err != nil {
		logger.Log.WithError(err).Error("failed to record consent")
		http.Error(w, "failed to record consent", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]interface{}{"signature": signature})
}

func (h *Handler) handleListAuditLogs(w http.ResponseWriter, r *http.Request) {
	studyID, err := uuid.Parse(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, "invalid study id", http.StatusBadRequest)
		return
	}
	limit := parseLimit(r, 100)
	logs, err := h.service.ListAuditLogs(r.Context(), studyID, limit)
	if err != nil {
		logger.Log.WithError(err).Error("failed to list audit logs")
		http.Error(w, "failed to list audit logs", http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"items": logs})
}

func parseLimit(r *http.Request, fallback int) int {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return fallback
	}
	if v, err := strconv.Atoi(raw); err == nil && v > 0 {
		return v
	}
	return fallback
}

func resolveActor(r *http.Request) string {
	if r == nil {
		return "system"
	}
	if filters, ok := r.Context().Value("rls_filters").(map[string]interface{}); ok {
		if user, ok := filters["user_id"].(string); ok && user != "" {
			return user
		}
	}
	return "system"
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
