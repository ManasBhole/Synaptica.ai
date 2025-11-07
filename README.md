# Synaptica.ai - Healthcare Data Pipeline Platform

A comprehensive microservices architecture for healthcare data ingestion, processing, and analytics with AI/ML capabilities.

## Architecture Overview

The platform consists of multiple microservices that handle:
- **Upstream Data Ingestion**: Hospitals/EHR, Labs, Imaging, Wearables, Telehealth
- **Privacy & Normalization**: PHI detection, de-identification, schema normalization
- **Data Storage**: Lakehouse, RT OLAP, OLTP, Feature Store, Redis Cache
- **AI/Analytics**: LLM pipelines, cohort queries, model training & serving
- **Downstream Consumption**: Clinician apps, Pharma/CRO, Insurers, Internal ops

### Diagram

```mermaid
%% Synaptica.ai healthcare architecture
flowchart LR

  subgraph U[UPSTREAM PRODUCERS]
    H(Hospitals/EHR\nFHIR/HL7/ABDM)
    L(Labs/Diagnostics)
    I(Imaging Systems\nDICOM/meta)
    W(Wearables/IoT\nCGM/HR/HRV)
    T(Telehealth/Notes)
  end

  AGW(API Gateway\nOIDC, mTLS, RLS)
  ING(Ingestion Service\nFHIR, device JSON, file drops)
  BUS((Event Bus\nKafka/Pub/Sub))
  DLP(DLP & PHI Detector\nregex+NER+LLM)
  DEID(De-ID & Token Vault\nk/l-diversity checks)
  NORM(Schema Normalizer\nFHIR→canonical, codes)
  LINK(Record Linkage\ndeterministic + probabilistic)

  subgraph DATA[DATA PLANES]
    LAKE[(Lakehouse\nDelta/BigQuery)]
    CLICK[(RT OLAP\nClickHouse/Pinot)]
    PG[(OLTP Truth\nPostgres/AlloyDB)]
    FEAST[(Feature Store\nOffline+Online)]
    REDIS[(Online FS Cache\nRedis)]
  end

  LLM(LLM Pipelines\nNL→Cohort, Notes NLP, Code map)
  COHORT(Cohort/Query Engine\nDSL + verifier)
  TRAIN(Model Training\nBatch/AutoML)
  SERVE(Model Serving\nTriton/Vertex/TF Serving)
  CLEAN(Clean Room\nDP ε budgets, lineage)

  subgraph D[DOWNSTREAM CONSUMERS]
    CLIN(Clinician Apps/Alerts)
    PHAR(Pharma/CRO RWD/RWE)
    PAY(Insurers/TPA Analytics)
    OPS(Internal Ops & Dashboards)
  end

  H -->|"UPSTREAM ▶ FHIR/HL7, ABDM"| AGW
  L -->|"UPSTREAM ▶ lab feeds/CSV"| AGW
  I -->|"UPSTREAM ▶ DICOM meta"| AGW
  W -->|"UPSTREAM ▶ device JSON/SDK"| AGW
  T -->|"UPSTREAM ▶ notes, transcripts"| AGW
  AGW -->|"ingest calls"| ING
  ING -->|"UPSTREAM ▶ events"| BUS
  BUS -->|sanitize| DLP
  DLP -->|"UPSTREAM ▶ de-identify"| DEID
  DEID -->|"tokenized records"| NORM
  NORM -->|"canonical rows"| LINK
  LINK -->|"DOWNSTREAM ▶ immutable facts"| LAKE
  LINK -->|"DOWNSTREAM ▶ denormalized facts"| CLICK
  LINK -->|"consents/ids"| PG
  LAKE -->|"features build (batch)"| FEAST
  FEAST -->|"materialize hot"| REDIS
  LAKE -->|"NL schema/context"| LLM
  LAKE -->|"cohort scans"| COHORT
  CLICK -->|"sub-sec slicing"| COHORT
  TRAIN -->|"model.artifacts"| SERVE
  LAKE -->|"training data"| TRAIN
  FEAST -->|"feature views"| TRAIN
  REDIS -->|"features p95<10ms"| SERVE
  SERVE -->|"DOWNSTREAM ▶ risk scores"| CLIN
  COHORT -->|"DOWNSTREAM ▶ DP queries"| CLEAN
  CLEAN -->|"aggregates only"| PHAR
  CLEAN -->|"actuarial sets"| PAY
  COHORT -->|"ops queries/metrics"| OPS
```

## Services

