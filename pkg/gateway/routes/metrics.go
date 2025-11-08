package routes

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"gorm.io/gorm"
)

type MetricsHandler struct {
	db *gorm.DB
}

type OverviewMetrics struct {
	GatewayLatencyMs     float64 `json:"gatewayLatencyMs"`
	IngestionThroughput  int     `json:"ingestionThroughput"`
	KafkaLag             int     `json:"kafkaLag"`
	PIIDetectedToday     int     `json:"piiDetectedToday"`
	TrainingJobsActive   int     `json:"trainingJobsActive"`
	PredictionsPerMinute int     `json:"predictionsPerMinute"`
}

type PipelineStatus struct {
	ID        string    `json:"id"`
	Stage     string    `json:"stage"`
	Status    string    `json:"status"`
	UpdatedAt time.Time `json:"updatedAt"`
	Details   string    `json:"details"`
}

func NewMetricsHandler(db *gorm.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

func (h *MetricsHandler) Register(r *mux.Router) {
	r.HandleFunc("/metrics/overview", h.handleOverview).Methods(http.MethodGet)
	r.HandleFunc("/pipelines/status", h.handlePipelineStatus).Methods(http.MethodGet)
	r.HandleFunc("/metrics/prediction-latency", h.handlePredictionLatency).Methods(http.MethodGet)
	r.HandleFunc("/training/jobs", h.handleTrainingJobs).Methods(http.MethodGet)
}

func (h *MetricsHandler) handleOverview(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.collectMetrics()
	if err != nil {
		logger.Log.WithError(err).Error("failed to collect metrics")
		http.Error(w, "failed to collect metrics", http.StatusInternalServerError)
		return
	}

	writeJSON(w, metrics)
}

func (h *MetricsHandler) handlePipelineStatus(w http.ResponseWriter, r *http.Request) {
	metrics, err := h.collectMetrics()
	if err != nil {
		logger.Log.WithError(err).Error("failed to collect pipeline status")
		http.Error(w, "failed to collect pipeline status", http.StatusInternalServerError)
		return
	}

	now := time.Now().UTC()
	statuses := []PipelineStatus{
		{
			ID:        "ingestion",
			Stage:     "API Gateway ➝ Ingestion",
			Status:    deriveStatus(metrics.KafkaLag < 5, metrics.IngestionThroughput > 0),
			UpdatedAt: now,
			Details:   formatDetails("%d msgs/min • backlog %d", metrics.IngestionThroughput, metrics.KafkaLag),
		},
		{
			ID:        "privacy",
			Stage:     "DLP ➝ De-ID",
			Status:    deriveStatus(metrics.PIIDetectedToday < 25, true),
			UpdatedAt: now,
			Details:   formatDetails("%d PII alerts today", metrics.PIIDetectedToday),
		},
		{
			ID:        "ai-normalizer",
			Stage:     "Normalizer ➝ Linkage ➝ Storage",
			Status:    deriveStatus(metrics.PredictionsPerMinute > 0, metrics.TrainingJobsActive >= 0),
			UpdatedAt: now,
			Details:   formatDetails("%d predictions/min • %d jobs active", metrics.PredictionsPerMinute, metrics.TrainingJobsActive),
		},
	}

	writeJSON(w, statuses)
}

func (h *MetricsHandler) collectMetrics() (OverviewMetrics, error) {
	metrics := OverviewMetrics{}

	var latency sql.NullFloat64
	if err := h.db.Raw(`
		SELECT AVG(EXTRACT(EPOCH FROM updated_at - created_at) * 1000)
		FROM ingestion_requests
		WHERE updated_at > NOW() - INTERVAL '1 hour' AND status = 'published'
	`).Scan(&latency).Error; err != nil {
		return metrics, err
	}
	if latency.Valid {
		metrics.GatewayLatencyMs = latency.Float64
	} else {
		metrics.GatewayLatencyMs = 150
	}

	var throughput sql.NullInt64
	if err := h.db.Raw(`
		SELECT COUNT(*)
		FROM ingestion_requests
		WHERE created_at > NOW() - INTERVAL '1 minute'
	`).Scan(&throughput).Error; err != nil {
		return metrics, err
	}
	if throughput.Valid {
		metrics.IngestionThroughput = int(throughput.Int64)
	}

	var lag sql.NullInt64
	if err := h.db.Raw(`
		SELECT COUNT(*)
		FROM ingestion_requests
		WHERE status <> 'published'
	`).Scan(&lag).Error; err != nil {
		return metrics, err
	}
	if lag.Valid {
		metrics.KafkaLag = int(lag.Int64)
	}

	var pii sql.NullInt64
	if err := h.db.Raw(`
		SELECT COUNT(*)
		FROM ingestion_requests
		WHERE status = 'failed' AND DATE(updated_at) = CURRENT_DATE
	`).Scan(&pii).Error; err != nil {
		return metrics, err
	}
	if pii.Valid {
		metrics.PIIDetectedToday = int(pii.Int64)
	}

	var training sql.NullInt64
	if err := h.db.Raw(`
		SELECT COUNT(*)
		FROM training_jobs
		WHERE status IN ('queued', 'running')
	`).Scan(&training).Error; err != nil {
		return metrics, err
	}
	if training.Valid {
		metrics.TrainingJobsActive = int(training.Int64)
	}

	var predictions sql.NullInt64
	if err := h.db.Raw(`
		SELECT COUNT(*)
		FROM lakehouse_facts
		WHERE timestamp > NOW() - INTERVAL '1 minute'
	`).Scan(&predictions).Error; err != nil {
		return metrics, err
	}
	if predictions.Valid {
		metrics.PredictionsPerMinute = int(predictions.Int64)
	}

	return metrics, nil
}

func deriveStatus(conditionA, conditionB bool) string {
	switch {
	case conditionA && conditionB:
		return "healthy"
	case conditionA || conditionB:
		return "degraded"
	default:
		return "failing"
	}
}

func formatDetails(format string, args ...interface{}) string {
	return fmt.Sprintf(format, args...)
}

func writeJSON(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		logger.Log.WithError(err).Error("failed to write json response")
	}
}

