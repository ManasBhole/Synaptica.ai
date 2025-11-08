package models

import (
	"time"

	"github.com/google/uuid"
)

// Upstream data models
type IngestRequest struct {
	Source    string                 `json:"source"` // hospital, lab, imaging, wearable, telehealth
	Format    string                 `json:"format"` // FHIR, HL7, ABDM, CSV, DICOM, JSON
	Data      map[string]interface{} `json:"data"`
	PatientID string                 `json:"patient_id,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

type IngestResponse struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
}

// Event Bus models
type Event struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"` // upstream, sanitize, deidentify, normalize, link
	Source    string                 `json:"source"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]string      `json:"metadata,omitempty"`
}

// DLP & PHI Detection
type PHIDetectionResult struct {
	Detected    bool          `json:"detected"`
	Confidence  float64       `json:"confidence"`
	PHITypes    []string      `json:"phi_types"` // SSN, DOB, Name, Address, etc.
	Positions   []PHIPosition `json:"positions"`
	Suggestions []string      `json:"suggestions,omitempty"`
}

type PHIPosition struct {
	Start int    `json:"start"`
	End   int    `json:"end"`
	Type  string `json:"type"`
	Value string `json:"value"`
}

// De-Identification
type DeIDRequest struct {
	Data       map[string]interface{} `json:"data"`
	PHIResults PHIDetectionResult     `json:"phi_results"`
	Retention  string                 `json:"retention,omitempty"` // k-anonymity, l-diversity
}

type DeIDResponse struct {
	TokenizedData  map[string]interface{} `json:"tokenized_data"`
	TokenVault     map[string]string      `json:"token_vault"` // token -> original value
	AnonymityLevel string                 `json:"anonymity_level"`
}

// Schema Normalization
type NormalizedRecord struct {
	ID           string                 `json:"id"`
	PatientID    string                 `json:"patient_id"`
	ResourceType string                 `json:"resource_type"` // Observation, Condition, Procedure, etc.
	Canonical    map[string]interface{} `json:"canonical"`
	Codes        map[string]string      `json:"codes"` // SNOMED, LOINC, ICD
	Timestamp    time.Time              `json:"timestamp"`
}

// Record Linkage
type LinkageRequest struct {
	Records []NormalizedRecord `json:"records"`
}

type LinkageResult struct {
	MasterPatientID string   `json:"master_patient_id"`
	LinkedIDs       []string `json:"linked_ids"`
	Confidence      float64  `json:"confidence"`
	Method          string   `json:"method"` // deterministic, probabilistic
}

// Cohort Query
type CohortQuery struct {
	ID          string                 `json:"id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	DSL         string                 `json:"dsl"`
	Description string                 `json:"description"`
	Filters     map[string]interface{} `json:"filters"`
	Limit       int                    `json:"limit,omitempty"`
	Fields      []string               `json:"fields,omitempty"`
}

type CohortResult struct {
	CohortID   string                 `json:"cohort_id"`
	PatientIDs []string               `json:"patient_ids"`
	Count      int                    `json:"count"`
	QueryTime  time.Duration          `json:"query_time"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

type CohortDrilldownRequest struct {
	CohortID  string   `json:"cohort_id"`
	TenantID  string   `json:"tenant_id,omitempty"`
	DSL       string   `json:"dsl"`
	PatientID string   `json:"patient_id"`
	Fields    []string `json:"fields,omitempty"`
	Limit     int      `json:"limit,omitempty"`
}

type TimelineEvent struct {
	PatientID    string                 `json:"patient_id"`
	ResourceType string                 `json:"resource_type"`
	Concept      interface{}            `json:"concept,omitempty"`
	Unit         interface{}            `json:"unit,omitempty"`
	Value        interface{}            `json:"value,omitempty"`
	Timestamp    time.Time              `json:"timestamp"`
	Codes        map[string]interface{} `json:"codes,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type LinkedPatient struct {
	PatientID  string                 `json:"patient_id"`
	Score      float64                `json:"score"`
	Method     string                 `json:"method"`
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

type LinkageSummary struct {
	MasterPatientID string          `json:"master_patient_id"`
	PrimaryScore    float64         `json:"primary_score"`
	Method          string          `json:"method"`
	LinkedPatients  []LinkedPatient `json:"linked_patients"`
}

type CohortDrilldown struct {
	CohortID        string                 `json:"cohort_id"`
	PatientID       string                 `json:"patient_id"`
	MasterPatientID string                 `json:"master_patient_id,omitempty"`
	Timeline        []TimelineEvent        `json:"timeline"`
	Features        map[string]interface{} `json:"features,omitempty"`
	Linkage         *LinkageSummary        `json:"linkage,omitempty"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type CohortTemplate struct {
	ID          string    `json:"id"`
	TenantID    string    `json:"tenant_id,omitempty"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	DSL         string    `json:"dsl"`
	Tags        []string  `json:"tags,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

// Feature Store
type Feature struct {
	Name      string                 `json:"name"`
	Value     interface{}            `json:"value"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type FeatureSet struct {
	PatientID string             `json:"patient_id"`
	Features  map[string]Feature `json:"features"`
	Version   int                `json:"version"`
}

// Model Training
type TrainingJob struct {
	ID           uuid.UUID              `json:"id"`
	ModelType    string                 `json:"model_type"`
	Config       map[string]interface{} `json:"config"`
	Status       string                 `json:"status"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
	Metrics      map[string]interface{} `json:"metrics,omitempty"`
	ArtifactPath string                 `json:"artifact_path,omitempty"`
	ErrorMessage string                 `json:"error_message,omitempty"`
}

// Model Serving
type PredictionRequest struct {
	PatientID string                 `json:"patient_id"`
	Features  map[string]interface{} `json:"features"`
	ModelName string                 `json:"model_name"`
}

type PredictionResponse struct {
	PatientID    string                 `json:"patient_id"`
	Predictions  map[string]interface{} `json:"predictions"`
	Confidence   float64                `json:"confidence"`
	ModelVersion string                 `json:"model_version"`
	Latency      time.Duration          `json:"latency"`
}

// Clean Room
type CleanRoomQuery struct {
	ID       string                 `json:"id"`
	Query    map[string]interface{} `json:"query"`
	DPBudget float64                `json:"dp_budget"` // epsilon
	Lineage  []string               `json:"lineage"`
}

type CleanRoomResult struct {
	QueryID      string                 `json:"query_id"`
	Aggregates   map[string]interface{} `json:"aggregates"`
	NoiseAdded   float64                `json:"noise_added"`
	DPBudgetUsed float64                `json:"dp_budget_used"`
}
