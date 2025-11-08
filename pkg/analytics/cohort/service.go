package cohort

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/synaptica-ai/platform/pkg/analytics/dsl"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/storage"
)

type Service struct {
	lakehouse *storage.LakehouseWriter
	olap      *storage.OLAPWriter
}

func NewService(lakehouse *storage.LakehouseWriter, olap *storage.OLAPWriter) *Service {
	return &Service{lakehouse: lakehouse, olap: olap}
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
