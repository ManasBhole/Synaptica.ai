package storage

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/synaptica-ai/platform/pkg/analytics/dsl"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type LakehouseFact struct {
	ID           string            `gorm:"primaryKey;column:id"`
	MasterID     string            `gorm:"column:master_id"`
	PatientID    string            `gorm:"column:patient_id"`
	ResourceType string            `gorm:"column:resource_type"`
	Canonical    datatypes.JSONMap `gorm:"column:canonical"`
	Codes        datatypes.JSONMap `gorm:"column:codes"`
	Timestamp    time.Time         `gorm:"column:timestamp"`
	CreatedAt    time.Time         `gorm:"column:created_at"`
}

func (LakehouseFact) TableName() string {
	return "lakehouse_facts"
}

type LakehouseWriter struct {
	db *gorm.DB
}

func NewLakehouseWriter(db *gorm.DB) *LakehouseWriter {
	return &LakehouseWriter{db: db}
}

func (w *LakehouseWriter) AutoMigrate() error {
	return w.db.AutoMigrate(&LakehouseFact{})
}

func (w *LakehouseWriter) Write(ctx context.Context, fact *LakehouseFact) error {
	fact.CreatedAt = time.Now().UTC()
	return w.db.WithContext(ctx).Create(fact).Error
}

func (w *LakehouseWriter) QueryCohort(ctx context.Context, parsed dsl.Query, request models.CohortQuery) (models.CohortResult, error) {
	start := time.Now()
	limit := parsed.Limit
	if limit <= 0 {
		limit = 500
	}
	if limit > 5000 {
		limit = 5000
	}

	countQuery := w.buildCohortQuery(ctx, parsed, request)
	var total int64
	if err := countQuery.Distinct("patient_id").Count(&total).Error; err != nil {
		return models.CohortResult{}, err
	}

	idQuery := w.buildCohortQuery(ctx, parsed, request)
	var patientIDs []string
	if err := idQuery.Distinct("patient_id").Order("patient_id").Limit(minInt(limit, 500)).Pluck("patient_id", &patientIDs).Error; err != nil {
		return models.CohortResult{}, err
	}

	recordsQuery := w.buildCohortQuery(ctx, parsed, request)
	var facts []LakehouseFact
	sampleLimit := minInt(limit, 200)
	if err := recordsQuery.Order("timestamp desc").Limit(sampleLimit).Find(&facts).Error; err != nil {
		return models.CohortResult{}, err
	}

	metadata := map[string]interface{}{
		"fields":  parsed.SelectFields,
		"records": projectFacts(facts, parsed.SelectFields),
	}

	return models.CohortResult{
		CohortID:   request.ID,
		PatientIDs: patientIDs,
		Count:      int(total),
		QueryTime:  time.Since(start),
		Metadata:   metadata,
	}, nil
}

func (w *LakehouseWriter) buildCohortQuery(ctx context.Context, parsed dsl.Query, request models.CohortQuery) *gorm.DB {
	tx := w.db.WithContext(ctx).Model(&LakehouseFact{})
	if request.ID != "" {
		tx = tx.Where("id = ?", request.ID)
	}
	for _, clause := range parsed.Filters {
		tx = applyClause(tx, clause)
	}
	return tx
}

type fieldSpec struct {
	Column string
	Kind   string
}

var cohortFieldMap = map[string]fieldSpec{
	"patient_id":    {Column: "patient_id", Kind: "string"},
	"master_id":     {Column: "master_id", Kind: "string"},
	"resource_type": {Column: "resource_type", Kind: "string"},
	"concept":       {Column: "canonical ->> 'concept'", Kind: "string"},
	"unit":          {Column: "canonical ->> 'unit'", Kind: "string"},
	"value":         {Column: "canonical ->> 'value'", Kind: "numeric"},
	"timestamp":     {Column: "timestamp", Kind: "time"},
	"code_loinc":    {Column: "codes ->> 'loinc'", Kind: "string"},
	"code_snomed":   {Column: "codes ->> 'snomed'", Kind: "string"},
}

func applyClause(tx *gorm.DB, clause dsl.Clause) *gorm.DB {
	spec, ok := cohortFieldMap[clause.Field]
	if !ok {
		return tx
	}
	op := clause.Operator
	if op == "!=" {
		op = "<>"
	}
	value := strings.TrimSpace(clause.Value)
	value = strings.Trim(value, "'\"")

	switch spec.Kind {
	case "string":
		if value == "" {
			return tx
		}
		return tx.Where(fmt.Sprintf("%s %s ?", spec.Column, op), value)
	case "numeric":
		if value == "" {
			return tx
		}
		f, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return tx
		}
		return tx.Where(fmt.Sprintf("CAST(%s AS DOUBLE PRECISION) %s ?", spec.Column, op), f)
	case "time":
		t, err := parseTime(value)
		if err != nil {
			return tx
		}
		return tx.Where(fmt.Sprintf("%s %s ?", spec.Column, op), t)
	default:
		return tx
	}
}

func projectFacts(facts []LakehouseFact, fields []string) []map[string]interface{} {
	if len(fields) == 0 {
		fields = []string{"patient_id", "resource_type", "concept", "value", "timestamp"}
	}
	result := make([]map[string]interface{}, 0, len(facts))
	for _, fact := range facts {
		record := make(map[string]interface{}, len(fields))
		for _, field := range fields {
			switch field {
			case "patient_id":
				record["patient_id"] = fact.PatientID
			case "master_id":
				record["master_id"] = fact.MasterID
			case "resource_type":
				record["resource_type"] = fact.ResourceType
			case "concept":
				record["concept"] = fact.Canonical["concept"]
			case "unit":
				record["unit"] = fact.Canonical["unit"]
			case "value":
				record["value"] = fact.Canonical["value"]
			case "timestamp":
				record["timestamp"] = fact.Timestamp
			case "code_loinc":
				record["code_loinc"] = fact.Codes["loinc"]
			case "code_snomed":
				record["code_snomed"] = fact.Codes["snomed"]
			default:
				record[field] = nil
			}
		}
		result = append(result, record)
	}
	return result
}

func parseTime(value string) (time.Time, error) {
	formats := []string{time.RFC3339, "2006-01-02 15:04", "2006-01-02"}
	for _, layout := range formats {
		if t, err := time.Parse(layout, value); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid time format: %s", value)
}

func minInt(a, b int) int {
	return int(math.Min(float64(a), float64(b)))
}

func (w *LakehouseWriter) GetTrainingData(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	var facts []LakehouseFact
	tx := w.db.WithContext(ctx)
	if filters != nil {
		if patientIDs, ok := filters["patient_ids"].([]string); ok && len(patientIDs) > 0 {
			tx = tx.Where("patient_id IN ?", patientIDs)
		}
	}
	if err := tx.Order("timestamp desc").Limit(1000).Find(&facts).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(facts))
	for _, fact := range facts {
		row := map[string]interface{}{
			"master_id":     fact.MasterID,
			"patient_id":    fact.PatientID,
			"resource_type": fact.ResourceType,
			"timestamp":     fact.Timestamp,
			"canonical":     map[string]interface{}(fact.Canonical),
			"codes":         map[string]interface{}(fact.Codes),
		}
		rows = append(rows, row)
	}
	return rows, nil
}
