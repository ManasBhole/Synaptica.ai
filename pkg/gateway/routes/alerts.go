package routes

import (
	"encoding/json"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"gorm.io/gorm"
)

type AlertsHandler struct {
	db *gorm.DB
}

type Alert struct {
	ID        string                 `json:"id"`
	Source    string                 `json:"source"`
	Format    string                 `json:"format"`
	Status    string                 `json:"status"`
	Error     string                 `json:"error"`
	Payload   map[string]interface{} `json:"payload"`
	UpdatedAt string                 `json:"updatedAt"`
}

type AlertSummary struct {
	Critical int `json:"critical"`
	Warning  int `json:"warning"`
	Info     int `json:"info"`
}

type AlertsResponse struct {
	Summary AlertSummary `json:"summary"`
	Items   []Alert      `json:"items"`
}

func NewAlertsHandler(db *gorm.DB) *AlertsHandler {
	return &AlertsHandler{db: db}
}

func (h *AlertsHandler) Register(r *mux.Router) {
	r.HandleFunc("/alerts", h.handleList).Methods(http.MethodGet)
}

func (h *AlertsHandler) handleList(w http.ResponseWriter, r *http.Request) {
	summary := AlertSummary{}
	if err := h.db.Raw(`
		SELECT
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS critical,
			SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) AS warning,
			SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END) AS info
		FROM ingestion_requests
		WHERE updated_at > NOW() - INTERVAL '7 days'
	`).Scan(&summary).Error; err != nil {
		logger.Log.WithError(err).Error("failed to summarize alerts")
		http.Error(w, "failed to fetch alerts", http.StatusInternalServerError)
		return
	}

	var rows []struct {
		ID        string `gorm:"column:id"`
		Source    string `gorm:"column:source"`
		Format    string `gorm:"column:format"`
		Status    string `gorm:"column:status"`
		Error     string `gorm:"column:error"`
		Payload   []byte `gorm:"column:payload"`
		UpdatedAt string `gorm:"column:updated_at"`
	}

	if err := h.db.Raw(`
		SELECT id, source, format, status, COALESCE(error, '') AS error, payload, TO_CHAR(updated_at, 'YYYY-MM-DD"T"HH24:MI:SSZ') AS updated_at
		FROM ingestion_requests
		WHERE status IN ('failed', 'accepted')
		ORDER BY updated_at DESC
		LIMIT 25
	`).Scan(&rows).Error; err != nil {
		logger.Log.WithError(err).Error("failed to load alert rows")
		http.Error(w, "failed to fetch alerts", http.StatusInternalServerError)
		return
	}

	items := make([]Alert, 0, len(rows))
	for _, row := range rows {
		payloadMap := map[string]interface{}{}
		if len(row.Payload) > 0 {
			if err := json.Unmarshal(row.Payload, &payloadMap); err != nil {
				payloadMap = map[string]interface{}{"raw": string(row.Payload)}
			}
		}

		items = append(items, Alert{
			ID:        row.ID,
			Source:    row.Source,
			Format:    row.Format,
			Status:    row.Status,
			Error:     row.Error,
			Payload:   payloadMap,
			UpdatedAt: row.UpdatedAt,
		})
	}

	writeJSON(w, AlertsResponse{Summary: summary, Items: items})
}
