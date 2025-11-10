package routes

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/observability/metrics"
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

type PipelineActivitySummary struct {
	Accepted         int `json:"accepted"`
	Published        int `json:"published"`
	Failed           int `json:"failed"`
	DLQ              int `json:"dlq"`
	Backlog          int `json:"backlog"`
	ThroughputPerMin int `json:"throughputPerMin"`
}

type PipelineEvent struct {
	ID          string     `json:"id"`
	Source      string     `json:"source"`
	Format      string     `json:"format"`
	Status      string     `json:"status"`
	Error       string     `json:"error,omitempty"`
	RetryCount  int        `json:"retryCount"`
	LastAttempt *time.Time `json:"lastAttempt,omitempty"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
}

type PipelineActivity struct {
	Summary PipelineActivitySummary `json:"summary"`
	Events  []PipelineEvent         `json:"events"`
}

type DLPStats struct {
	TodayFailed     int           `json:"todayFailed"`
	TodayAccepted   int           `json:"todayAccepted"`
	TokenVaultSize  int           `json:"tokenVaultSize"`
	TopReasons      []ReasonCount `json:"topReasons"`
	RecentIncidents []DLPIncident `json:"recentIncidents"`
}

type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

type DLPIncident struct {
	ID         string    `json:"id"`
	Source     string    `json:"source"`
	Format     string    `json:"format"`
	Status     string    `json:"status"`
	Error      string    `json:"error"`
	UpdatedAt  time.Time `json:"updatedAt"`
	CreatedAt  time.Time `json:"createdAt"`
	RetryCount int       `json:"retryCount"`
}

func NewMetricsHandler(db *gorm.DB) *MetricsHandler {
	return &MetricsHandler{db: db}
}

func (h *MetricsHandler) Register(r *mux.Router) {
	r.HandleFunc("/metrics/overview", h.handleOverview).Methods(http.MethodGet)
	r.HandleFunc("/pipelines/status", h.handlePipelineStatus).Methods(http.MethodGet)
	r.HandleFunc("/pipelines/activity", h.handlePipelineActivity).Methods(http.MethodGet)
	r.HandleFunc("/metrics/dlp", h.handleDLPStats).Methods(http.MethodGet)
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

func (h *MetricsHandler) handlePipelineActivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	summary, err := h.collectPipelineSummary(ctx)
	if err != nil {
		logger.Log.WithError(err).Error("failed to collect pipeline summary")
		http.Error(w, "failed to collect pipeline summary", http.StatusInternalServerError)
		return
	}

	metrics.ObservePipelineCounts(summary.Accepted, summary.Published, summary.Failed, summary.Backlog, summary.ThroughputPerMin)

	events, err := h.fetchRecentIngestion(ctx, 25)
	if err != nil {
		logger.Log.WithError(err).Error("failed to load recent ingestion events")
		http.Error(w, "failed to load pipeline activity", http.StatusInternalServerError)
		return
	}

	writeJSON(w, PipelineActivity{Summary: summary, Events: events})
}

func (h *MetricsHandler) collectPipelineSummary(ctx context.Context) (PipelineActivitySummary, error) {
	summary := PipelineActivitySummary{}

	var counts struct {
		Accepted  sql.NullInt64
		Published sql.NullInt64
		Failed    sql.NullInt64
	}

	if err := h.db.WithContext(ctx).Raw(`
		SELECT
			SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) AS accepted,
			SUM(CASE WHEN status = 'published' THEN 1 ELSE 0 END) AS published,
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed
		FROM ingestion_requests
		WHERE created_at > NOW() - INTERVAL '1 hour'
	`).Scan(&counts).Error; err != nil {
		return summary, err
	}

	if counts.Accepted.Valid {
		summary.Accepted = int(counts.Accepted.Int64)
	}
	if counts.Published.Valid {
		summary.Published = int(counts.Published.Int64)
	}
	if counts.Failed.Valid {
		summary.Failed = int(counts.Failed.Int64)
		summary.DLQ = summary.Failed
	}

	var backlog sql.NullInt64
	if err := h.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM ingestion_requests
		WHERE status <> 'published'
	`).Scan(&backlog).Error; err != nil {
		return summary, err
	}
	if backlog.Valid {
		summary.Backlog = int(backlog.Int64)
	}

	var throughput sql.NullInt64
	if err := h.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM ingestion_requests
		WHERE created_at > NOW() - INTERVAL '1 minute'
	`).Scan(&throughput).Error; err != nil {
		return summary, err
	}
	if throughput.Valid {
		summary.ThroughputPerMin = int(throughput.Int64)
	}

	return summary, nil
}

func (h *MetricsHandler) fetchRecentIngestion(ctx context.Context, limit int) ([]PipelineEvent, error) {
	if limit <= 0 {
		limit = 25
	}

	rows := []struct {
		ID          string     `gorm:"column:id"`
		Source      string     `gorm:"column:source"`
		Format      string     `gorm:"column:format"`
		Status      string     `gorm:"column:status"`
		Error       string     `gorm:"column:error"`
		RetryCount  int        `gorm:"column:retry_count"`
		LastAttempt *time.Time `gorm:"column:last_attempt"`
		CreatedAt   time.Time  `gorm:"column:created_at"`
		UpdatedAt   time.Time  `gorm:"column:updated_at"`
	}{}

	if err := h.db.WithContext(ctx).Raw(`
		SELECT id, source, format, status, COALESCE(error, '') AS error, retry_count, last_attempt, created_at, updated_at
		FROM ingestion_requests
		ORDER BY created_at DESC
		LIMIT ?
	`, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	events := make([]PipelineEvent, 0, len(rows))
	for _, row := range rows {
		// treat empty string error as omitted
		event := PipelineEvent{
			ID:         row.ID,
			Source:     row.Source,
			Format:     row.Format,
			Status:     row.Status,
			RetryCount: row.RetryCount,
			CreatedAt:  row.CreatedAt,
			UpdatedAt:  row.UpdatedAt,
		}
		if row.LastAttempt != nil {
			event.LastAttempt = row.LastAttempt
		}
		if strings.TrimSpace(row.Error) != "" {
			event.Error = row.Error
		}
		events = append(events, event)
	}
	return events, nil
}

func (h *MetricsHandler) handleDLPStats(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	stats, err := h.collectDLPStats(ctx)
	if err != nil {
		logger.Log.WithError(err).Error("failed to collect dlp stats")
		http.Error(w, "failed to collect dlp stats", http.StatusInternalServerError)
		return
	}

	metrics.ObserveDLPCounts(stats.TodayFailed, stats.TodayAccepted, stats.TokenVaultSize)

	incidents, err := h.fetchRecentDLPIncidents(ctx, 20)
	if err != nil {
		logger.Log.WithError(err).Error("failed to fetch dlp incidents")
		http.Error(w, "failed to fetch dlp incidents", http.StatusInternalServerError)
		return
	}

	stats.RecentIncidents = incidents
	writeJSON(w, stats)
}

func (h *MetricsHandler) collectDLPStats(ctx context.Context) (DLPStats, error) {
	stats := DLPStats{}

	if err := h.db.WithContext(ctx).Raw(`
		SELECT
			SUM(CASE WHEN status = 'failed' THEN 1 ELSE 0 END) AS failed,
			SUM(CASE WHEN status = 'accepted' THEN 1 ELSE 0 END) AS accepted
		FROM ingestion_requests
		WHERE DATE(created_at) = CURRENT_DATE
	`).Row().Scan(&stats.TodayFailed, &stats.TodayAccepted); err != nil {
		return stats, err
	}

	if err := h.db.WithContext(ctx).Raw(`
		SELECT COUNT(*)
		FROM deid_token_vault
	`).Row().Scan(&stats.TokenVaultSize); err != nil {
		return stats, err
	}

	type row struct {
		Reason string
		Count  int
	}
	var rows []row
	if err := h.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(NULLIF(TRIM(error), ''), 'Unknown reason') AS reason,
			COUNT(*) AS count
		FROM ingestion_requests
		WHERE status = 'failed'
		GROUP BY reason
		ORDER BY count DESC
		LIMIT 5
	`).Scan(&rows).Error; err != nil {
		return stats, err
	}

	stats.TopReasons = make([]ReasonCount, 0, len(rows))
	for _, r := range rows {
		stats.TopReasons = append(stats.TopReasons, ReasonCount{Reason: r.Reason, Count: r.Count})
	}

	return stats, nil
}

