CREATE TABLE IF NOT EXISTS organizations (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT UNIQUE NOT NULL,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY,
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    email TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    role TEXT NOT NULL,
    password_hash TEXT NOT NULL,
    avatar_url TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_users_org ON users(organization_id);
CREATE INDEX IF NOT EXISTS idx_users_role ON users(role);

CREATE TABLE IF NOT EXISTS ingestion_requests (
    id UUID PRIMARY KEY,
    source TEXT NOT NULL,
    format TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL,
    error TEXT,
    retry_count INTEGER DEFAULT 0,
    last_attempt TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_ingestion_requests_status ON ingestion_requests(status);
CREATE INDEX IF NOT EXISTS idx_ingestion_requests_created_at ON ingestion_requests(created_at);

CREATE TABLE IF NOT EXISTS deid_token_vault (
    token TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS lakehouse_facts (
    id UUID PRIMARY KEY,
    master_id UUID,
    patient_id TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    canonical JSONB NOT NULL,
    codes JSONB,
    timestamp TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_lakehouse_patient ON lakehouse_facts(patient_id);
CREATE INDEX IF NOT EXISTS idx_lakehouse_master ON lakehouse_facts(master_id);

CREATE TABLE IF NOT EXISTS olap_rollups (
    id UUID PRIMARY KEY,
    master_id UUID,
    patient_id TEXT,
    metric TEXT NOT NULL,
    value JSONB NOT NULL,
    event_time TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_olap_metric ON olap_rollups(metric);
CREATE INDEX IF NOT EXISTS idx_olap_patient ON olap_rollups(patient_id);

CREATE TABLE IF NOT EXISTS feature_offline_store (
    id TEXT PRIMARY KEY,
    patient_id TEXT NOT NULL,
    features JSONB NOT NULL,
    version INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_feature_patient ON feature_offline_store(patient_id);

CREATE TABLE IF NOT EXISTS training_jobs (
    id UUID PRIMARY KEY,
    model_type TEXT NOT NULL,
    config JSONB,
    filters JSONB,
    status TEXT NOT NULL,
    metrics JSONB,
    artifact_path TEXT,
    error_message TEXT,
    promoted BOOLEAN NOT NULL DEFAULT FALSE,
    promoted_at TIMESTAMPTZ,
    promoted_by TEXT,
    promotion_notes TEXT,
    deployment_target TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

ALTER TABLE training_jobs ADD COLUMN IF NOT EXISTS promoted BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE training_jobs ADD COLUMN IF NOT EXISTS promoted_at TIMESTAMPTZ;
ALTER TABLE training_jobs ADD COLUMN IF NOT EXISTS promoted_by TEXT;
ALTER TABLE training_jobs ADD COLUMN IF NOT EXISTS promotion_notes TEXT;
ALTER TABLE training_jobs ADD COLUMN IF NOT EXISTS deployment_target TEXT;

CREATE INDEX IF NOT EXISTS idx_training_jobs_status ON training_jobs(status);
CREATE INDEX IF NOT EXISTS idx_training_jobs_model ON training_jobs(model_type);

CREATE TABLE IF NOT EXISTS prediction_logs (
    id UUID PRIMARY KEY,
    patient_id TEXT NOT NULL,
    model_name TEXT NOT NULL,
    request JSONB NOT NULL,
    response JSONB NOT NULL,
    latency_ms DOUBLE PRECISION NOT NULL,
    confidence DOUBLE PRECISION NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_prediction_logs_patient ON prediction_logs(patient_id);
CREATE INDEX IF NOT EXISTS idx_prediction_logs_created_at ON prediction_logs(created_at);

CREATE TABLE IF NOT EXISTS master_patients (
    id UUID PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS patient_linkages (
    id UUID PRIMARY KEY,
    master_id UUID REFERENCES master_patients(id),
    patient_id TEXT NOT NULL,
    deterministic_key TEXT,
    score DOUBLE PRECISION NOT NULL,
    method TEXT NOT NULL,
    attributes JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_patient_linkages_master ON patient_linkages(master_id);
CREATE INDEX IF NOT EXISTS idx_patient_linkages_patient ON patient_linkages(patient_id);
CREATE INDEX IF NOT EXISTS idx_patient_linkages_det_key ON patient_linkages(deterministic_key);

CREATE TABLE IF NOT EXISTS cohort_templates (
    id UUID PRIMARY KEY,
    tenant_id TEXT,
    name TEXT NOT NULL,
    description TEXT,
    dsl TEXT NOT NULL,
    tags TEXT[] DEFAULT ARRAY[]::TEXT[],
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cohort_templates_tenant ON cohort_templates(tenant_id);

CREATE TABLE IF NOT EXISTS cohort_materializations (
    id UUID PRIMARY KEY,
    cohort_id TEXT NOT NULL,
    tenant_id TEXT,
    dsl TEXT NOT NULL,
    fields JSONB,
    filters JSONB,
    status TEXT NOT NULL,
    result_count INTEGER DEFAULT 0,
    error_message TEXT,
    requested_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_cohort_materializations_status ON cohort_materializations(status);
CREATE INDEX IF NOT EXISTS idx_cohort_materializations_cohort ON cohort_materializations(cohort_id);

CREATE TABLE IF NOT EXISTS studies (
    id UUID PRIMARY KEY,
    code TEXT UNIQUE NOT NULL,
    name TEXT NOT NULL,
    phase TEXT,
    therapeutic_area TEXT,
    status TEXT NOT NULL DEFAULT 'draft',
    sponsor TEXT,
    protocol_summary JSONB,
    start_date DATE,
    end_date DATE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS study_sites (
    id UUID PRIMARY KEY,
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    site_code TEXT NOT NULL,
    name TEXT NOT NULL,
    country TEXT,
    principal_investigator TEXT,
    status TEXT NOT NULL DEFAULT 'planned',
    contact JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (study_id, site_code)
);

CREATE TABLE IF NOT EXISTS study_forms (
    id UUID PRIMARY KEY,
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    description TEXT,
    schema JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'draft',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (study_id, slug, version)
);

CREATE TABLE IF NOT EXISTS visit_templates (
    id UUID PRIMARY KEY,
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    visit_order INTEGER NOT NULL,
    window_start_days INTEGER,
    window_end_days INTEGER,
    required BOOLEAN NOT NULL DEFAULT TRUE,
    forms JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (study_id, visit_order)
);

CREATE TABLE IF NOT EXISTS subjects (
    id UUID PRIMARY KEY,
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    site_id UUID REFERENCES study_sites(id) ON DELETE SET NULL,
    subject_code TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'screening',
    randomization_arm TEXT,
    consented_at TIMESTAMPTZ,
    demographics JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (study_id, subject_code)
);

CREATE TABLE IF NOT EXISTS subject_visits (
    id UUID PRIMARY KEY,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    visit_template_id UUID NOT NULL REFERENCES visit_templates(id) ON DELETE CASCADE,
    scheduled_date DATE,
    actual_date DATE,
    status TEXT NOT NULL DEFAULT 'scheduled',
    forms JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS consent_versions (
    id UUID PRIMARY KEY,
    study_id UUID NOT NULL REFERENCES studies(id) ON DELETE CASCADE,
    version TEXT NOT NULL,
    title TEXT,
    summary TEXT,
    document_url TEXT,
    effective_at TIMESTAMPTZ NOT NULL,
    superseded_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (study_id, version)
);

CREATE TABLE IF NOT EXISTS consent_signatures (
    id UUID PRIMARY KEY,
    subject_id UUID NOT NULL REFERENCES subjects(id) ON DELETE CASCADE,
    consent_version_id UUID NOT NULL REFERENCES consent_versions(id) ON DELETE CASCADE,
    signed_at TIMESTAMPTZ NOT NULL,
    signer_name TEXT,
    method TEXT,
    ip_address TEXT,
    metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (subject_id, consent_version_id)
);

CREATE TABLE IF NOT EXISTS study_audit_logs (
    id BIGSERIAL PRIMARY KEY,
    study_id UUID REFERENCES studies(id) ON DELETE CASCADE,
    subject_id UUID REFERENCES subjects(id) ON DELETE SET NULL,
    actor TEXT NOT NULL,
    role TEXT,
    action TEXT NOT NULL,
    entity TEXT,
    entity_id TEXT,
    payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_studies_status ON studies(status);
CREATE INDEX IF NOT EXISTS idx_study_sites_status ON study_sites(status);
CREATE INDEX IF NOT EXISTS idx_subjects_status ON subjects(status);
CREATE INDEX IF NOT EXISTS idx_subject_visits_status ON subject_visits(status);
CREATE INDEX IF NOT EXISTS idx_consent_signatures_subject ON consent_signatures(subject_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_study ON study_audit_logs(study_id, created_at);
