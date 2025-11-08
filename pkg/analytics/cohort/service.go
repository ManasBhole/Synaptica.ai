package cohort

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/synaptica-ai/platform/pkg/analytics/dsl"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/linkage"
	"github.com/synaptica-ai/platform/pkg/storage"
)

type Service struct {
	lakehouse *storage.LakehouseWriter
	olap      *storage.OLAPWriter
	features  *storage.FeatureStore
	linkage   *linkage.Repository
}

func NewService(lakehouse *storage.LakehouseWriter, olap *storage.OLAPWriter, opts ...Option) *Service {
	svc := &Service{lakehouse: lakehouse, olap: olap}
	for _, opt := range opts {
		if opt != nil {
			opt(svc)
		}
	}
	return svc
}

func (s *Service) Execute(ctx context.Context, query models.CohortQuery) (models.CohortResult, error) {
	parsed, err := dsl.Parse(query.DSL)
	if err != nil {
		return models.CohortResult{}, err
	}
	query.Filters = normalizeFilters(parsed.Filters)
	query.Fields = parsed.SelectFields
	query.Limit = parsed.Limit
	result, err := s.lakehouse.QueryCohort(ctx, parsed, query)
	if err != nil {
		return models.CohortResult{}, err
	}
	if result.Metadata == nil {
		result.Metadata = make(map[string]interface{})
	}
	if len(parsed.Filters) > 0 {
		if rows, err := s.olap.QuerySubSecondSlicing(ctx, query.Filters); err == nil {
			result.Metadata["slices"] = rows
		}
	}
	if query.TenantID != "" {
		result.Metadata["tenant"] = query.TenantID
	}
	return result, nil
}

func (s *Service) Export(ctx context.Context, query models.CohortQuery, w io.Writer) error {
	parsed, err := dsl.Parse(query.DSL)
	if err != nil {
		return err
	}
	query.Filters = normalizeFilters(parsed.Filters)
	query.Fields = parsed.SelectFields
	query.Limit = parsed.Limit

	fields := parsed.SelectFields
	if len(fields) == 0 {
		fields = []string{"patient_id", "resource_type", "concept", "value", "timestamp"}
	}

	writer := csv.NewWriter(w)
	if err := writer.Write(fields); err != nil {
		return err
	}

	err = s.lakehouse.ExportCohort(ctx, parsed, query, func(fact *storage.LakehouseFact) error {
		projection := storage.FactToProjection(fact, fields)
		row := make([]string, len(fields))
		for i, field := range fields {
			row[i] = stringifyValue(projection[field])
		}
		if err := writer.Write(row); err != nil {
			return err
		}
		return nil
	})
	writer.Flush()
	if err != nil {
		return err
	}
	if err := writer.Error(); err != nil {
		return err
	}
	return nil
}

