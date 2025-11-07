package linkage

import (
	"time"

	"gorm.io/datatypes"
)

type MasterPatient struct {
	ID        string    `gorm:"primaryKey;column:id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

type PatientLink struct {
	ID               string            `gorm:"primaryKey;column:id"`
	MasterID         string            `gorm:"column:master_id"`
	PatientID        string            `gorm:"column:patient_id"`
	DeterministicKey string            `gorm:"column:deterministic_key"`
	Score            float64           `gorm:"column:score"`
	Method           string            `gorm:"column:method"`
	Attributes       datatypes.JSONMap `gorm:"column:attributes"`
	CreatedAt        time.Time         `gorm:"column:created_at"`
}

func (MasterPatient) TableName() string {
	return "master_patients"
}

func (PatientLink) TableName() string {
	return "patient_linkages"
}
