package storage

import (
	"context"
	"time"

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

func (w *LakehouseWriter) QueryCohort(ctx context.Context, query models.CohortQuery) (models.CohortResult, error) {
	start := time.Now()
	var facts []LakehouseFact
	tx := w.db.WithContext(ctx)
	if query.ID != "" {
		tx = tx.Where("id = ?", query.ID)
	}
	if query.Filters != nil {
		if patient, ok := query.Filters["patient_id"].(string); ok && patient != "" {
			tx = tx.Where("patient_id = ?", patient)
		}
	}
	if err := tx.Limit(500).Order("timestamp desc").Find(&facts).Error; err != nil {
		return models.CohortResult{}, err
	}
	patientIDs := make(map[string]struct{})
	for _, fact := range facts {
		patientIDs[fact.PatientID] = struct{}{}
	}
	ids := make([]string, 0, len(patientIDs))
	for id := range patientIDs {
		ids = append(ids, id)
	}
	return models.CohortResult{
		CohortID:   query.ID,
		PatientIDs: ids,
		Count:      len(ids),
		QueryTime:  time.Since(start),
	}, nil
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
