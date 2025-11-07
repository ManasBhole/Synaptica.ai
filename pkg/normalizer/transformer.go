package normalizer

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/synaptica-ai/platform/pkg/common/models"
	"github.com/synaptica-ai/platform/pkg/terminology"
)

type Transformer struct {
	catalog terminology.Catalog
	allowed map[string]struct{}
}

func NewTransformer(cat terminology.Catalog, allowed []string) *Transformer {
	allowedSet := make(map[string]struct{})
	for _, r := range allowed {
		allowedSet[strings.ToLower(strings.TrimSpace(r))] = struct{}{}
	}
	return &Transformer{catalog: cat, allowed: allowedSet}
}

func (t *Transformer) Transform(data map[string]interface{}) (*models.NormalizedRecord, error) {
	if data == nil {
		return nil, errors.New("nil payload")
	}

	resourceType := getString(data["resourceType"])
	if resourceType == "" {
		return nil, errors.New("resourceType missing")
	}
	resourceKey := strings.ToLower(resourceType)
	if len(t.allowed) > 0 {
		if _, ok := t.allowed[resourceKey]; !ok {
			return nil, fmt.Errorf("resourceType %s not allowed", resourceType)
		}
	}

	patientID := getString(data["patient_id"])
	if patientID == "" {
		patientID = extractPatientReference(data)
	}

	timestamp := time.Now().UTC()
	if tsStr := getString(data["effectiveDateTime"]); tsStr != "" {
		if parsed, err := time.Parse(time.RFC3339, tsStr); err == nil {
			timestamp = parsed
		}
	}

	canonical := map[string]interface{}{}
	codes := make(map[string]string)

	switch resourceKey {
	case "observation":
		canonical = buildObservationCanonical(data)
		conceptName := strings.ToLower(getString(canonical["concept"]))
		if conceptName != "" {
			if concept, ok := t.catalog.Lookup(conceptName); ok {
				if concept.SNOMED != "" {
					codes["SNOMED"] = concept.SNOMED
				}
				if concept.LOINC != "" {
					codes["LOINC"] = concept.LOINC
				}
				if concept.ICD10 != "" {
					codes["ICD10"] = concept.ICD10
				}
			}
		}
	case "condition":
		canonical = buildConditionCanonical(data)
		conceptName := strings.ToLower(getString(canonical["concept"]))
		if conceptName != "" {
			if concept, ok := t.catalog.Lookup(conceptName); ok {
				if concept.SNOMED != "" {
					codes["SNOMED"] = concept.SNOMED
				}
				if concept.ICD10 != "" {
					codes["ICD10"] = concept.ICD10
				}
			}
		}
	case "procedure":
		canonical = buildProcedureCanonical(data)
	default:
		return nil, fmt.Errorf("resourceType %s unsupported", resourceType)
	}

	if len(canonical) == 0 {
		return nil, fmt.Errorf("unable to build canonical record for %s", resourceType)
	}

	record := &models.NormalizedRecord{
		ID:           uuid.New().String(),
		PatientID:    patientID,
		ResourceType: strings.Title(resourceKey),
		Canonical:    canonical,
		Codes:        codes,
		Timestamp:    timestamp,
	}

	return record, nil
}

func buildObservationCanonical(data map[string]interface{}) map[string]interface{} {
	canonical := make(map[string]interface{})
	codeMap := extractMap(data["code"])
	concept := strings.ToLower(getString(codeMap["text"]))
	if concept == "" && len(codeMap) > 0 {
		concept = strings.ToLower(getString(codeMap["display"]))
	}
	if concept == "" {
		concept = strings.ToLower(getString(data["concept"]))
	}
	canonical["concept"] = concept

	valueQuantity := extractMap(data["valueQuantity"])
	value := data["value"]
	unit := data["unit"]
	if len(valueQuantity) > 0 {
		if v, ok := valueQuantity["value"]; ok {
			value = v
		}
		if u, ok := valueQuantity["unit"]; ok {
			unit = u
		}
	}
	canonical["value"] = value
	canonical["unit"] = unit
	canonical["effectiveDateTime"] = getString(data["effectiveDateTime"])
	return canonical
}

func buildConditionCanonical(data map[string]interface{}) map[string]interface{} {
	canonical := make(map[string]interface{})
	desc := getString(data["description"])
	if desc == "" {
		desc = getString(data["concept"])
	}
	if desc == "" {
		code := extractMap(data["code"])
		desc = getString(code["text"])
	}
	canonical["concept"] = strings.ToLower(desc)
	canonical["recordedDate"] = getString(data["recordedDate"])
	canonical["clinicalStatus"] = getString(extractMap(data["clinicalStatus"])["text"])
	return canonical
}

func buildProcedureCanonical(data map[string]interface{}) map[string]interface{} {
	canonical := make(map[string]interface{})
	desc := getString(data["description"])
	if desc == "" {
		code := extractMap(data["code"])
		desc = getString(code["text"])
	}
	canonical["concept"] = strings.ToLower(desc)
	canonical["performed"] = getString(data["performedDateTime"])
	return canonical
}

func extractPatientReference(data map[string]interface{}) string {
	subject := extractMap(data["subject"])
	if ref := getString(subject["reference"]); ref != "" {
		parts := strings.Split(ref, "/")
		return parts[len(parts)-1]
	}
	return ""
}

func extractMap(value interface{}) map[string]interface{} {
	if m, ok := value.(map[string]interface{}); ok {
		return m
	}
	return map[string]interface{}{}
}

func getString(v interface{}) string {
	switch val := v.(type) {
	case string:
		return strings.TrimSpace(val)
	case fmt.Stringer:
		return strings.TrimSpace(val.String())
	default:
		return ""
	}
}