func (s *Service) Drilldown(ctx context.Context, req models.CohortDrilldownRequest) (models.CohortDrilldown, error) {
	if strings.TrimSpace(req.DSL) == "" {
		return models.CohortDrilldown{}, fmt.Errorf("dsl is required")
	}
	if strings.TrimSpace(req.PatientID) == "" {
		return models.CohortDrilldown{}, fmt.Errorf("patient_id is required")
	}

	parsed, err := dsl.Parse(req.DSL)
	if err != nil {
		return models.CohortDrilldown{}, err
	}
	query := models.CohortQuery{
		ID:       req.CohortID,
		TenantID: req.TenantID,
		DSL:      req.DSL,
		Filters:  normalizeFilters(parsed.Filters),
		Fields:   parsed.SelectFields,
		Limit:    parsed.Limit,
	}

	facts, err := s.lakehouse.FetchTimeline(ctx, parsed, query, req.PatientID, req.Limit)
	if err != nil {
		return models.CohortDrilldown{}, err
	}

	timeline := make([]models.TimelineEvent, 0, len(facts))
	for _, fact := range facts {
		metadata := map[string]interface{}(fact.Canonical)
		codes := map[string]interface{}(fact.Codes)
		timeline = append(timeline, models.TimelineEvent{
			PatientID:    fact.PatientID,
			ResourceType: fact.ResourceType,
			Concept:      metadata["concept"],
			Unit:         metadata["unit"],
			Value:        metadata["value"],
			Timestamp:    fact.Timestamp,
			Codes:        codes,
			Metadata:     metadata,
		})
	}

	var features map[string]interface{}
	if s.features != nil {
		if features, err = s.features.GetFeatures(ctx, req.PatientID); err != nil {
			return models.CohortDrilldown{}, err
		}
		if len(features) == 0 {
			if offline, err := s.features.GetLatestOfflineFeatures(ctx, req.PatientID); err == nil && len(offline) > 0 {
				features = offline
			}
		}
	}

	var linkageSummary *models.LinkageSummary
	masterID := ""
	if s.linkage != nil {
		if links, err := s.linkage.FindLinksByPatient(ctx, req.PatientID, 25); err == nil && len(links) > 0 {
			summary := &models.LinkageSummary{
				MasterPatientID: links[0].MasterID,
				PrimaryScore:    links[0].Score,
				Method:          links[0].Method,
				LinkedPatients:  make([]models.LinkedPatient, 0, len(links)),
			}
			masterID = links[0].MasterID
			for _, link := range links {
				summary.LinkedPatients = append(summary.LinkedPatients, models.LinkedPatient{
					PatientID:  link.PatientID,
					Score:      link.Score,
					Method:     link.Method,
					Attributes: map[string]interface{}(link.Attributes),
				})
			}
			linkageSummary = summary
		}
	}

	metadata := map[string]interface{}{
		"fields":  query.Fields,
		"filters": query.Filters,
		"tenant":  req.TenantID,
	}
	if req.Limit > 0 {
		metadata["limit"] = req.Limit
	}

	result := models.CohortDrilldown{
		CohortID:        req.CohortID,
		PatientID:       req.PatientID,
		MasterPatientID: masterID,
		Timeline:        timeline,
		Metadata:        metadata,
	}
	if len(features) > 0 {
		result.Features = features
	}
	if linkageSummary != nil {
		result.Linkage = linkageSummary
	}
	return result, nil
}

func (s *Service) VerifyDSL(input string) error {
	_, err := dsl.Parse(input)
	return err
}

func normalizeFilters(filters []dsl.Clause) map[string]interface{} {
	result := make(map[string]interface{})
	for _, clause := range filters {
		result[clause.Field] = clause.Value
	}
	return result
}

func stringifyValue(value interface{}) string {
	switch v := value.(type) {
	case nil:
		return ""
	case string:
		return v
	case fmt.Stringer:
		return v.String()
	case fmt.GoStringer:
		return v.GoString()
	case time.Time:
		return v.UTC().Format(time.RFC3339)
	case *time.Time:
		if v == nil {
			return ""
		}
		return v.UTC().Format(time.RFC3339)
	case float32:
		return fmt.Sprintf("%.6f", float64(v))
	case float64:
		return fmt.Sprintf("%.6f", v)
	case int, int8, int16, int32, int64:
		return fmt.Sprintf("%d", v)
	case uint, uint8, uint16, uint32, uint64:
		return fmt.Sprintf("%d", v)
	case bool:
		if v {
			return "true"
		}
		return "false"
	default:
		if bytes, err := json.Marshal(v); err == nil {
			return string(bytes)
		}
		return fmt.Sprintf("%v", v)
	}
}

type Option func(*Service)

func WithFeatureStore(store *storage.FeatureStore) Option {
	return func(s *Service) {
		s.features = store
	}
}

func WithLinkageRepository(repo *linkage.Repository) Option {
	return func(s *Service) {
		s.linkage = repo
	}
}
