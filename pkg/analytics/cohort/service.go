package cohort

import (
	"context"

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
	result, err := s.lakehouse.QueryCohort(ctx, query)
	if err != nil {
		return models.CohortResult{}, err
	}
	if len(parsed.Filters) > 0 {
		if rows, err := s.olap.QuerySubSecondSlicing(ctx, query.Filters); err == nil {
			result.Metadata = map[string]interface{}{"slices": rows}
		}
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
