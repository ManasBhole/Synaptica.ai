package routes

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/synaptica-ai/platform/pkg/analytics/cohort"
	"github.com/synaptica-ai/platform/pkg/common/database"
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
	r.HandleFunc("/cohort/export", h.handleExport).Methods(http.MethodPost)
	r.HandleFunc("/cohort/{id}", h.handleDrilldown).Methods(http.MethodPost)
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
	req.TenantID = resolveTenantID(r.Context())

	redisClient := database.GetRedis()
	cacheKey := ""
	if redisClient != nil {
		cacheKey = buildCohortCacheKey(req.TenantID, req)
		if cached, err := redisClient.Get(r.Context(), cacheKey).Result(); err == nil {
			var cachedResult models.CohortResult
			if unmarshalErr := json.Unmarshal([]byte(cached), &cachedResult); unmarshalErr == nil {
				if cachedResult.Metadata == nil {
					cachedResult.Metadata = make(map[string]interface{})
				}
				cachedResult.Metadata["cacheHit"] = true
				writeJSON(w, cachedResult)
				return
			}
		} else if !errors.Is(err, redis.Nil) {
			logger.Log.WithError(err).Warn("cohort cache lookup failed")
		}
	}

	result, err := h.service.Execute(r.Context(), req)
	if err != nil {
		logger.Log.WithError(err).Error("failed to execute cohort query")
		http.Error(w, "failed to execute cohort query", http.StatusBadRequest)
		return
	}

	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	result.Metadata["cacheHit"] = false

	if redisClient != nil && cacheKey != "" {
		payload, err := json.Marshal(result)
		if err != nil {
			logger.Log.WithError(err).Warn("failed to marshal cohort result for cache")
		} else {
			if err := redisClient.Set(r.Context(), cacheKey, payload, 2*time.Minute).Err(); err != nil {
				logger.Log.WithError(err).Warn("failed to store cohort cache entry")
			}
		}
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

func resolveTenantID(ctx context.Context) string {
	if ctx == nil {
		return "public"
	}
	if filters, ok := ctx.Value("rls_filters").(map[string]interface{}); ok {
		if tenant, ok := filters["tenant_id"].(string); ok && tenant != "" {
			return tenant
		}
		if user, ok := filters["user_id"].(string); ok && user != "" {
			return user
		}
	}
	return "public"
}

func buildCohortCacheKey(tenant string, query models.CohortQuery) string {
	fields := append([]string(nil), query.Fields...)
	sort.Strings(fields)
	payload := struct {
		Tenant string   `json:"tenant"`
		DSL    string   `json:"dsl"`
		Limit  int      `json:"limit"`
		Fields []string `json:"fields"`
	}{
		Tenant: tenant,
		DSL:    query.DSL,
		Limit:  query.Limit,
		Fields: fields,
	}

	bytes, _ := json.Marshal(payload)
	hash := sha1.Sum(bytes)
	if tenant == "" {
		tenant = "public"
	}
	return "cohort:" + tenant + ":" + hex.EncodeToString(hash[:])
}

func (h *CohortHandler) handleExport(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	var req models.CohortQuery
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid cohort query", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.DSL) == "" {
		http.Error(w, "dsl is required", http.StatusBadRequest)
		return
	}
	if req.ID == "" {
		req.ID = generateCohortID()
	}
	req.TenantID = resolveTenantID(r.Context())

	filename := sanitizeFilename(req.ID)
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s.csv\"", filename))

	if err := h.service.Export(r.Context(), req, w); err != nil {
		logger.Log.WithError(err).Error("failed to export cohort")
	}
}

func sanitizeFilename(name string) string {
	if strings.TrimSpace(name) == "" {
		return "cohort-export"
	}
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.ReplaceAll(name, " ", "-")
	builder := strings.Builder{}
	for _, r := range name {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			builder.WriteRune(r)
			continue
		}
		builder.WriteRune('-')
	}
	sanitized := builder.String()
	if sanitized == "" {
		return "cohort-export"
	}
	return sanitized
}

func (h *CohortHandler) handleDrilldown(w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()
	vars := mux.Vars(r)
	cohortID := vars["id"]
	if cohortID == "" {
		http.Error(w, "cohort id is required", http.StatusBadRequest)
		return
	}

	var req models.CohortDrilldownRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.CohortID = cohortID
	req.TenantID = resolveTenantID(r.Context())

	result, err := h.service.Drilldown(r.Context(), req)
	if err != nil {
		logger.Log.WithError(err).Error("failed to drilldown cohort")
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	writeJSON(w, result)
}
