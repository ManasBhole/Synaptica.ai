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