type PredictionLatencyPoint struct {
	Timestamp time.Time `json:"timestamp"`
	LatencyMs float64   `json:"latencyMs"`
}

func (h *MetricsHandler) handlePredictionLatency(w http.ResponseWriter, r *http.Request) {
	var rows []struct {
		Timestamp time.Time       `gorm:"column:bucket"`
		Latency   sql.NullFloat64 `gorm:"column:latency_ms"`
	}

	if err := h.db.WithContext(r.Context()).Raw(`
		SELECT
			date_trunc('minute', created_at) AS bucket,
			AVG(EXTRACT(EPOCH FROM created_at - timestamp) * 1000) AS latency_ms
		FROM lakehouse_facts
		WHERE created_at > NOW() - INTERVAL '2 hour'
		GROUP BY bucket
		ORDER BY bucket ASC
	`).Scan(&rows).Error; err != nil {
		logger.Log.WithError(err).Error("failed to load prediction latency")
		http.Error(w, "failed to load prediction latency", http.StatusInternalServerError)
		return
	}

	points := make([]PredictionLatencyPoint, 0, len(rows))
	for _, row := range rows {
		latency := 0.0
		if row.Latency.Valid {
			latency = row.Latency.Float64
		}
		points = append(points, PredictionLatencyPoint{
			Timestamp: row.Timestamp,
			LatencyMs: latency,
		})
	}

	writeJSON(w, points)
}

type TrainingJobSummary struct {
	ID          string     `json:"id"`
	ModelType   string     `json:"modelType"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"createdAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Accuracy    *float64   `json:"accuracy,omitempty"`
	Loss        *float64   `json:"loss,omitempty"`
}

func (h *MetricsHandler) handleTrainingJobs(w http.ResponseWriter, r *http.Request) {
	limit := 10
	if val := r.URL.Query().Get("limit"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 && parsed <= 50 {
			limit = parsed
		}
	}

	var rows []struct {
		ID          string          `gorm:"column:id"`
		ModelType   string          `gorm:"column:model_type"`
		Status      string          `gorm:"column:status"`
		CreatedAt   time.Time       `gorm:"column:created_at"`
		CompletedAt *time.Time      `gorm:"column:completed_at"`
		Accuracy    sql.NullFloat64 `gorm:"column:accuracy"`
		Loss        sql.NullFloat64 `gorm:"column:loss"`
	}

	if err := h.db.WithContext(r.Context()).Raw(`
		SELECT
			id,
			model_type,
			status,
			created_at,
			completed_at,
			NULLIF((metrics ->> 'accuracy'), '')::DOUBLE PRECISION AS accuracy,
			NULLIF((metrics ->> 'loss'), '')::DOUBLE PRECISION AS loss
		FROM training_jobs
		ORDER BY created_at DESC
		LIMIT ?
	`, limit).Scan(&rows).Error; err != nil {
		logger.Log.WithError(err).Error("failed to list training jobs")
		http.Error(w, "failed to list training jobs", http.StatusInternalServerError)
		return
	}

	jobs := make([]TrainingJobSummary, 0, len(rows))
	for _, row := range rows {
		var accPtr *float64
		if row.Accuracy.Valid {
			v := row.Accuracy.Float64
			accPtr = &v
		}
		var lossPtr *float64
		if row.Loss.Valid {
			v := row.Loss.Float64
			lossPtr = &v
		}

		jobs = append(jobs, TrainingJobSummary{
			ID:          row.ID,
			ModelType:   row.ModelType,
			Status:      row.Status,
			CreatedAt:   row.CreatedAt,
			CompletedAt: row.CompletedAt,
			Accuracy:    accPtr,
			Loss:        lossPtr,
		})
	}

	writeJSON(w, map[string]interface{}{"jobs": jobs})
}
