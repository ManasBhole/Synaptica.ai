package pipeline

import (
	"strings"
	"time"

	"github.com/synaptica-ai/platform/pkg/common/models"
)

func ExtractFeatures(record *models.NormalizedRecord, linkage *models.LinkageResult) map[string]interface{} {
	features := map[string]interface{}{
		"master_patient_id": linkage.MasterPatientID,
		"resource_type":     strings.ToLower(record.ResourceType),
		"event_time":        record.Timestamp.Format(time.RFC3339),
	}

	if record.Canonical != nil {
		if value, ok := record.Canonical["value"]; ok {
			features["value"] = value
		}
		if unit, ok := record.Canonical["unit"]; ok {
			features["unit"] = unit
		}
		if concept, ok := record.Canonical["concept"].(string); ok {
			features["concept"] = concept
		}
	}

	return features
}
