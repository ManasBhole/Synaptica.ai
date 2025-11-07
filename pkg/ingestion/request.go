package ingestion

import "github.com/synaptica-ai/platform/pkg/common/models"

type RequestWrapper struct {
	Source    string                 `json:"source"`
	Format    string                 `json:"format"`
	Data      map[string]interface{} `json:"data"`
	PatientID string                 `json:"patient_id,omitempty"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

func (r RequestWrapper) ToModel() models.IngestRequest {
	return models.IngestRequest{
		Source:    r.Source,
		Format:    r.Format,
		Data:      r.Data,
		PatientID: r.PatientID,
		Metadata:  r.Metadata,
	}
}
