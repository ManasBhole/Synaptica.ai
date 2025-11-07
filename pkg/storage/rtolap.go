package storage

import (
	"context"
	"time"

	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type Rollup struct {
	ID        string            `gorm:"primaryKey;column:id"`
	MasterID  string            `gorm:"column:master_id"`
	PatientID string            `gorm:"column:patient_id"`
	Metric    string            `gorm:"column:metric"`
	Value     datatypes.JSONMap `gorm:"column:value"`
	EventTime time.Time         `gorm:"column:event_time"`
	CreatedAt time.Time         `gorm:"column:created_at"`
}

func (Rollup) TableName() string {
	return "olap_rollups"
}

type OLAPWriter struct {
	db *gorm.DB
}

func NewOLAPWriter(db *gorm.DB) *OLAPWriter {
	return &OLAPWriter{db: db}
}

func (w *OLAPWriter) AutoMigrate() error {
	return w.db.AutoMigrate(&Rollup{})
}

func (w *OLAPWriter) Write(ctx context.Context, rollup *Rollup) error {
	rollup.CreatedAt = time.Now().UTC()
	return w.db.WithContext(ctx).Create(rollup).Error
}

func (w *OLAPWriter) QuerySubSecondSlicing(ctx context.Context, filters map[string]interface{}) ([]map[string]interface{}, error) {
	var rollups []Rollup
	tx := w.db.WithContext(ctx)
	if filters != nil {
		if metric, ok := filters["metric"].(string); ok && metric != "" {
			tx = tx.Where("metric = ?", metric)
		}
		if patient, ok := filters["patient_id"].(string); ok && patient != "" {
			tx = tx.Where("patient_id = ?", patient)
		}
	}
	if err := tx.Order("event_time desc").Limit(200).Find(&rollups).Error; err != nil {
		return nil, err
	}
	rows := make([]map[string]interface{}, 0, len(rollups))
	for _, r := range rollups {
		rows = append(rows, map[string]interface{}{
			"master_id":  r.MasterID,
			"patient_id": r.PatientID,
			"metric":     r.Metric,
			"value":      map[string]interface{}(r.Value),
			"event_time": r.EventTime,
		})
	}
	return rows, nil
}