func (h *MetricsHandler) fetchRecentDLPIncidents(ctx context.Context, limit int) ([]DLPIncident, error) {
	if limit <= 0 {
		limit = 20
	}

	rows := []struct {
		ID         string    `gorm:"column:id"`
		Source     string    `gorm:"column:source"`
		Format     string    `gorm:"column:format"`
		Status     string    `gorm:"column:status"`
		Error      string    `gorm:"column:error"`
		UpdatedAt  time.Time `gorm:"column:updated_at"`
		CreatedAt  time.Time `gorm:"column:created_at"`
		RetryCount int       `gorm:"column:retry_count"`
	}{}

	if err := h.db.WithContext(ctx).Raw(`
		SELECT id, source, format, status, COALESCE(error, '') AS error, updated_at, created_at, retry_count
		FROM ingestion_requests
		WHERE status = 'failed'
		ORDER BY updated_at DESC
		LIMIT ?
	`, limit).Scan(&rows).Error; err != nil {
		return nil, err
	}

	incidents := make([]DLPIncident, 0, len(rows))
	for _, row := range rows {
		incidents = append(incidents, DLPIncident{
			ID:         row.ID,
			Source:     row.Source,
			Format:     row.Format,
			Status:     row.Status,
			Error:      strings.TrimSpace(row.Error),
			UpdatedAt:  row.UpdatedAt,
			CreatedAt:  row.CreatedAt,
			RetryCount: row.RetryCount,
		})
	}

	return incidents, nil
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
			AVG(latency_ms) AS latency_ms
		FROM prediction_logs
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
