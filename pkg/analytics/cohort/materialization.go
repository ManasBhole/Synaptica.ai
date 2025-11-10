package cohort

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/logger"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/storage"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	materialStatusQueued    = "queued"
	materialStatusRunning   = "running"
	materialStatusCompleted = "completed"
	materialStatusFailed    = "failed"
)

type materializationModel struct {
	ID           uuid.UUID      `gorm:"primaryKey;column:id"`
	CohortID     string         `gorm:"column:cohort_id"`
	TenantID     *string        `gorm:"column:tenant_id"`
	DSL          string         `gorm:"column:dsl"`
	Fields       datatypes.JSON `gorm:"column:fields"`
	Filters      datatypes.JSON `gorm:"column:filters"`
	Status       string         `gorm:"column:status"`
	ResultCount  int            `gorm:"column:result_count"`
	ErrorMessage string         `gorm:"column:error_message"`
	RequestedBy  string         `gorm:"column:requested_by"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	StartedAt    *time.Time     `gorm:"column:started_at"`
	CompletedAt  *time.Time     `gorm:"column:completed_at"`
}

func (materializationModel) TableName() string {
	return "cohort_materializations"
}

type MaterializationRepository struct {
	db *gorm.DB
}

func NewMaterializationRepository(db *gorm.DB) *MaterializationRepository {
	return &MaterializationRepository{db: db}
}

func (r *MaterializationRepository) AutoMigrate() error {
	return r.db.AutoMigrate(&materializationModel{})
}

func (r *MaterializationRepository) Create(ctx context.Context, model *materializationModel) error {
	return r.db.WithContext(ctx).Create(model).Error
}

func (r *MaterializationRepository) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return r.db.WithContext(ctx).Model(&materializationModel{}).Where("id = ?", id).Updates(updates).Error
}

func (r *MaterializationRepository) Get(ctx context.Context, id uuid.UUID) (*materializationModel, error) {
	var model materializationModel
	result := r.db.WithContext(ctx).First(&model, "id = ?", id)
	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, result.Error
	}
	return &model, result.Error
}

func (r *MaterializationRepository) List(ctx context.Context, tenantID string, limit int) ([]materializationModel, error) {
	if limit <= 0 {
		limit = 50
	}
	query := r.db.WithContext(ctx).Order("created_at DESC").Limit(limit)
	if tenantID != "" {
		query = query.Where("tenant_id = ? OR tenant_id IS NULL", tenantID)
	}
	var records []materializationModel
	if err := query.Find(&records).Error; err != nil {
		return nil, err
	}
	return records, nil
}

func modelToDomain(model *materializationModel) models.CohortMaterialization {
	var fields []string
	if len(model.Fields) > 0 {
		_ = json.Unmarshal(model.Fields, &fields)
	}
	filters := map[string]interface{}{}
	if len(model.Filters) > 0 {
		_ = json.Unmarshal(model.Filters, &filters)
	}
	result := models.CohortMaterialization{
		ID:           model.ID,
		CohortID:     model.CohortID,
		DSL:          model.DSL,
		Fields:       fields,
		Filters:      filters,
		Status:       model.Status,
		ResultCount:  model.ResultCount,
		ErrorMessage: model.ErrorMessage,
		RequestedBy:  model.RequestedBy,
		CreatedAt:    model.CreatedAt,
		StartedAt:    model.StartedAt,
		CompletedAt:  model.CompletedAt,
	}
	if model.TenantID != nil {
		result.TenantID = *model.TenantID
	}
	return result
}

type Materializer struct {
	repo         *MaterializationRepository
	service      *Service
	featureStore *storage.FeatureStore
	workers      chan struct{}
}

func NewMaterializer(repo *MaterializationRepository, svc *Service, featureStore *storage.FeatureStore, maxWorkers int) *Materializer {
	if maxWorkers <= 0 {
		maxWorkers = 1
	}
	return &Materializer{
		repo:         repo,
		service:      svc,
		featureStore: featureStore,
		workers:      make(chan struct{}, maxWorkers),
	}
}

func (m *Materializer) Enqueue(ctx context.Context, req models.CohortMaterializeRequest) (models.CohortMaterialization, error) {
	jobID := uuid.New()
	var tenantPtr *string
	if strings.TrimSpace(req.TenantID) != "" {
		tenant := req.TenantID
		tenantPtr = &tenant
	}
	fieldsJSON, _ := json.Marshal(req.Fields)
	filtersJSON, _ := json.Marshal(req.Filters)
	model := &materializationModel{
		ID:          jobID,
		CohortID:    req.CohortID,
		TenantID:    tenantPtr,
		DSL:         req.DSL,
		Fields:      datatypes.JSON(fieldsJSON),
		Filters:     datatypes.JSON(filtersJSON),
		Status:      materialStatusQueued,
		RequestedBy: req.RequestedBy,
		CreatedAt:   time.Now().UTC(),
	}
	if err := m.repo.Create(ctx, model); err != nil {
		return models.CohortMaterialization{}, err
	}

	go m.run(jobID, req)

	return modelToDomain(model), nil
}

func (m *Materializer) List(ctx context.Context, tenantID string, limit int) ([]models.CohortMaterialization, error) {
	entries, err := m.repo.List(ctx, tenantID, limit)
	if err != nil {
		return nil, err
	}
	result := make([]models.CohortMaterialization, 0, len(entries))
	for _, entry := range entries {
		copy := entry
		result = append(result, modelToDomain(&copy))
	}
	return result, nil
}

func (m *Materializer) run(jobID uuid.UUID, req models.CohortMaterializeRequest) {
	m.workers <- struct{}{}
	defer func() { <-m.workers }()

	ctx := context.Background()
	started := time.Now().UTC()
	_ = m.repo.Update(ctx, jobID, map[string]interface{}{
		"status":     materialStatusRunning,
		"started_at": started,
	})

	result, err := m.service.Execute(ctx, models.CohortQuery{
		ID:       req.CohortID,
		TenantID: req.TenantID,
		DSL:      req.DSL,
		Filters:  req.Filters,
		Limit:    req.Limit,
		Fields:   req.Fields,
	})
	if err != nil {
		m.fail(ctx, jobID, err)
		return
	}

	if err := m.materializeResult(ctx, jobID, req, result); err != nil {
		m.fail(ctx, jobID, err)
		return
	}

	completed := time.Now().UTC()
	_ = m.repo.Update(ctx, jobID, map[string]interface{}{
		"status":        materialStatusCompleted,
		"result_count":  len(result.PatientIDs),
		"completed_at":  completed,
		"error_message": "",
	})
}

func (m *Materializer) fail(ctx context.Context, jobID uuid.UUID, err error) {
	logger.Log.WithError(err).Error("cohort materialization failed")
	completed := time.Now().UTC()
	_ = m.repo.Update(ctx, jobID, map[string]interface{}{
		"status":        materialStatusFailed,
		"error_message": err.Error(),
		"completed_at":  completed,
	})
}

func (m *Materializer) materializeResult(ctx context.Context, jobID uuid.UUID, req models.CohortMaterializeRequest, result models.CohortResult) error {
	if m.featureStore == nil {
		return fmt.Errorf("feature store not configured")
	}
	records := extractRecords(result.Metadata)
	grouped := groupRecordsByPatient(records)
	version := int(time.Now().Unix())
	for _, patientID := range result.PatientIDs {
		recs := grouped[patientID]
		features := buildMaterializedFeatures(patientID, req.CohortID, recs)
		if err := m.featureStore.BuildFeatures(ctx, patientID, features, version); err != nil {
			return err
		}
		if err := m.featureStore.MaterializeHotFeatures(ctx, patientID, features); err != nil {
			return err
		}
	}
	return nil
}

func extractRecords(metadata map[string]interface{}) []map[string]interface{} {
	raw := metadata["records"]
	if raw == nil {
		return nil
	}
	switch v := raw.(type) {
	case []map[string]interface{}:
		return v
	case []interface{}:
		records := make([]map[string]interface{}, 0, len(v))
		for _, item := range v {
			if rec, ok := item.(map[string]interface{}); ok {
				records = append(records, rec)
			}
		}
		return records
	default:
		return nil
	}
}

func groupRecordsByPatient(records []map[string]interface{}) map[string][]map[string]interface{} {
	grouped := make(map[string][]map[string]interface{})
	for _, rec := range records {
		pid, _ := rec["patient_id"].(string)
		if pid == "" {
			continue
		}
		grouped[pid] = append(grouped[pid], rec)
	}
	return grouped
}

func buildMaterializedFeatures(patientID, cohortID string, records []map[string]interface{}) map[string]interface{} {
	features := map[string]interface{}{
		"cohort_id":       cohortID,
		"patient_id":      patientID,
		"record_count":    len(records),
		"materialized_at": time.Now().UTC(),
	}
	if len(records) == 0 {
		return features
	}
	latestRecord := latestRecord(records)
	if concept, ok := latestRecord["concept"]; ok {
		features["latest_concept"] = concept
	}
	if value, ok := latestRecord["value"]; ok {
		features["latest_value"] = value
	}
	if ts, ok := latestRecord["timestamp"]; ok {
		features["latest_timestamp"] = ts
	}

	sum := 0.0
	count := 0
	for _, rec := range records {
		if val, ok := rec["value"]; ok {
			if f, err := numeric(val); err == nil {
				sum += f
				count++
			}
		}
	}
	if count > 0 {
		features["average_value"] = sum / float64(count)
	}
	return features
}

func latestRecord(records []map[string]interface{}) map[string]interface{} {
	if len(records) == 0 {
		return map[string]interface{}{}
	}
	latest := records[0]
	latestTime := parseTimestamp(latest["timestamp"])
	for _, rec := range records[1:] {
		ts := parseTimestamp(rec["timestamp"])
		if ts.After(latestTime) {
			latest = rec
			latestTime = ts
		}
	}
	return latest
}

func parseTimestamp(value interface{}) time.Time {
	switch v := value.(type) {
	case time.Time:
		return v
	case string:
		if t, err := time.Parse(time.RFC3339, v); err == nil {
			return t
		}
	}
	return time.Time{}
}

func numeric(value interface{}) (float64, error) {
	switch v := value.(type) {
	case float64:
		return v, nil
	case float32:
		return float64(v), nil
	case int:
		return float64(v), nil
	case int64:
		return float64(v), nil
	case json.Number:
		return v.Float64()
	case string:
		return strconv.ParseFloat(strings.TrimSpace(v), 64)
	default:
		return 0, fmt.Errorf("unsupported numeric type %T", value)
	}
}
