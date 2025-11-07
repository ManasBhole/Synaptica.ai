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
