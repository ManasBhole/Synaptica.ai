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

type CohortMaterializeRequest struct {
	CohortID    string                 `json:"cohort_id"`
	TenantID    string                 `json:"tenant_id,omitempty"`
	DSL         string                 `json:"dsl"`
	Fields      []string               `json:"fields,omitempty"`
	Filters     map[string]interface{} `json:"filters,omitempty"`
	Limit       int                    `json:"limit,omitempty"`
	RequestedBy string                 `json:"requested_by,omitempty"`
}

type CohortMaterialization struct {
	ID           uuid.UUID              `json:"id"`
	CohortID     string                 `json:"cohort_id"`
	TenantID     string                 `json:"tenant_id,omitempty"`
	DSL          string                 `json:"dsl"`
	Fields       []string               `json:"fields,omitempty"`
	Filters      map[string]interface{} `json:"filters,omitempty"`
	Status       string                 `json:"status"`
	ResultCount  int                    `json:"result_count"`
	ErrorMessage string                 `json:"error_message,omitempty"`
	RequestedBy  string                 `json:"requested_by,omitempty"`
	CreatedAt    time.Time              `json:"created_at"`
	StartedAt    *time.Time             `json:"started_at,omitempty"`
	CompletedAt  *time.Time             `json:"completed_at,omitempty"`
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
	ID               uuid.UUID              `json:"id"`
	ModelType        string                 `json:"model_type"`
	Config           map[string]interface{} `json:"config"`
	Status           string                 `json:"status"`
	CreatedAt        time.Time              `json:"created_at"`
	StartedAt        *time.Time             `json:"started_at,omitempty"`
	CompletedAt      *time.Time             `json:"completed_at,omitempty"`
	Metrics          map[string]interface{} `json:"metrics,omitempty"`
	ArtifactPath     string                 `json:"artifact_path,omitempty"`
	ErrorMessage     string                 `json:"error_message,omitempty"`
	Promoted         bool                   `json:"promoted"`
	PromotedAt       *time.Time             `json:"promoted_at,omitempty"`
	PromotedBy       string                 `json:"promoted_by,omitempty"`
	PromotionNotes   string                 `json:"promotion_notes,omitempty"`
	DeploymentTarget string                 `json:"deployment_target,omitempty"`
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

// EDC / Study Operations
type Study struct {
	ID              uuid.UUID              `json:"id"`
	Code            string                 `json:"code"`
	Name            string                 `json:"name"`
	Phase           string                 `json:"phase,omitempty"`
	TherapeuticArea string                 `json:"therapeutic_area,omitempty"`
	Status          string                 `json:"status"`
	Sponsor         string                 `json:"sponsor,omitempty"`
	ProtocolSummary map[string]interface{} `json:"protocol_summary,omitempty"`
	StartDate       *time.Time             `json:"start_date,omitempty"`
	EndDate         *time.Time             `json:"end_date,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	Sites           []StudySite            `json:"sites,omitempty"`
	Forms           []StudyForm            `json:"forms,omitempty"`
	VisitTemplates  []VisitTemplate        `json:"visit_templates,omitempty"`
	ActiveSubjects  int                    `json:"active_subjects,omitempty"`
	TotalSubjects   int                    `json:"total_subjects,omitempty"`
}

type StudySite struct {
	ID                    uuid.UUID              `json:"id"`
	StudyID               uuid.UUID              `json:"study_id"`
	SiteCode              string                 `json:"site_code"`
	Name                  string                 `json:"name"`
	Country               string                 `json:"country,omitempty"`
	PrincipalInvestigator string                 `json:"principal_investigator,omitempty"`
	Status                string                 `json:"status"`
	Contact               map[string]interface{} `json:"contact,omitempty"`
	CreatedAt             time.Time              `json:"created_at"`
}

type StudyForm struct {
	ID          uuid.UUID              `json:"id"`
	StudyID     uuid.UUID              `json:"study_id"`
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Version     int                    `json:"version"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type VisitTemplate struct {
	ID              uuid.UUID `json:"id"`
	StudyID         uuid.UUID `json:"study_id"`
	Name            string    `json:"name"`
	VisitOrder      int       `json:"visit_order"`
	WindowStartDays *int      `json:"window_start_days,omitempty"`
	WindowEndDays   *int      `json:"window_end_days,omitempty"`
	Required        bool      `json:"required"`
	Forms           []string  `json:"forms,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Subject struct {
	ID               uuid.UUID              `json:"id"`
	StudyID          uuid.UUID              `json:"study_id"`
	SiteID           *uuid.UUID             `json:"site_id,omitempty"`
	SubjectCode      string                 `json:"subject_code"`
	Status           string                 `json:"status"`
	RandomizationArm string                 `json:"randomization_arm,omitempty"`
	ConsentedAt      *time.Time             `json:"consented_at,omitempty"`
	Demographics     map[string]interface{} `json:"demographics,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

type SubjectVisit struct {
	ID              uuid.UUID              `json:"id"`
	SubjectID       uuid.UUID              `json:"subject_id"`
	VisitTemplateID uuid.UUID              `json:"visit_template_id"`
	ScheduledDate   *time.Time             `json:"scheduled_date,omitempty"`
	ActualDate      *time.Time             `json:"actual_date,omitempty"`
	Status          string                 `json:"status"`
	Forms           map[string]interface{} `json:"forms,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

type ConsentVersion struct {
	ID           uuid.UUID  `json:"id"`
	StudyID      uuid.UUID  `json:"study_id"`
	Version      string     `json:"version"`
	Title        string     `json:"title,omitempty"`
	Summary      string     `json:"summary,omitempty"`
	DocumentURL  string     `json:"document_url,omitempty"`
	EffectiveAt  time.Time  `json:"effective_at"`
	SupersededAt *time.Time `json:"superseded_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
}

type ConsentSignature struct {
	ID               uuid.UUID              `json:"id"`
	SubjectID        uuid.UUID              `json:"subject_id"`
	ConsentVersionID uuid.UUID              `json:"consent_version_id"`
	SignedAt         time.Time              `json:"signed_at"`
	SignerName       string                 `json:"signer_name,omitempty"`
	Method           string                 `json:"method,omitempty"`
	IPAddress        string                 `json:"ip_address,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
}

type AuditLog struct {
	ID        int64                  `json:"id"`
	StudyID   uuid.UUID              `json:"study_id"`
	SubjectID *uuid.UUID             `json:"subject_id,omitempty"`
	Actor     string                 `json:"actor"`
	Role      string                 `json:"role,omitempty"`
	Action    string                 `json:"action"`
	Entity    string                 `json:"entity,omitempty"`
	EntityID  string                 `json:"entity_id,omitempty"`
	Payload   map[string]interface{} `json:"payload,omitempty"`
	CreatedAt time.Time              `json:"created_at"`
}

type CreateStudyRequest struct {
	Code            string                 `json:"code"`
	Name            string                 `json:"name"`
	Phase           string                 `json:"phase,omitempty"`
	TherapeuticArea string                 `json:"therapeutic_area,omitempty"`
	Sponsor         string                 `json:"sponsor,omitempty"`
	ProtocolSummary map[string]interface{} `json:"protocol_summary,omitempty"`
	StartDate       *time.Time             `json:"start_date,omitempty"`
	EndDate         *time.Time             `json:"end_date,omitempty"`
}

type UpdateStudyStatusRequest struct {
	Status string `json:"status"`
}

type CreateStudySiteRequest struct {
	SiteCode              string                 `json:"site_code"`
	Name                  string                 `json:"name"`
	Country               string                 `json:"country,omitempty"`
	PrincipalInvestigator string                 `json:"principal_investigator,omitempty"`
	Contact               map[string]interface{} `json:"contact,omitempty"`
}

type CreateStudyFormRequest struct {
	Name        string                 `json:"name"`
	Slug        string                 `json:"slug"`
	Description string                 `json:"description,omitempty"`
	Schema      map[string]interface{} `json:"schema"`
	Status      string                 `json:"status,omitempty"`
}

type CreateVisitTemplateRequest struct {
	Name            string   `json:"name"`
	VisitOrder      int      `json:"visit_order"`
	WindowStartDays *int     `json:"window_start_days,omitempty"`
	WindowEndDays   *int     `json:"window_end_days,omitempty"`
	Required        *bool    `json:"required,omitempty"`
	Forms           []string `json:"forms,omitempty"`
}

type EnrollSubjectRequest struct {
	SiteID           *uuid.UUID             `json:"site_id,omitempty"`
	SubjectCode      string                 `json:"subject_code"`
	RandomizationArm string                 `json:"randomization_arm,omitempty"`
	Demographics     map[string]interface{} `json:"demographics,omitempty"`
}

type ConsentSignatureRequest struct {
	ConsentVersionID uuid.UUID              `json:"consent_version_id"`
	SignedAt         time.Time              `json:"signed_at"`
	SignerName       string                 `json:"signer_name,omitempty"`
	Method           string                 `json:"method,omitempty"`
	IPAddress        string                 `json:"ip_address,omitempty"`
	Metadata         map[string]interface{} `json:"metadata,omitempty"`
}