### Core Services
- `api-gateway`: OIDC, mTLS, Row-Level Security gateway
- `ingestion-service`: Handles FHIR, device JSON, file drops
- `event-bus`: Kafka/Pub-Sub integration
- `dlp-service`: PHI detection using regex, NER, and LLM
- `deid-service`: De-identification and token vault with k/l-diversity
- `normalizer-service`: Schema normalization to canonical FHIR
- `linkage-service`: Record linkage (deterministic + probabilistic)
- `llm-service`: NL→Cohort, Notes NLP, Code mapping
- `cohort-service`: Cohort/Query Engine with DSL verifier
- `training-service`: Model training with AutoML
- `serving-service`: Model serving with Triton/Vertex/TF Serving
- `cleanroom-service`: Clean room with differential privacy
- `storage-service`: Emulates Delta/ClickHouse sinks for feature store

## Tech Stack

- **Language**: Go 1.21+
- `LAKEHOUSE_TABLE`, `OLAP_TABLE`: Postgres tables used to emulate Delta/ClickHouse sinks
- `FEATURE_OFFLINE_TABLE`, `FEATURE_ONLINE_PREFIX`
- `FEATURE_MATERIALIZE_CRON`: cadence for future background jobs (currently manual)
- `FEATURE_CACHE_TTL`: TTL for Redis hot features (default 5m)
- **Databases**: PostgreSQL (OLTP), ClickHouse (RT OLAP), Redis (Cache)
- **Message Queue**: Kafka
- **API Gateway**: Custom implementation with OIDC/mTLS
- **Storage**: Delta Lake / BigQuery (Lakehouse)
- **ML**: TensorFlow Serving, Triton

## Getting Started

### Prerequisites
- Go 1.21+
- Docker & Docker Compose
- PostgreSQL, Redis, Kafka (via Docker)

### Run Locally

```bash
# Start infrastructure
docker-compose up -d

# Apply database schema and seed data
psql postgresql://synaptica:synaptica123@localhost:5432/synaptica -f db/schema.sql
psql postgresql://synaptica:synaptica123@localhost:5432/synaptica -f db/seed/ingestion_requests.sql
psql postgresql://synaptica:synaptica123@localhost:5432/synaptica -f db/seed/normalized_records.sql
psql postgresql://synaptica:synaptica123@localhost:5432/synaptica -f db/seed/patient_linkages.sql
psql postgresql://synaptica:synaptica123@localhost:5432/synaptica -f db/seed/storage_facts.sql

# Run services (each in separate terminal)
cd cmd/api-gateway && go run main.go
cd cmd/ingestion-service && go run main.go
cd cmd/storage-service && go run main.go
# ... etc
```

### Environment Variables

Key variables for production hardening:

- `INGESTION_ALLOWED_SOURCES`: comma-separated list of onboarding sources (default: hospital,lab,imaging,wearable,telehealth)
- `INGESTION_KAFKA_TOPIC`: topic for main ingestion events (default: `upstream-events`)
- `INGESTION_DLQ_TOPIC`: dead-letter topic for failed publishes (`upstream-events-dlq`)
- `MAX_REQUEST_BODY_BYTES`: request payload limit (default 4MB)
- `GATEWAY_RATE_LIMIT_RPS` / `GATEWAY_RATE_LIMIT_BURST`: rate limiting controls
- `GATEWAY_REQUEST_TIMEOUT`: timeout for gateway to downstream services
- `INGESTION_STATUS_TTL`: retention window for ingestion audit records (default 7 days)
- `DLP_RULES_PATH`: YAML file listing PHI patterns (`configs/dlp_rules.yaml`)
- `DEID_TOKEN_SALT`: salt used for hashing tokens stored in the vault (change per environment)
- `TERMINOLOGY_CATALOG_PATH`: YAML vocabulary for SNOMED/LOINC/ICD mappings (`configs/terminology.yaml`)
- `NORMALIZER_KAFKA_TOPIC` / `NORMALIZER_DLQ_TOPIC`
- `NORMALIZER_ALLOWED_RESOURCES`: resource types transformer will accept
- `LINKAGE_KAFKA_TOPIC` / `LINKAGE_DLQ_TOPIC`
- `LINKAGE_DETERMINISTIC_KEYS`: fields used for exact matching (`patient_id,token_patient_id`)
- `LINKAGE_THRESHOLD`: probabilistic match threshold (default 0.85)

### Development

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build all services
./scripts/build.sh
```

## Configuration

Services use environment variables for configuration. See `.env.example` for details.

## License

Proprietary - Synaptica.ai

